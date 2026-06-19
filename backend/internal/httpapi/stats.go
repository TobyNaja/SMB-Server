package httpapi

import (
	"strings"

	"smb-server/backend/internal/audit"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
)

type statsHandlers struct {
	exec       samba.Executor
	configPath string
	auditSvc   *audit.Service
}

func registerStatsRoutes(g fiber.Router, exec samba.Executor, configPath string, auditSvc *audit.Service) {
	h := &statsHandlers{exec: exec, configPath: configPath, auditSvc: auditSvc}
	g.Get("/stats", h.stats)
	g.Get("/samba/status", h.sambaStatus)
}

func (h *statsHandlers) stats(c *fiber.Ctx) error {
	p := samba.NewSmbConfParser(h.configPath+"/shares.conf", h.configPath+"/smb.conf")
	shares, _ := p.GetAllShares()
	users := h.exec.GetUsers()
	groups := h.exec.GetGroups()
	recent, _ := h.auditSvc.GetLogs(5, "", "")
	return c.JSON(fiber.Map{
		"shares_count": len(shares),
		"users_count":  len(users),
		"groups_count": len(groups),
		"recent_audit": recent,
	})
}

func (h *statsHandlers) sambaStatus(c *fiber.Ctx) error {
	check := func(proc string) bool {
		r := h.exec.Execute("pidof " + proc)
		return r.Success && strings.TrimSpace(r.Output) != ""
	}
	return c.JSON(fiber.Map{
		"smbd":     check("smbd"),
		"nmbd":     check("nmbd"),
		"winbindd": check("winbindd"),
	})
}
