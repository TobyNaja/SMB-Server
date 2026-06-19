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
		LDAPServer:         getEnv("LDAP_SERVER", "10.70.37.143"),
		LDAPPort:           getEnvInt("LDAP_PORT", 389),
		LDAPBaseDN:         getEnv("LDAP_BASE_DN", "DC=it,DC=kmitl,DC=ac,DC=th"),
		LDAPBindDN:         getEnv("LDAP_BIND_DN", "ldap-bind-nas@IT.KMITL.AC.TH"),
		LDAPBindPW:         getEnv("LDAP_BIND_PW", ""),
		LDAPDomain:         getEnv("LDAP_DOMAIN", "IT.KMITL.AC.TH"),
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
