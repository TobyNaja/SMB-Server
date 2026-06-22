package domain

import "time"

type AuditEntry struct {
	Timestamp    string                 `json:"timestamp"`
	Action       string                 `json:"action"`
	Actor        string                 `json:"actor"`
	ResourceType string                 `json:"resource_type"`
	ResourceName string                 `json:"resource_name"`
	Status       string                 `json:"status"`
	Details      map[string]interface{} `json:"details"`
	ClientIP     string                 `json:"client_ip,omitempty"`
}

func NewAuditEntry(action, actor, resourceType, resourceName, status, clientIP string) AuditEntry {
	return AuditEntry{
		Timestamp:    time.Now().UTC().Format(time.RFC3339Nano),
		Action:       action,
		Actor:        actor,
		ResourceType: resourceType,
		ResourceName: resourceName,
		Status:       status,
		Details:      map[string]interface{}{},
		ClientIP:     clientIP,
	}
}
