package audit_test

import (
	"path/filepath"
	"testing"

	"smb-server/backend/internal/audit"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newSvc(t *testing.T) *audit.Service {
	t.Helper()
	return audit.NewService(filepath.Join(t.TempDir(), "audit.json"))
}

func TestLogAndRetrieve(t *testing.T) {
	svc := newSvc(t)
	svc.Log("LOGIN", "admin", "AUTH", "web_login", "success", nil, "127.0.0.1")
	svc.Log("CREATE", "admin", "SHARE", "docs", "success", map[string]interface{}{"path": "/srv/docs"}, "")

	logs, err := svc.GetLogs(100, "", "")
	require.NoError(t, err)
	assert.Len(t, logs, 2)
	// Newest first
	assert.Equal(t, "CREATE", logs[0].Action)
	assert.Equal(t, "LOGIN", logs[1].Action)
}

func TestFilterByAction(t *testing.T) {
	svc := newSvc(t)
	svc.Log("LOGIN", "admin", "AUTH", "web_login", "success", nil, "")
	svc.Log("DELETE", "admin", "SHARE", "old", "success", nil, "")

	logs, err := svc.GetLogs(100, "DELETE", "")
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "DELETE", logs[0].Action)
}

func TestFilterByActor(t *testing.T) {
	svc := newSvc(t)
	svc.Log("LOGIN", "alice", "AUTH", "web_login", "success", nil, "")
	svc.Log("LOGIN", "bob", "AUTH", "web_login", "success", nil, "")

	logs, err := svc.GetLogs(100, "", "alice")
	require.NoError(t, err)
	assert.Len(t, logs, 1)
	assert.Equal(t, "alice", logs[0].Actor)
}

func TestLimitCapsResult(t *testing.T) {
	svc := newSvc(t)
	for i := 0; i < 10; i++ {
		svc.Log("ACTION", "admin", "T", "r", "success", nil, "")
	}
	logs, err := svc.GetLogs(3, "", "")
	require.NoError(t, err)
	assert.Len(t, logs, 3)
}

func TestEmptyLogReturnsEmptySlice(t *testing.T) {
	svc := newSvc(t)
	logs, err := svc.GetLogs(100, "", "")
	require.NoError(t, err)
	assert.Empty(t, logs)
}
