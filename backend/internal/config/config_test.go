package config_test

import (
	"os"
	"testing"

	"smb-server/backend/internal/config"

	"github.com/stretchr/testify/assert"
)

// unsetKeys clears env vars and restores them via t.Cleanup.
func unsetKeys(t *testing.T, keys ...string) {
	t.Helper()
	originals := make(map[string]string, len(keys))
	for _, k := range keys {
		originals[k] = os.Getenv(k)
		os.Unsetenv(k)
	}
	t.Cleanup(func() {
		for k, v := range originals {
			if v != "" {
				os.Setenv(k, v)
			} else {
				os.Unsetenv(k)
			}
		}
	})
}

// ── LDAP credential defaults ─────────────────────────────────────────────────

// TestLDAPDefaultsAreEmpty verifies that no real server addresses or account
// names are hardcoded as fallback defaults. All LDAP fields must be empty
// when the corresponding env vars are absent.
func TestLDAPDefaultsAreEmpty(t *testing.T) {
	unsetKeys(t, "LDAP_SERVER", "LDAP_BASE_DN", "LDAP_BIND_DN", "LDAP_DOMAIN")

	cfg := config.Load()
	assert.Empty(t, cfg.LDAPServer, "LDAP_SERVER default must be empty (no real IP hardcoded)")
	assert.Empty(t, cfg.LDAPBaseDN, "LDAP_BASE_DN default must be empty")
	assert.Empty(t, cfg.LDAPBindDN, "LDAP_BIND_DN default must be empty (no real account hardcoded)")
	assert.Empty(t, cfg.LDAPDomain, "LDAP_DOMAIN default must be empty")
}

func TestLDAPConfigLoadedFromEnv(t *testing.T) {
	os.Setenv("LDAP_SERVER", "192.168.1.10")
	os.Setenv("LDAP_BASE_DN", "DC=example,DC=com")
	os.Setenv("LDAP_BIND_DN", "bind@example.com")
	os.Setenv("LDAP_DOMAIN", "EXAMPLE.COM")
	t.Cleanup(func() {
		os.Unsetenv("LDAP_SERVER")
		os.Unsetenv("LDAP_BASE_DN")
		os.Unsetenv("LDAP_BIND_DN")
		os.Unsetenv("LDAP_DOMAIN")
	})

	cfg := config.Load()
	assert.Equal(t, "192.168.1.10", cfg.LDAPServer)
	assert.Equal(t, "DC=example,DC=com", cfg.LDAPBaseDN)
	assert.Equal(t, "bind@example.com", cfg.LDAPBindDN)
	assert.Equal(t, "EXAMPLE.COM", cfg.LDAPDomain)
}

// ── CookieSecure parsing ─────────────────────────────────────────────────────

func TestCookieSecureDefaultFalse(t *testing.T) {
	unsetKeys(t, "COOKIE_SECURE")
	assert.False(t, config.Load().CookieSecure)
}

func TestCookieSecure_TrueVariants(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("COOKIE_SECURE") })
	for _, v := range []string{"true", "1", "yes"} {
		os.Setenv("COOKIE_SECURE", v)
		assert.True(t, config.Load().CookieSecure, "COOKIE_SECURE=%q should be true", v)
	}
}

func TestCookieSecure_FalseVariants(t *testing.T) {
	t.Cleanup(func() { os.Unsetenv("COOKIE_SECURE") })
	for _, v := range []string{"false", "0", "no"} {
		os.Setenv("COOKIE_SECURE", v)
		assert.False(t, config.Load().CookieSecure, "COOKIE_SECURE=%q should be false", v)
	}
}
