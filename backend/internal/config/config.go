package config

import (
	"os"
	"strconv"

	"github.com/joho/godotenv"
)

type Config struct {
	APIHost            string
	APIPort            int
	SambaContainer     string
	SambaConfigPath    string
	SambaDataPath      string
	SecretKey          string
	TokenExpiryMinutes int
	AuditLogPath       string
	AdminCredsFile     string
	CookieSecure       bool
	LDAPServer         string
	LDAPPort           int
	LDAPBaseDN         string
	LDAPBindDN         string
	LDAPBindPW         string
	LDAPDomain         string
}

func Load() *Config {
	_ = godotenv.Load()
	return &Config{
		APIHost:            getEnv("API_HOST", "0.0.0.0"),
		APIPort:            getEnvInt("API_PORT", 8080),
		SambaContainer:     getEnv("SAMBA_CONTAINER", "samba-server"),
		SambaConfigPath:    getEnv("SAMBA_CONFIG_PATH", "/etc/samba"),
		SambaDataPath:      getEnv("SAMBA_DATA_PATH", "/var/lib/samba"),
		SecretKey:          getEnv("SECRET_KEY", "dev-secret-key-change-in-production"),
		TokenExpiryMinutes: getEnvInt("TOKEN_EXPIRY_MINUTES", 1440),
		AuditLogPath:       getEnv("AUDIT_LOG_PATH", "/mnt/shared/audit.json"),
		AdminCredsFile:     getEnv("ADMIN_CREDS_FILE", "/mnt/shared/.admin"),
		CookieSecure:       getEnvBool("COOKIE_SECURE", false),
		// LDAP defaults are empty — AD features require explicit env configuration.
		LDAPServer: getEnv("LDAP_SERVER", ""),
		LDAPPort:   getEnvInt("LDAP_PORT", 389),
		LDAPBaseDN: getEnv("LDAP_BASE_DN", ""),
		LDAPBindDN: getEnv("LDAP_BIND_DN", ""),
		LDAPBindPW: getEnv("LDAP_BIND_PW", ""),
		LDAPDomain: getEnv("LDAP_DOMAIN", ""),
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func getEnvInt(key string, fallback int) int {
	if v := os.Getenv(key); v != "" {
		if i, err := strconv.Atoi(v); err == nil {
			return i
		}
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := os.Getenv(key)
	if v == "true" || v == "1" || v == "yes" {
		return true
	}
	if v == "false" || v == "0" || v == "no" {
		return false
	}
	return fallback
}
