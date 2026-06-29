package fake

import (
    "smb-server/backend/internal/domain"
    "smb-server/backend/internal/port"
)

type AuditLog struct {
    Entries []domain.AuditEntry
}

func NewAuditLog() *AuditLog { return &AuditLog{} }

func (l *AuditLog) Append(e domain.AuditEntry) {
    l.Entries = append(l.Entries, e)
}

func (l *AuditLog) Query(limit int, action, actor string) ([]domain.AuditEntry, error) {
    var out []domain.AuditEntry
    for _, e := range l.Entries {
        if action != "" && e.Action != action { continue }
        if actor != "" && e.Actor != actor { continue }
        out = append(out, e)
    }
    if limit > 0 && len(out) > limit { out = out[len(out)-limit:] }
    return out, nil
}

var _ port.AuditLog = (*AuditLog)(nil)
