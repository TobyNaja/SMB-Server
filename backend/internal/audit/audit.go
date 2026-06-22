package audit

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

const maxEntries = 10000

// Entry mirrors the Python AuditService log record format.
type Entry struct {
	Timestamp    string                 `json:"timestamp"`
	Action       string                 `json:"action"`
	Actor        string                 `json:"actor"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Status       string                 `json:"status"`
	Details      map[string]interface{} `json:"details"`
	ClientIP     string                 `json:"client_ip,omitempty"`
}

// Service manages the append-only JSON audit log.
type Service struct {
	logPath string
}

func NewService(logPath string) *Service {
	return &Service{logPath: logPath}
}

func (s *Service) ensureFile() error {
	if err := os.MkdirAll(filepath.Dir(s.logPath), 0o755); err != nil {
		return err
	}
	if _, err := os.Stat(s.logPath); os.IsNotExist(err) {
		return os.WriteFile(s.logPath, []byte("[]"), 0o640)
	}
	return nil
}

func (s *Service) load() ([]Entry, error) {
	_ = s.ensureFile()
	data, err := os.ReadFile(s.logPath)
	if err != nil {
		return nil, err
	}
	var entries []Entry
	if err := json.Unmarshal(data, &entries); err != nil {
		return []Entry{}, nil
	}
	return entries, nil
}

func (s *Service) save(entries []Entry) error {
	data, err := json.MarshalIndent(entries, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.logPath, data, 0o640)
}

// Log appends an audit entry (best effort — never returns an error to callers).
func (s *Service) Log(action, actor, resourceType, resourceName, status string, details map[string]interface{}, clientIP string) {
	if details == nil {
		details = map[string]interface{}{}
	}
	entry := Entry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		Action:       action,
		Actor:        actor,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Status:       status,
		Details:      details,
		ClientIP:     clientIP,
	}
	entries, _ := s.load()
	entries = append(entries, entry)
	if len(entries) > maxEntries {
		entries = entries[len(entries)-maxEntries:]
	}
	_ = s.save(entries)
}

// GetLogs returns up to limit entries newest-first, with optional action/actor filters.
func (s *Service) GetLogs(limit int, action, actor string) ([]Entry, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	entries, err := s.load()
	if err != nil {
		return nil, err
	}

	// Filter
	var filtered []Entry
	for _, e := range entries {
		if action != "" && e.Action != action {
			continue
		}
		if actor != "" && e.Actor != actor {
			continue
		}
		filtered = append(filtered, e)
	}

	// Return newest-first, capped at limit
	if len(filtered) > limit {
		filtered = filtered[len(filtered)-limit:]
	}
	// Reverse
	for i, j := 0, len(filtered)-1; i < j; i, j = i+1, j-1 {
		filtered[i], filtered[j] = filtered[j], filtered[i]
	}
	return filtered, nil
}
