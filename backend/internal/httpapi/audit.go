package httpapi

import (
	"smb-server/backend/internal/audit"

	"github.com/gofiber/fiber/v2"
)

type auditHandlers struct {
	svc *audit.Service
}

func registerAuditRoutes(g fiber.Router, svc *audit.Service) {
	h := &auditHandlers{svc: svc}
	g.Get("/logs", h.getLogs)
}

func (h *auditHandlers) getLogs(c *fiber.Ctx) error {
	limit := c.QueryInt("limit", 100)
	action := c.Query("action")
	act := c.Query("actor")
	logs, err := h.svc.GetLogs(limit, action, act)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"detail": err.Error()})
	}
	return c.JSON(fiber.Map{"logs": logs, "count": len(logs)})
}
