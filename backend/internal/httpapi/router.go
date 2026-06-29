package httpapi

import (
	"strings"

	"smb-server/backend/internal/audit"
	"smb-server/backend/internal/auth"
	"smb-server/backend/internal/config"
	"smb-server/backend/internal/ldap"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

// SecurityHeaders returns a middleware that sets defensive HTTP security headers.
func SecurityHeaders() fiber.Handler {
	return func(c *fiber.Ctx) error {
		c.Set("X-Frame-Options", "DENY")
		c.Set("X-Content-Type-Options", "nosniff")
		c.Set("Referrer-Policy", "no-referrer")
		c.Set("Content-Security-Policy",
			"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self' data:; connect-src 'self'")
		return c.Next()
	}
}

// SetupRoutes registers all API routes. Static file serving is handled in main.go.
func SetupRoutes(app *fiber.App, cfg *config.Config, authSvc *auth.Service, exec samba.Executor, auditSvc *audit.Service) {
	app.Use(AuthMiddleware(authSvc))

	app.Get("/health", healthHandler)

	registerAuthRoutes(app.Group("/auth"), authSvc, cfg.CookieSecure)

	api := app.Group("/api")
	registerUsersRoutes(api.Group("/users"), exec)
	registerGroupsRoutes(api.Group("/groups"), exec)
	registerSharesRoutes(api.Group("/shares"), exec, cfg.SambaConfigPath, auditSvc)
	registerADRoutes(api.Group("/ad"), exec, ldap.Config{
		Server:  cfg.LDAPServer,
		Port:    cfg.LDAPPort,
		BaseDN:  cfg.LDAPBaseDN,
		BindDN:  cfg.LDAPBindDN,
		BindPW:  cfg.LDAPBindPW,
		Domain:  cfg.LDAPDomain,
	})
	// Derive builtin store path from audit log path sibling
	builtinPath := cfg.AuditLogPath[:strings.LastIndex(cfg.AuditLogPath, "/")+1] + "builtin_groups.json"
	registerBuiltinRoutes(api.Group("/builtin"), exec, builtinPath)
	registerAuditRoutes(api.Group("/audit"), auditSvc)
	registerStatsRoutes(api, exec, cfg.SambaConfigPath, auditSvc)
}
