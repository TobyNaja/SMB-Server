package port

import "smb-server/backend/internal/domain"

// AuditLog abstracts audit entry persistence.
type AuditLog interface {
	Append(entry domain.AuditEntry)
	Query(limit int, action, actor string) ([]domain.AuditEntry, error)
}
