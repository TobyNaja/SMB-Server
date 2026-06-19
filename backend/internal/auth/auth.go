package auth

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"golang.org/x/crypto/bcrypt"
)

// Admin represents a single admin account entry in the .admin store.
type Admin struct {
	Username       string     `json:"username"`
	HashedPassword string     `json:"hashed_password"`
	CreatedAt      time.Time  `json:"created_at"`
	LastLogin      *time.Time `json:"last_login,omitempty"`
}

// AdminPublic is Admin without hashed_password, safe to return in API responses.
type AdminPublic struct {
	Username  string     `json:"username"`
	CreatedAt time.Time  `json:"created_at"`
	LastLogin *time.Time `json:"last_login,omitempty"`
}

// Service handles bcrypt hashing, JWT signing, and the multi-admin .admin store.
type Service struct {
	secretKey   string
	tokenExpiry time.Duration
	adminFile   string
}

func New(secretKey, adminFile string, tokenExpiryMinutes int) *Service {
	return &Service{
		secretKey:   secretKey,
		tokenExpiry: time.Duration(tokenExpiryMinutes) * time.Minute,
		adminFile:   adminFile,
	}
}

// HashPassword hashes with bcrypt cost 12.
func (s *Service) HashPassword(password string) (string, error) {
	b, err := bcrypt.GenerateFromPassword([]byte(password), 12)
	return string(b), err
}

// VerifyPassword checks a plaintext password against a bcrypt hash.
func (s *Service) VerifyPassword(password, hash string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// CreateToken creates a signed JWT HS256 token for the given username.
// Returns the token string and the expiry duration in seconds.
func (s *Service) CreateToken(username string) (string, int64, error) {
	exp := time.Now().Add(s.tokenExpiry)
	claims := jwt.MapClaims{
		"username": username,
		"exp":      exp.Unix(),
		"iat":      time.Now().Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString([]byte(s.secretKey))
	return signed, int64(s.tokenExpiry.Seconds()), err
}

// VerifyToken parses and validates a JWT, returning the embedded username.
func (s *Service) VerifyToken(tokenStr string) (string, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, errors.New("unexpected signing method")
		}
		return []byte(s.secretKey), nil
	}, jwt.WithValidMethods([]string{"HS256"}))
	if err != nil || !token.Valid {
		return "", errors.New("invalid or expired token")
	}
	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return "", errors.New("invalid claims")
	}
	username, _ := claims["username"].(string)
	if username == "" {
		return "", errors.New("missing username in token")
	}
	return username, nil
}

// loadAdmins reads the .admin file. Auto-upgrades old single-object format to list.
func (s *Service) loadAdmins() ([]Admin, error) {
	data, err := os.ReadFile(s.adminFile)
	if err != nil {
		if os.IsNotExist(err) {
			return []Admin{}, nil
		}
		return nil, err
	}

	// Try list format first (new format)
	var admins []Admin
	if err := json.Unmarshal(data, &admins); err == nil {
		return admins, nil
	}

	// Fallback: single-object format (legacy Python .admin)
	var single Admin
	if err := json.Unmarshal(data, &single); err == nil && single.Username != "" {
		admins = []Admin{single}
		_ = s.saveAdmins(admins) // upgrade in place, best-effort
		return admins, nil
	}

	return []Admin{}, nil
}

func (s *Service) saveAdmins(admins []Admin) error {
	if err := os.MkdirAll(filepath.Dir(s.adminFile), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(admins, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(s.adminFile, data, 0o600); err != nil {
		return err
	}
	return os.Chmod(s.adminFile, 0o600)
}

// AdminExists returns true if at least one admin account exists.
func (s *Service) AdminExists() bool {
	admins, _ := s.loadAdmins()
	return len(admins) > 0
}

// ListAdmins returns public-safe admin records (no hashed passwords).
func (s *Service) ListAdmins() ([]AdminPublic, error) {
	admins, err := s.loadAdmins()
	if err != nil {
		return nil, err
	}
	result := make([]AdminPublic, len(admins))
	for i, a := range admins {
		result[i] = AdminPublic{Username: a.Username, CreatedAt: a.CreatedAt, LastLogin: a.LastLogin}
	}
	return result, nil
}

// Authenticate verifies username+password. Updates LastLogin on success.
func (s *Service) Authenticate(username, password string) error {
	admins, err := s.loadAdmins()
	if err != nil {
		return err
	}
	for i, a := range admins {
		if a.Username == username {
			if !s.VerifyPassword(password, a.HashedPassword) {
				return errors.New("invalid password")
			}
			now := time.Now()
			admins[i].LastLogin = &now
			_ = s.saveAdmins(admins)
			return nil
		}
	}
	return errors.New("admin not found")
}

// AddAdmin creates a new admin. Returns error if username exists or password too short.
func (s *Service) AddAdmin(username, password string) error {
	if len(password) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	admins, err := s.loadAdmins()
	if err != nil {
		return err
	}
	for _, a := range admins {
		if a.Username == username {
			return errors.New("admin already exists")
		}
	}
	hash, err := s.HashPassword(password)
	if err != nil {
		return err
	}
	admins = append(admins, Admin{
		Username:       username,
		HashedPassword: hash,
		CreatedAt:      time.Now(),
	})
	return s.saveAdmins(admins)
}

// DeleteAdmin removes an admin. Guards: cannot delete self, cannot delete last admin.
func (s *Service) DeleteAdmin(username, requester string) error {
	if username == requester {
		return errors.New("cannot delete yourself")
	}
	admins, err := s.loadAdmins()
	if err != nil {
		return err
	}
	if len(admins) <= 1 {
		return errors.New("cannot delete the last admin")
	}
	filtered := make([]Admin, 0, len(admins))
	for _, a := range admins {
		if a.Username != username {
			filtered = append(filtered, a)
		}
	}
	if len(filtered) == len(admins) {
		return errors.New("admin not found")
	}
	return s.saveAdmins(filtered)
}

// ChangePassword verifies old password then replaces hash.
func (s *Service) ChangePassword(username, oldPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return errors.New("password must be at least 8 characters")
	}
	admins, err := s.loadAdmins()
	if err != nil {
		return err
	}
	for i, a := range admins {
		if a.Username == username {
			if !s.VerifyPassword(oldPassword, a.HashedPassword) {
				return errors.New("current password is incorrect")
			}
			hash, err := s.HashPassword(newPassword)
			if err != nil {
				return err
			}
			admins[i].HashedPassword = hash
			return s.saveAdmins(admins)
		}
	}
	return errors.New("admin not found")
}
