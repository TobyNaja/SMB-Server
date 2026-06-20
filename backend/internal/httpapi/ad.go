package httpapi

import (
	"smb-server/backend/internal/ldap"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

type adHandlers struct {
	svc *ldap.Service
}

func registerADRoutes(g fiber.Router, exec samba.Executor, cfg ldap.Config) {
	h := &adHandlers{svc: ldap.NewService(exec, cfg)}
	g.Get("/status", h.status)
	g.Get("/users", h.searchUsers)
	g.Get("/users/:username", h.getUser)
	g.Get("/groups", h.searchGroups)
	g.Get("/ous", h.listOUs)
}

func (h *adHandlers) status(c *fiber.Ctx) error {
	result := h.svc.TestConnection()
	return c.JSON(result)
}

func (h *adHandlers) searchUsers(c *fiber.Ctx) error {
	q := c.Query("q")
	ou := c.Query("ou")
	limit := c.QueryInt("limit", 0)
	users, err := h.svc.SearchUsers(q, ou, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"users": users, "count": len(users)})
}

func (h *adHandlers) getUser(c *fiber.Ctx) error {
	username := c.Params("username")
	user, err := h.svc.GetUser(username)
	if err != nil || user == nil {
		return c.Status(404).JSON(fiber.Map{"detail": "User not found in AD"})
	}
	return c.JSON(user)
}

func (h *adHandlers) searchGroups(c *fiber.Ctx) error {
	q := c.Query("q")
	limit := c.QueryInt("limit", 0)
	groups, err := h.svc.SearchGroups(q, limit)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"groups": groups, "count": len(groups)})
}

func (h *adHandlers) listOUs(c *fiber.Ctx) error {
	ous := h.svc.ListOUs()
	return c.JSON(fiber.Map{"ous": ous})
}
