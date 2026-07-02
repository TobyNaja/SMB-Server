package httpapi_test

import (
	"encoding/json"
	"fmt"
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

// setupTestApp creates a minimal Fiber app wired up the same way as main.go,
// with an in-memory admin account ready for login.
func setupTestApp(t *testing.T) *fiber.App {
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

	app := fiber.New()
	app.Use(httpapi.SecurityHeaders())
	httpapi.SetupRoutes(app, cfg, authSvc, samba.NewFakeExecutor(), audit.NewService(cfg.AuditLogPath))
	return app
}

// loginAndGetToken logs in and returns the Bearer token for use in subsequent requests.
func loginAndGetToken(t *testing.T, app *fiber.App) string {
	t.Helper()
	body := fmt.Sprintf(`{"username":"testadmin","password":"testpassword123"}`)
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	var result struct {
		AccessToken string `json:"access_token"`
	}
	json.NewDecoder(resp.Body).Decode(&result)
	require.NotEmpty(t, result.AccessToken)
	return result.AccessToken
}

// ── Security headers ─────────────────────────────────────────────────────────

func TestSecurityHeaders_PresentOnEveryResponse(t *testing.T) {
	app := fiber.New()
	app.Use(httpapi.SecurityHeaders())
	app.Get("/test", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/test", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)

	assert.Equal(t, "DENY", resp.Header.Get("X-Frame-Options"))
	assert.Equal(t, "nosniff", resp.Header.Get("X-Content-Type-Options"))
	assert.Equal(t, "no-referrer", resp.Header.Get("Referrer-Policy"))
	assert.NotEmpty(t, resp.Header.Get("Content-Security-Policy"))
}

func TestSecurityHeaders_CSPBlocksExternalSources(t *testing.T) {
	app := fiber.New()
	app.Use(httpapi.SecurityHeaders())
	app.Get("/", func(c *fiber.Ctx) error { return c.SendStatus(200) })

	resp, _ := app.Test(httptest.NewRequest("GET", "/", nil))
	csp := resp.Header.Get("Content-Security-Policy")

	assert.Contains(t, csp, "default-src 'self'")
	assert.Contains(t, csp, "connect-src 'self'")
}

// ── JWT Cookie ───────────────────────────────────────────────────────────────

func TestLoginCookie_HasSameSiteStrict(t *testing.T) {
	app := setupTestApp(t)

	body := `{"username":"testadmin","password":"testpassword123"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	cookieHeader := resp.Header.Get("Set-Cookie")
	assert.Contains(t, cookieHeader, "SameSite=Strict")
	assert.Contains(t, cookieHeader, "HttpOnly")
}

func TestLoginCookie_NoSecureFlagOnHTTP(t *testing.T) {
	// CookieSecure=false (default) must not set the Secure flag,
	// so the cookie works in HTTP-only deployments.
	app := setupTestApp(t) // uses CookieSecure=false

	body := `{"username":"testadmin","password":"testpassword123"}`
	req := httptest.NewRequest("POST", "/auth/login", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	resp, err := app.Test(req)
	require.NoError(t, err)

	cookieHeader := resp.Header.Get("Set-Cookie")
	// "Secure" must not appear as an attribute (case-insensitive check)
	assert.NotContains(t, strings.ToLower(cookieHeader), "; secure")
}

// ── AD status — no credentials leak ─────────────────────────────────────────

func TestADStatus_HidesSensitiveFields(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	req := httptest.NewRequest("GET", "/api/ad/status", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := app.Test(req)
	require.NoError(t, err)
	require.Equal(t, 200, resp.StatusCode)

	var body map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&body)

	assert.NotContains(t, body, "ldap_server")
	assert.NotContains(t, body, "bind_dn")
	assert.NotContains(t, body, "base_dn")
	assert.NotContains(t, body, "error")
	_, hasDomain := body["domain"]
	assert.True(t, hasDomain, "domain field should be present")
	_, hasConnected := body["connected"]
	assert.True(t, hasConnected, "connected field should be present")
}

// TestADStatus_RequiresAuth verifies the endpoint is protected.
func TestADStatus_RequiresAuth(t *testing.T) {
	app := setupTestApp(t)

	req := httptest.NewRequest("GET", "/api/ad/status", nil)
	resp, err := app.Test(req)
	require.NoError(t, err)
	assert.Equal(t, 401, resp.StatusCode)
}

// ── Subfolder ACL path — shell injection guard ──────────────────────────────

// postJSON is a small helper for authenticated JSON POSTs.
func postJSON(t *testing.T, app *fiber.App, token, path, body string) int {
	t.Helper()
	req := httptest.NewRequest("POST", path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := app.Test(req)
	require.NoError(t, err)
	return resp.StatusCode
}

// TestSubfolderPermissions_RejectsShellInjection verifies that a subfolder path
// carrying shell metacharacters is rejected before any command is executed,
// while a clean relative path is accepted.
func TestSubfolderPermissions_RejectsShellInjection(t *testing.T) {
	app := setupTestApp(t)
	token := loginAndGetToken(t, app)

	// A share must exist so resolveSubfolder gets past the share lookup.
	require.Equal(t, 201, postJSON(t, app, token, "/api/shares",
		`{"name":"injtest","path":"/mnt/shared/injtest"}`))

	// Command-substitution attempt must be rejected (400), not executed.
	assert.Equal(t, 400, postJSON(t, app, token,
		"/api/shares/injtest/subfolders/permissions",
		`{"subfolder_path":"x'$(reboot)'","username":"alice","permissions":"rx"}`))

	// A clean relative path is accepted (FakeExecutor returns success).
	assert.Equal(t, 200, postJSON(t, app, token,
		"/api/shares/injtest/subfolders/permissions",
		`{"subfolder_path":"Secret_Plan","username":"alice","permissions":"rx"}`))
}
