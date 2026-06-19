package auth_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
	"time"

	"smb-server/backend/internal/auth"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) (*auth.Service, string) {
	t.Helper()
	dir := t.TempDir()
	adminFile := filepath.Join(dir, ".admin")
	svc := auth.New("test-secret-key-32chars-minimum!!", adminFile, 60)
	return svc, adminFile
}

// --- bcrypt ---

func TestHashAndVerifyPassword(t *testing.T) {
	svc, _ := newSvc(t)
	hash, err := svc.HashPassword("MyPassword1!")
	require.NoError(t, err)
	assert.True(t, svc.VerifyPassword("MyPassword1!", hash))
	assert.False(t, svc.VerifyPassword("WrongPassword", hash))
}

// --- JWT ---

func TestCreateAndVerifyToken(t *testing.T) {
	svc, _ := newSvc(t)
	token, expiresIn, err := svc.CreateToken("admin")
	require.NoError(t, err)
	assert.Greater(t, expiresIn, int64(0))

	username, err := svc.VerifyToken(token)
	require.NoError(t, err)
	assert.Equal(t, "admin", username)
}

func TestVerifyToken_Invalid(t *testing.T) {
	svc, _ := newSvc(t)
	_, err := svc.VerifyToken("not.a.token")
	assert.Error(t, err)
}

func TestVerifyToken_Expired(t *testing.T) {
	dir := t.TempDir()
	adminFile := filepath.Join(dir, ".admin")
	// token expiry of 0 minutes → instantly expired
	svc := auth.New("test-secret-key-32chars-minimum!!", adminFile, -1)
	token, _, err := svc.CreateToken("admin")
	require.NoError(t, err)
	time.Sleep(2 * time.Millisecond)
	_, err = svc.VerifyToken(token)
	assert.Error(t, err)
}

// --- Admin store: multi-admin ---

func TestAddAndListAdmins(t *testing.T) {
	svc, _ := newSvc(t)
	err := svc.AddAdmin("alice", "password123")
	require.NoError(t, err)
	err = svc.AddAdmin("bob", "password456")
	require.NoError(t, err)

	admins, err := svc.ListAdmins()
	require.NoError(t, err)
	assert.Len(t, admins, 2)
	assert.Equal(t, "alice", admins[0].Username)
	assert.Equal(t, "bob", admins[1].Username)
}

func TestAddAdmin_DuplicateRejected(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	err := svc.AddAdmin("alice", "password999")
	assert.Error(t, err)
}

func TestAddAdmin_ShortPasswordRejected(t *testing.T) {
	svc, _ := newSvc(t)
	err := svc.AddAdmin("alice", "short")
	assert.Error(t, err)
}

func TestDeleteAdmin_CannotDeleteSelf(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	require.NoError(t, svc.AddAdmin("bob", "password456"))
	err := svc.DeleteAdmin("alice", "alice")
	assert.Error(t, err)
}

func TestDeleteAdmin_CannotDeleteLast(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	err := svc.DeleteAdmin("alice", "someone-else")
	assert.Error(t, err)
}

func TestDeleteAdmin_Success(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	require.NoError(t, svc.AddAdmin("bob", "password456"))
	err := svc.DeleteAdmin("bob", "alice")
	require.NoError(t, err)
	admins, _ := svc.ListAdmins()
	assert.Len(t, admins, 1)
	assert.Equal(t, "alice", admins[0].Username)
}

func TestAuthenticate_Success(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	err := svc.Authenticate("alice", "password123")
	assert.NoError(t, err)
}

func TestAuthenticate_WrongPassword(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "password123"))
	err := svc.Authenticate("alice", "wrongpass")
	assert.Error(t, err)
}

func TestAuthenticate_UnknownUser(t *testing.T) {
	svc, _ := newSvc(t)
	err := svc.Authenticate("nobody", "password123")
	assert.Error(t, err)
}

func TestChangePassword(t *testing.T) {
	svc, _ := newSvc(t)
	require.NoError(t, svc.AddAdmin("alice", "oldpassword"))
	require.NoError(t, svc.ChangePassword("alice", "oldpassword", "newpassword"))
	assert.NoError(t, svc.Authenticate("alice", "newpassword"))
	assert.Error(t, svc.Authenticate("alice", "oldpassword"))
}

// --- Legacy .admin migration ---

func TestLegacySingleObjectMigration(t *testing.T) {
	dir := t.TempDir()
	adminFile := filepath.Join(dir, ".admin")

	// Write old Python single-object .admin format
	legacy := map[string]interface{}{
		"username":        "admin",
		"hashed_password": "$2b$12$fakehashfortest000000000000000000000000000000000000000",
		"created_at":      time.Now().Format(time.RFC3339),
	}
	data, _ := json.MarshalIndent(legacy, "", "  ")
	require.NoError(t, os.WriteFile(adminFile, data, 0o600))

	svc := auth.New("test-secret-key-32chars-minimum!!", adminFile, 60)
	admins, err := svc.ListAdmins()
	require.NoError(t, err)
	require.Len(t, admins, 1)
	assert.Equal(t, "admin", admins[0].Username)

	// Verify file was upgraded to list format
	upgraded, _ := os.ReadFile(adminFile)
	var list []map[string]interface{}
	require.NoError(t, json.Unmarshal(upgraded, &list))
	assert.Len(t, list, 1)
}
