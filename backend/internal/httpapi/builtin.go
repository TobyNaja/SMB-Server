package httpapi

import (
	"smb-server/backend/internal/builtin"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

type builtinHandlers struct {
	svc *builtin.Service
}

type memberActionRequest struct {
	Username string `json:"username"`
}

func registerBuiltinRoutes(g fiber.Router, exec samba.Executor, storePath string) {
	h := &builtinHandlers{svc: builtin.NewService(exec, storePath)}
	g.Get("", h.list)
	g.Get("/:group/members", h.getMembers)
	g.Post("/:group/members", h.addMember)
	g.Delete("/:group/members/:username", h.removeMember)
}

func (h *builtinHandlers) list(c *fiber.Ctx) error {
	groups, err := h.svc.ListGroups()
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"groups": groups})
}

func (h *builtinHandlers) getMembers(c *fiber.Ctx) error {
	group := c.Params("group")
	result, err := h.svc.GetGroup(group)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(result)
}

func (h *builtinHandlers) addMember(c *fiber.Ctx) error {
	group := c.Params("group")
	var req memberActionRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	members, err := h.svc.AddMember(group, req.Username)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Added '" + req.Username + "' to BUILTIN\\" + group, "members": members})
}

func (h *builtinHandlers) removeMember(c *fiber.Ctx) error {
	group := c.Params("group")
	username := c.Params("username")
	members, err := h.svc.RemoveMember(group, username)
	if err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"message": "Removed '" + username + "' from BUILTIN\\" + group, "members": members})
}
