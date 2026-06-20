package httpapi

import (
	"time"

	"smb-server/backend/internal/auth"

	"github.com/gofiber/fiber/v2"
)

type authHandlers struct {
	svc          *auth.Service
	cookieSecure bool
}

type loginRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type adminCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
}

type changePasswordRequest struct {
	OldPassword string `json:"old_password"`
	NewPassword string `json:"new_password"`
}

func registerAuthRoutes(g fiber.Router, svc *auth.Service, cookieSecure bool) {
	h := &authHandlers{svc: svc, cookieSecure: cookieSecure}
	g.Get("/status", h.status)   // public — tells client if first-run setup is needed
	g.Post("/setup", h.setup)    // public — create first admin (blocked once one exists)
	g.Post("/login", h.login)
	g.Post("/logout", h.logout)
	g.Get("/me", h.me)
	g.Post("/change-password", h.changePassword)
	g.Get("/admins", h.listAdmins)
	g.Post("/admins", h.addAdmin)
	g.Delete("/admins/:username", h.deleteAdmin)
}

// status returns whether the system has been set up (at least one admin exists).
// Public — called by the frontend on load to decide whether to show /setup.
func (h *authHandlers) status(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"setup_required": !h.svc.AdminExists()})
}

// setup creates the first admin account. Returns 409 if any admin already exists.
func (h *authHandlers) setup(c *fiber.Ctx) error {
	if h.svc.AdminExists() {
		return c.Status(409).JSON(fiber.Map{"detail": "Setup already complete — use /auth/admins to manage accounts"})
	}
	var req adminCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"detail": "username and password required"})
	}
	if err := h.svc.AddAdmin(req.Username, req.Password); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"message": "Admin account created — please log in"})
}

func (h *authHandlers) login(c *fiber.Ctx) error {
	var req loginRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"detail": "username and password required"})
	}
	if err := h.svc.Authenticate(req.Username, req.Password); err != nil {
		return c.Status(401).JSON(fiber.Map{"detail": err.Error()})
	}
	token, expiresIn, err := h.svc.CreateToken(req.Username)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": "token creation failed"})
	}
	c.Cookie(&fiber.Cookie{
		Name:     "access_token",
		Value:    token,
		HTTPOnly: true,
		Secure:   h.cookieSecure,
		SameSite: "Strict",
		Path:     "/",
		MaxAge:   int(expiresIn),
	})
	return c.JSON(fiber.Map{
		"access_token": token,
		"token_type":   "bearer",
		"expires_in":   expiresIn,
	})
}

func (h *authHandlers) logout(c *fiber.Ctx) error {
	c.ClearCookie("access_token")
	return c.JSON(fiber.Map{"message": "Logged out successfully"})
}

func (h *authHandlers) me(c *fiber.Ctx) error {
	username := actor(c)
	if username == "unknown" {
		return c.Status(401).JSON(fiber.Map{"detail": "Not authenticated"})
	}
	expiresAt := time.Now().Add(24 * time.Hour)
	if tok, ok := c.Locals("token").(string); ok && tok != "" {
		if exp, err := h.svc.TokenExpiry(tok); err == nil {
			expiresAt = exp
		}
	}
	return c.JSON(fiber.Map{
		"username":   username,
		"is_admin":   true,
		"expires_at": expiresAt.UTC().Format(time.RFC3339),
	})
}

func (h *authHandlers) changePassword(c *fiber.Ctx) error {
	var req changePasswordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if err := h.svc.ChangePassword(actor(c), req.OldPassword, req.NewPassword); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Password changed successfully"})
}

func (h *authHandlers) listAdmins(c *fiber.Ctx) error {
	admins, err := h.svc.ListAdmins()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"admins": admins, "count": len(admins)})
}

func (h *authHandlers) addAdmin(c *fiber.Ctx) error {
	var req adminCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if err := h.svc.AddAdmin(req.Username, req.Password); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.Status(201).JSON(fiber.Map{"message": "Admin created"})
}

func (h *authHandlers) deleteAdmin(c *fiber.Ctx) error {
	username := c.Params("username")
	if err := h.svc.DeleteAdmin(username, actor(c)); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Admin deleted"})
}
