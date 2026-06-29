package port

import "smb-server/backend/internal/domain"

// AuthStore abstracts admin credential persistence.
type AuthStore interface {
	Verify(username, password string) bool
	Add(username, password string) error
	Delete(username string) error
	List() ([]domain.Admin, error)
	ChangePassword(username, newPassword string) error
}
