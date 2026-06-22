package fake

import (
    "smb-server/backend/internal/domain"
    "smb-server/backend/internal/port"
)

type ShareStore struct {
    shares map[string]*domain.Share
}

func NewShareStore() *ShareStore {
    return &ShareStore{shares: map[string]*domain.Share{}}
}

func (s *ShareStore) ShareExists(name string) bool {
    _, ok := s.shares[name]
    return ok
}

func (s *ShareStore) CreateShare(name, path, comment string) error {
    if s.ShareExists(name) { return domain.ErrAlreadyExists }
    s.shares[name] = &domain.Share{Name: name, Path: path, Comment: comment,
        Browseable: true, ReadOnly: true, CreateMask: "0775", DirectoryMask: "0775"}
    return nil
}

func (s *ShareStore) GetShare(name string) (*domain.Share, error) {
    sh, ok := s.shares[name]
    if !ok { return nil, domain.ErrNotFound }
    cp := *sh
    return &cp, nil
}

func (s *ShareStore) GetAllShares() ([]*domain.Share, error) {
    out := make([]*domain.Share, 0, len(s.shares))
    for _, sh := range s.shares { cp := *sh; out = append(out, &cp) }
    return out, nil
}

func (s *ShareStore) UpdateShare(name string, updates map[string]interface{}) error {
    if !s.ShareExists(name) { return domain.ErrNotFound }
    // ponytail: simple type switch for test doubles — production adapter does full mapping
    sh := s.shares[name]
    for k, v := range updates {
        switch k {
        case "browseable": if b, ok := v.(bool); ok { sh.Browseable = b }
        case "read_only":  if b, ok := v.(bool); ok { sh.ReadOnly = b }
        case "guest_ok":   if b, ok := v.(bool); ok { sh.GuestOK = b }
        case "abse":       if b, ok := v.(bool); ok { sh.ABSE = b }
        case "comment":    if str, ok := v.(string); ok { sh.Comment = str }
        }
    }
    return nil
}

func (s *ShareStore) DeleteShare(name string) error {
    if !s.ShareExists(name) { return domain.ErrNotFound }
    delete(s.shares, name)
    return nil
}

func (s *ShareStore) SetShareABSE(name string, enabled bool) error {
    if !s.ShareExists(name) { return domain.ErrNotFound }
    s.shares[name].ABSE = enabled
    return nil
}

func (s *ShareStore) SetUserList(name, field string, users []string) error {
    if !s.ShareExists(name) { return domain.ErrNotFound }
    sh := s.shares[name]
    switch field {
    case "valid users":   sh.ValidUsers = users
    case "write list":    sh.WriteList = users
    case "read list":     sh.ReadList = users
    case "admin users":   sh.AdminUsers = users
    case "invalid users": sh.InvalidUsers = users
    }
    return nil
}

func (s *ShareStore) GetGlobal() map[string]interface{} {
    return map[string]interface{}{"abse": false, "workgroup": "TEST"}
}

// Ensure interface compliance.
var _ port.ShareStore = (*ShareStore)(nil)
