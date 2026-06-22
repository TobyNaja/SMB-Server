package port

import "smb-server/backend/internal/domain"

// ShareStore abstracts read/write of shares.conf.
type ShareStore interface {
	GetShare(name string) (*domain.Share, error)
	GetAllShares() ([]*domain.Share, error)
	CreateShare(name, path, comment string) error
	UpdateShare(name string, updates map[string]interface{}) error
	DeleteShare(name string) error
	SetShareABSE(name string, enabled bool) error
	SetUserList(name, field string, users []string) error
	ShareExists(name string) bool
	GetGlobal() map[string]interface{}
}
