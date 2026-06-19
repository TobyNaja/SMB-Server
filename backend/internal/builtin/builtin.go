package builtin

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"smb-server/backend/internal/samba"
)

// GroupMeta holds display metadata for a BUILTIN group.
type GroupMeta struct {
	Description string `json:"description"`
	Color       string `json:"color"`
	Icon        string `json:"icon"`
}

// GroupResult is the full API response for one BUILTIN group.
type GroupResult struct {
	Name        string   `json:"name"`
	FullName    string   `json:"full_name"`
	Description string   `json:"description"`
	Color       string   `json:"color"`
	Icon        string   `json:"icon"`
	Members     []string `json:"members"`
	MemberCount int      `json:"member_count"`
}

// groups is the canonical list of BUILTIN groups mirroring the Python definition.
var groups = map[string]GroupMeta{
	"Administrators":   {Description: "Full control over Samba server — เข้าได้ทุก Share อัตโนมัติ", Color: "danger", Icon: "shield-fill-check"},
	"Users":            {Description: "Standard users — user ทั่วไปที่ login ได้", Color: "primary", Icon: "people-fill"},
	"Guests":           {Description: "Guest access — เข้าได้โดยไม่ต้อง login", Color: "secondary", Icon: "person-dash-fill"},
	"Power Users":      {Description: "ระหว่าง Admin กับ User — สิทธิ์พิเศษบางอย่าง", Color: "warning", Icon: "lightning-fill"},
	"Backup Operators": {Description: "Backup access — สำหรับทำ Backup", Color: "info", Icon: "archive-fill"},
	"Print Operators":  {Description: "Printer management — จัดการ Printer", Color: "dark", Icon: "printer-fill"},
}

// groupOrder preserves display order.
var groupOrder = []string{
	"Administrators", "Users", "Guests",
	"Power Users", "Backup Operators", "Print Operators",
}

// Service manages BUILTIN group membership persistence and Samba sync.
type Service struct {
	exec      samba.Executor
	storePath string
}

func NewService(exec samba.Executor, storePath string) *Service {
	return &Service{exec: exec, storePath: storePath}
}

func (s *Service) load() (map[string][]string, error) {
	data, err := os.ReadFile(s.storePath)
	if err != nil {
		if os.IsNotExist(err) {
			store := map[string][]string{}
			for name := range groups {
				store[name] = []string{}
			}
			return store, nil
		}
		return nil, err
	}
	var store map[string][]string
	if err := json.Unmarshal(data, &store); err != nil {
		return nil, err
	}
	// Ensure all groups present
	for name := range groups {
		if store[name] == nil {
			store[name] = []string{}
		}
	}
	return store, nil
}

func (s *Service) save(store map[string][]string) error {
	if err := os.MkdirAll(filepath.Dir(s.storePath), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(store, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.storePath, data, 0o644)
}

func (s *Service) applyToSamba(groupName, username, action string) {
	verb := "addmem"
	if action == "del" {
		verb = "delmem"
	}
	cmd := fmt.Sprintf("net sam %s 'BUILTIN\\%s' '%s' 2>&1 || true", verb, groupName, username)
	s.exec.Execute(cmd)
}

// ListGroups returns all BUILTIN groups with current membership.
func (s *Service) ListGroups() ([]GroupResult, error) {
	store, err := s.load()
	if err != nil {
		return nil, err
	}
	result := make([]GroupResult, 0, len(groupOrder))
	for _, name := range groupOrder {
		meta := groups[name]
		members := store[name]
		if members == nil {
			members = []string{}
		}
		result = append(result, GroupResult{
			Name:        name,
			FullName:    `BUILTIN\` + name,
			Description: meta.Description,
			Color:       meta.Color,
			Icon:        meta.Icon,
			Members:     members,
			MemberCount: len(members),
		})
	}
	return result, nil
}

// GetGroup returns one group's details or an error if the group doesn't exist.
func (s *Service) GetGroup(name string) (*GroupResult, error) {
	meta, ok := groups[name]
	if !ok {
		return nil, errors.New("builtin group '" + name + "' not found")
	}
	store, err := s.load()
	if err != nil {
		return nil, err
	}
	members := store[name]
	if members == nil {
		members = []string{}
	}
	return &GroupResult{
		Name:        name,
		FullName:    `BUILTIN\` + name,
		Description: meta.Description,
		Color:       meta.Color,
		Icon:        meta.Icon,
		Members:     members,
		MemberCount: len(members),
	}, nil
}

// AddMember adds username to the group and syncs to Samba. Returns updated member list.
func (s *Service) AddMember(groupName, username string) ([]string, error) {
	if _, ok := groups[groupName]; !ok {
		return nil, errors.New("builtin group '" + groupName + "' not found")
	}
	if username == "" {
		return nil, errors.New("username is required")
	}
	store, err := s.load()
	if err != nil {
		return nil, err
	}
	for _, m := range store[groupName] {
		if m == username {
			return nil, errors.New("'" + username + "' is already in " + groupName)
		}
	}
	store[groupName] = append(store[groupName], username)
	if err := s.save(store); err != nil {
		return nil, err
	}
	s.applyToSamba(groupName, username, "add")
	return store[groupName], nil
}

// RemoveMember removes username from the group and syncs to Samba. Returns updated list.
func (s *Service) RemoveMember(groupName, username string) ([]string, error) {
	if _, ok := groups[groupName]; !ok {
		return nil, errors.New("builtin group '" + groupName + "' not found")
	}
	store, err := s.load()
	if err != nil {
		return nil, err
	}
	members := store[groupName]
	filtered := make([]string, 0, len(members))
	found := false
	for _, m := range members {
		if m == username {
			found = true
			continue
		}
		filtered = append(filtered, m)
	}
	if !found {
		return nil, errors.New("'" + username + "' is not in " + groupName)
	}
	store[groupName] = filtered
	if err := s.save(store); err != nil {
		return nil, err
	}
	s.applyToSamba(groupName, username, "del")
	return filtered, nil
}
