package ldap

import (
	"fmt"
	"regexp"
	"smb-server/backend/internal/samba"
	"strings"
)

// Config carries the LDAP connection settings from config.Config.
type Config struct {
	Server string
	Port   int
	BaseDN string
	BindDN string
	BindPW string
	Domain string
}

// UserResult is the API-friendly representation of an AD user.
type UserResult struct {
	Username    string `json:"username"`
	DisplayName string `json:"display_name"`
	Email       string `json:"email"`
	Department  string `json:"department"`
	Title       string `json:"title"`
	OU          string `json:"ou"`
	Source      string `json:"source"`
}

// GroupResult is the API-friendly representation of an AD group.
type GroupResult struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	OU          string `json:"ou"`
	SMBName     string `json:"smb_name"`
	Source      string `json:"source"`
}

// OUResult is a static OU descriptor.
type OUResult struct {
	Name        string `json:"name"`
	DN          string `json:"dn"`
	Description string `json:"description"`
}

// ConnectionStatus is the /api/ad/status response shape.
type ConnectionStatus struct {
	LDAPServer string `json:"ldap_server"`
	Domain     string `json:"domain"`
	BaseDN     string `json:"base_dn"`
	BindDN     string `json:"bind_dn"`
	Connected  bool   `json:"connected"`
	Error      string `json:"error,omitempty"`
}

// Service executes ldapsearch inside the samba container and parses the LDIF output.
type Service struct {
	exec samba.Executor
	cfg  Config
}

func NewService(exec samba.Executor, cfg Config) *Service {
	return &Service{exec: exec, cfg: cfg}
}

func (s *Service) Domain() string { return s.cfg.Domain }

func (s *Service) ldapsearch(base, scope, filter string, attrs []string) string {
	attrsStr := strings.Join(attrs, " ")
	cmd := fmt.Sprintf(
		"ldapsearch -H ldap://%s:%d -D '%s' -w '%s' -b '%s' -s %s '%s' %s 2>&1",
		s.cfg.Server, s.cfg.Port,
		s.cfg.BindDN, s.cfg.BindPW,
		base, scope, filter, attrsStr,
	)
	return s.exec.Execute(cmd).Output
}

var ouRegex = regexp.MustCompile(`(?i)OU=([^,]+)`)

func extractOU(dn string) string {
	if m := ouRegex.FindStringSubmatch(dn); len(m) > 1 {
		return m[1]
	}
	return "Users"
}

func domainShort(domain string) string {
	parts := strings.SplitN(domain, ".", 2)
	return strings.ToUpper(parts[0])
}

// SearchUsers searches AD for users matching query (sAMAccountName, cn, or mail).
func (s *Service) SearchUsers(query, ou string, limit int) ([]UserResult, error) {
	base := s.cfg.BaseDN
	if ou != "" {
		base = ou + "," + s.cfg.BaseDN
	}

	var filter string
	if query != "" {
		filter = fmt.Sprintf(
			"(&(objectClass=user)(objectCategory=person)(!(userAccountControl:1.2.840.113556.1.4.803:=2))(|(sAMAccountName=*%s*)(cn=*%s*)(mail=*%s*)))",
			query, query, query,
		)
	} else {
		filter = "(&(objectClass=user)(objectCategory=person)(!(userAccountControl:1.2.840.113556.1.4.803:=2)))"
	}

	output := s.ldapsearch(base, "sub", filter, []string{
		"sAMAccountName", "cn", "mail", "department", "title", "distinguishedName",
	})

	entries := ParseLDIF(output)
	users := make([]UserResult, 0)
	short := domainShort(s.cfg.Domain)

	for _, e := range entries {
		sam := GetString(e, "samaccountname")
		if sam == "" || strings.HasSuffix(sam, "$") {
			continue
		}
		dn := GetString(e, "dn")
		users = append(users, UserResult{
			Username:    short + `\` + sam,
			DisplayName: GetString(e, "cn"),
			Email:       GetString(e, "mail"),
			Department:  GetString(e, "department"),
			Title:       GetString(e, "title"),
			OU:          extractOU(dn),
			Source:      "ad",
		})
		if len(users) >= limit {
			break
		}
	}
	return users, nil
}

// GetUser looks up a single AD user by sAMAccountName.
func (s *Service) GetUser(username string) (*UserResult, error) {
	output := s.ldapsearch(s.cfg.BaseDN, "sub",
		fmt.Sprintf("(&(objectClass=user)(sAMAccountName=%s))", username),
		[]string{"sAMAccountName", "cn", "mail", "department", "title", "memberOf"},
	)
	entries := ParseLDIF(output)
	if len(entries) == 0 {
		return nil, nil
	}
	e := entries[0]
	return &UserResult{
		Username:    GetString(e, "samaccountname"),
		DisplayName: GetString(e, "cn"),
		Email:       GetString(e, "mail"),
		Department:  GetString(e, "department"),
		Title:       GetString(e, "title"),
		Source:      "ad",
	}, nil
}

// SearchGroups searches AD groups by cn.
func (s *Service) SearchGroups(query string, limit int) ([]GroupResult, error) {
	var filter string
	if query != "" {
		filter = fmt.Sprintf("(&(objectClass=group)(cn=*%s*))", query)
	} else {
		filter = "(objectClass=group)"
	}

	output := s.ldapsearch(s.cfg.BaseDN, "sub", filter,
		[]string{"cn", "description", "distinguishedName"},
	)

	entries := ParseLDIF(output)
	groups := make([]GroupResult, 0)

	for _, e := range entries {
		cn := GetString(e, "cn")
		if cn == "" {
			continue
		}
		desc := GetString(e, "description")
		if len(desc) > 80 {
			desc = desc[:80]
		}
		dn := GetString(e, "dn")
		groups = append(groups, GroupResult{
			Name:        cn,
			Description: desc,
			OU:          extractOU(dn),
			SMBName:     "@" + cn,
			Source:      "ad",
		})
		if len(groups) >= limit {
			break
		}
	}
	return groups, nil
}

// TestConnection probes the LDAP server with a base-scope search.
func (s *Service) TestConnection() ConnectionStatus {
	output := s.ldapsearch(s.cfg.BaseDN, "base", "(objectClass=*)", []string{"dn"})
	status := ConnectionStatus{
		LDAPServer: s.cfg.Server,
		Domain:     s.cfg.Domain,
		BaseDN:     s.cfg.BaseDN,
		BindDN:     s.cfg.BindDN,
	}
	switch {
	case strings.Contains(output, "Invalid credentials"):
		status.Error = "Invalid credentials"
	case strings.Contains(output, "Can't contact"):
		status.Error = "Cannot connect to LDAP server"
	case strings.Contains(strings.ToLower(output), "dn:"):
		status.Connected = true
	default:
		if len(output) > 200 {
			output = output[:200]
		}
		status.Error = output
	}
	return status
}

// ListOUs returns the static OU list for IT.KMITL.
func (s *Service) ListOUs() []OUResult {
	return []OUResult{
		{Name: "Lecturer", DN: "OU=Lecturer," + s.cfg.BaseDN, Description: "อาจารย์"},
		{Name: "Staff", DN: "OU=Staff," + s.cfg.BaseDN, Description: "เจ้าหน้าที่"},
		{Name: "Student", DN: "OU=Student," + s.cfg.BaseDN, Description: "นักศึกษา"},
		{Name: "Group", DN: "OU=Group," + s.cfg.BaseDN, Description: "Groups"},
		{Name: "Service Accounts", DN: "OU=Service Accounts," + s.cfg.BaseDN, Description: "Service Accounts"},
	}
}
