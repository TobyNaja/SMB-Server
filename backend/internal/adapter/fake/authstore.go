package fake

import (
    "smb-server/backend/internal/domain"
    "smb-server/backend/internal/port"
)

type AuthStore struct {
    admins map[string]string // username → plaintext (test only)
}

func NewAuthStore() *AuthStore { return &AuthStore{admins: map[string]string{}} }

func (a *AuthStore) Verify(username, password string) bool {
    p, ok := a.admins[username]
    return ok && p == password
}

func (a *AuthStore) Add(username, password string) error {
    if _, exists := a.admins[username]; exists { return domain.ErrAlreadyExists }
    a.admins[username] = password
    return nil
}

func (a *AuthStore) Delete(username string) error {
    if _, exists := a.admins[username]; !exists { return domain.ErrNotFound }
    if len(a.admins) == 1 { return domain.ErrLastAdmin }
    delete(a.admins, username)
    return nil
}

func (a *AuthStore) List() ([]domain.Admin, error) {
    out := make([]domain.Admin, 0, len(a.admins))
    for u := range a.admins { out = append(out, domain.Admin{Username: u}) }
    return out, nil
}

func (a *AuthStore) ChangePassword(username, newPassword string) error {
    if _, exists := a.admins[username]; !exists { return domain.ErrNotFound }
    a.admins[username] = newPassword
    return nil
}

var _ port.AuthStore = (*AuthStore)(nil)
