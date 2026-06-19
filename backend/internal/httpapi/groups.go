package httpapi

import (
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

type groupsHandlers struct {
	exec samba.Executor
}

type groupCreateRequest struct {
	GroupName string `json:"group_name"`
}

func registerGroupsRoutes(g fiber.Router, exec samba.Executor) {
	h := &groupsHandlers{exec: exec}
	g.Get("", h.list)
	g.Post("", h.create)
	g.Post("/:group/members/:username", h.addMember)
	g.Delete("/:group/members/:username", h.removeMember)
}

func (h *groupsHandlers) list(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"groups": h.exec.GetGroups()})
}

func (h *groupsHandlers) create(c *fiber.Ctx) error {
	var req groupCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	result := h.exec.CreateGroup(req.GroupName)
	if !result.Success {
		return c.Status(400).JSON(fiber.Map{"detail": result.Output})
	}
	return c.Status(201).JSON(fiber.Map{"message": "Group " + req.GroupName + " created"})
}

func (h *groupsHandlers) addMember(c *fiber.Ctx) error {
	group := c.Params("group")
	username := c.Params("username")
	result := h.exec.AddUserToGroup(username, group)
	if !result.Success {
		return c.Status(400).JSON(fiber.Map{"detail": result.Output})
	}
	return c.JSON(fiber.Map{"message": "Added " + username + " to " + group})
}

func (h *groupsHandlers) removeMember(c *fiber.Ctx) error {
	group := c.Params("group")
	username := c.Params("username")
	result := h.exec.RemoveUserFromGroup(username, group)
	if !result.Success {
		return c.Status(400).JSON(fiber.Map{"detail": result.Output})
	}
	return c.JSON(fiber.Map{"message": "Removed " + username + " from " + group})
}
