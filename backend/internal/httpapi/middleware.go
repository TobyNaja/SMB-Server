package httpapi

import (
	"strings"

	"smb-server/backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

// publicPrefixes are paths that skip JWT authentication.
var publicPrefixes = []string{
	"/login",
	"/setup",
	"/health",
	"/auth/login",
	"/auth/setup",
	"/docs",
	"/openapi.json",
}

// staticExtensions are file types served without auth.
var staticExtensions = []string{
	".js", ".css", ".ico", ".svg", ".png", ".woff", ".woff2", ".ttf", ".map",
}

// AuthMiddleware validates JWT from cookie or Bearer header.
// Stores the authenticated username in c.Locals("username").
func AuthMiddleware(svc *auth.Service) fiber.Handler {
	return func(c *fiber.Ctx) error {
		path := c.Path()

		// Public paths
		for _, p := range publicPrefixes {
			if strings.HasPrefix(path, p) {
				return c.Next()
			}
		}

		// Static asset extensions (served by Fiber.Static, no auth)
		for _, ext := range staticExtensions {
			if strings.HasSuffix(path, ext) {
				return c.Next()
			}
		}

		// Token from cookie first
		token := c.Cookies("access_token")

		// Fallback to Bearer header
		if token == "" {
			if hdr := c.Get("Authorization"); strings.HasPrefix(hdr, "Bearer ") {
				token = hdr[7:]
			}
		}

		if token == "" {
			return unauthorizedResponse(c, path, "Not authenticated")
		}

		username, err := svc.VerifyToken(token)
		if err != nil {
			return unauthorizedResponse(c, path, "Invalid token")
		}

		c.Locals("username", username)
		return c.Next()
	}
}

func unauthorizedResponse(c *fiber.Ctx, path, msg string) error {
	if strings.HasPrefix(path, "/api") || strings.HasPrefix(path, "/auth") {
		return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{"detail": msg})
	}
	return c.Redirect("/login", fiber.StatusSeeOther)
}

// actor extracts the authenticated username from Locals, defaulting to "unknown".
func actor(c *fiber.Ctx) string {
	if u, ok := c.Locals("username").(string); ok && u != "" {
		return u
	}
	return "unknown"
}
