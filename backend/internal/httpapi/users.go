package httpapi

import (
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

type usersHandlers struct {
	exec samba.Executor
}

type userCreateRequest struct {
	Username string `json:"username"`
	Password string `json:"password"`
	Fullname string `json:"fullname"`
}

type passwordRequest struct {
	Password string `json:"password"`
}

func registerUsersRoutes(g fiber.Router, exec samba.Executor) {
	h := &usersHandlers{exec: exec}
	g.Get("", h.list)
	g.Post("", h.create)
	g.Delete("/:username", h.delete)
	g.Post("/:username/password", h.setPassword)
}

func (h *usersHandlers) list(c *fiber.Ctx) error {
	users := h.exec.GetUsers()
	return c.JSON(fiber.Map{"users": users})
}

func (h *usersHandlers) create(c *fiber.Ctx) error {
	var req userCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	if req.Username == "" || req.Password == "" {
		return c.Status(400).JSON(fiber.Map{"detail": "username and password required"})
	}
	result := h.exec.CreateUser(req.Username, req.Password)
	if !result.Success {
		return c.Status(400).JSON(fiber.Map{"detail": result.Output})
	}
	h.exec.ReloadSamba()
	return c.Status(201).JSON(fiber.Map{"message": "User " + req.Username + " created"})
}

func (h *usersHandlers) delete(c *fiber.Ctx) error {
	username := c.Params("username")
	h.exec.DeleteUser(username)
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "User " + username + " deleted"})
}

func (h *usersHandlers) setPassword(c *fiber.Ctx) error {
	username := c.Params("username")
	var req passwordRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"detail": "invalid request body"})
	}
	result := h.exec.SetPassword(username, req.Password)
	if !result.Success {
		return c.Status(400).JSON(fiber.Map{"detail": result.Output})
	}
	h.exec.ReloadSamba()
	return c.JSON(fiber.Map{"message": "Password updated"})
}
