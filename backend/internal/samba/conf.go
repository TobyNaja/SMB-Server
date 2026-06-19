package samba

import (
	"bufio"
	"fmt"
	"os"
	"strings"
)

var systemSections = map[string]bool{
	"global": true, "homes": true, "printers": true,
	"print$": true, "netlogon": true, "sysvol": true,
}

var userListFields = map[string]bool{
	"valid users": true, "invalid users": true,
	"read list": true, "write list": true, "admin users": true,
}

// ShareData represents a parsed share's key-value settings.
type ShareData map[string]string

// ShareInfo is the API-friendly representation of a share.
type ShareInfo struct {
	Name          string   `json:"name"`
	Path          string   `json:"path"`
	Comment       string   `json:"comment"`
	Browseable    bool     `json:"browseable"`
	ReadOnly      bool     `json:"read_only"`
	GuestOK       bool     `json:"guest_ok"`
	ABSE          bool     `json:"abse"`
	ValidUsers    []string `json:"valid_users"`
	WriteList     []string `json:"write_list"`
	ReadList      []string `json:"read_list"`
	AdminUsers    []string `json:"admin_users"`
	InvalidUsers  []string `json:"invalid_users"`
	CreateMask    string   `json:"create_mask"`
	DirectoryMask string   `json:"directory_mask"`
}

// SmbConfParser reads and writes shares.conf exclusively.
type SmbConfParser struct {
	sharesPath string
	globalPath string
	sections   map[string]ShareData
}

// NewSmbConfParser creates a parser and immediately reads shares.conf.
func NewSmbConfParser(sharesPath, globalPath string) *SmbConfParser {
	p := &SmbConfParser{
		sharesPath: sharesPath,
		globalPath: globalPath,
		sections:   map[string]ShareData{},
	}
	p.load()
	return p
}

func (p *SmbConfParser) load() {
	p.sections = map[string]ShareData{}
	parsed := p.parseFile(p.sharesPath)
	for k, v := range parsed {
		if !systemSections[k] {
			p.sections[k] = v
		}
	}
}

func (p *SmbConfParser) parseFile(path string) map[string]ShareData {
	result := map[string]ShareData{}
	f, err := os.Open(path)
	if err != nil {
		return result
	}
	defer f.Close()

	var section string
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, ";") {
			continue
		}
		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section = strings.TrimSpace(line[1 : len(line)-1])
			result[section] = ShareData{}
			continue
		}
		if section != "" {
			if idx := strings.Index(line, "="); idx > 0 {
				key := strings.TrimSpace(line[:idx])
				val := strings.TrimSpace(line[idx+1:])
				result[section][key] = val
			}
		}
	}
	return result
}

func (p *SmbConfParser) save() error {
	dir := p.sharesPath[:strings.LastIndex(p.sharesPath, "/")]
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	f, err := os.Create(p.sharesPath)
	if err != nil {
		return err
	}
	defer f.Close()

	header := "# ============================================\n" +
		"# SMB Shares - Managed by SMB Manager Web UI\n" +
		"# DO NOT EDIT MANUALLY\n" +
		"# ============================================\n\n"
	f.WriteString(header)

	for name, data := range p.sections {
		if systemSections[name] {
			continue
		}
		fmt.Fprintf(f, "[%s]\n", name)
		for k, v := range data {
			fmt.Fprintf(f, "    %s = %s\n", k, v)
		}
		f.WriteString("\n")
	}
	return nil
}

// GetGlobal reads read-only global settings from smb.conf.
func (p *SmbConfParser) GetGlobal() map[string]interface{} {
	global := p.parseFile(p.globalPath)
	g := global["global"]
	if g == nil {
		g = ShareData{}
	}
	abse := strings.ToLower(g["access based share enum"]) == "yes"
	return map[string]interface{}{
		"workgroup":     g["workgroup"],
		"realm":         g["realm"],
		"security":      g["security"],
		"netbios_name":  g["netbios name"],
		"server_string": g["server string"],
		"abse":          abse,
	}
}

// ShareExists returns whether the named share exists.
func (p *SmbConfParser) ShareExists(name string) (bool, error) {
	_, ok := p.sections[name]
	return ok, nil
}

// GetShare returns a parsed ShareInfo or nil if not found.
func (p *SmbConfParser) GetShare(name string) (*ShareInfo, error) {
	d, ok := p.sections[name]
	if !ok {
		return nil, nil
	}
	return &ShareInfo{
		Name:          name,
		Path:          d["path"],
		Comment:       d["comment"],
		Browseable:    strings.ToLower(d["browseable"]) != "no",
		ReadOnly:      strings.ToLower(d["read only"]) == "yes",
		GuestOK:       strings.ToLower(d["guest ok"]) == "yes",
		ABSE:          strings.ToLower(d["access based share enum"]) == "yes",
		ValidUsers:    parseUserList(d["valid users"]),
		WriteList:     parseUserList(d["write list"]),
		ReadList:      parseUserList(d["read list"]),
		AdminUsers:    parseUserList(d["admin users"]),
		InvalidUsers:  parseUserList(d["invalid users"]),
		CreateMask:    d["create mask"],
		DirectoryMask: d["directory mask"],
	}, nil
}

// GetAllShares returns all shares as ShareInfo slices.
func (p *SmbConfParser) GetAllShares() ([]*ShareInfo, error) {
	result := make([]*ShareInfo, 0, len(p.sections))
	for name := range p.sections {
		s, _ := p.GetShare(name)
		if s != nil {
			result = append(result, s)
		}
	}
	return result, nil
}

// CreateShare adds a new share with defaults. Returns false if it already exists.
func (p *SmbConfParser) CreateShare(name, path, comment string) bool {
	if _, exists := p.sections[name]; exists {
		return false
	}
	if comment == "" {
		comment = name + " share"
	}
	p.sections[name] = ShareData{
		"comment":                  comment,
		"path":                     path,
		"browseable":               "yes",
		"read only":                "yes",
		"guest ok":                 "no",
		"access based share enum":  "no",
		"create mask":              "0775",
		"directory mask":           "0775",
		"force create mode":        "0777",
		"force directory mode":     "0777",
		"valid users":              "",
	}
	_ = p.save()
	return true
}

var keyMap = map[string]string{
	"read_only": "read only",
	"guest_ok":  "guest ok",
	"abse":      "access based share enum",
}

var boolFields = map[string]bool{
	"browseable": true, "read only": true,
	"guest ok": true, "access based share enum": true,
}

// UpdateShare applies the updates map (snake_case or raw smb keys) to a share and saves.
func (p *SmbConfParser) UpdateShare(name string, updates map[string]interface{}) bool {
	d, ok := p.sections[name]
	if !ok {
		return false
	}
	for k, v := range updates {
		if v == nil {
			continue
		}
		realKey := k
		if mapped, ok := keyMap[k]; ok {
			realKey = mapped
		}
		if userListFields[realKey] {
			continue // use Set* methods for user lists
		}
		if boolFields[realKey] {
			if b, ok := v.(bool); ok {
				if b {
					d[realKey] = "yes"
				} else {
					d[realKey] = "no"
				}
				continue
			}
		}
		d[realKey] = fmt.Sprintf("%v", v)
	}
	p.sections[name] = d
	_ = p.save()
	return true
}

// DeleteShare removes the share and saves.
func (p *SmbConfParser) DeleteShare(name string) bool {
	if _, ok := p.sections[name]; !ok {
		return false
	}
	delete(p.sections, name)
	_ = p.save()
	return true
}

// SetShareABSE toggles ABSE for a single share.
func (p *SmbConfParser) SetShareABSE(name string, enabled bool) bool {
	d, ok := p.sections[name]
	if !ok {
		return false
	}
	if enabled {
		d["access based share enum"] = "yes"
	} else {
		d["access based share enum"] = "no"
	}
	p.sections[name] = d
	_ = p.save()
	return true
}

// setUserList is the generic setter that runs syncPerms before saving.
func (p *SmbConfParser) setUserList(name, field string, users []string) bool {
	d, ok := p.sections[name]
	if !ok {
		return false
	}
	d[field] = formatUserList(sanitizeUsers(users))
	p.sections[name] = d
	p.syncPerms(name)
	_ = p.save()
	return true
}

func (p *SmbConfParser) syncPerms(name string) {
	d, ok := p.sections[name]
	if !ok {
		return
	}
	synced := SyncPermissions(SharePerms{
		Valid:   parseUserList(d["valid users"]),
		Write:   parseUserList(d["write list"]),
		Read:    parseUserList(d["read list"]),
		Admin:   parseUserList(d["admin users"]),
		Invalid: parseUserList(d["invalid users"]),
	})
	d["valid users"] = formatUserList(synced.Valid)
	d["write list"] = formatUserList(synced.Write)
	d["read list"] = formatUserList(synced.Read)
	d["admin users"] = formatUserList(synced.Admin)
	d["invalid users"] = formatUserList(synced.Invalid)
	p.sections[name] = d
}

func (p *SmbConfParser) SetValidUsers(name string, users []string) bool {
	return p.setUserList(name, "valid users", users)
}
func (p *SmbConfParser) SetWriteList(name string, users []string) bool {
	return p.setUserList(name, "write list", users)
}
func (p *SmbConfParser) SetReadList(name string, users []string) bool {
	return p.setUserList(name, "read list", users)
}
func (p *SmbConfParser) SetAdminUsers(name string, users []string) bool {
	return p.setUserList(name, "admin users", users)
}
func (p *SmbConfParser) SetInvalidUsers(name string, users []string) bool {
	return p.setUserList(name, "invalid users", users)
}
