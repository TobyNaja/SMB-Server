package httpapi

import (
	"strings"
	"sync"

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

	// Fetch users, groups, and recent audit concurrently to minimize latency.
	var users []samba.UserInfo
	var groups []string
	var recent []audit.Entry
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); users = h.exec.GetUsers() }()
	go func() { defer wg.Done(); groups = h.exec.GetGroups() }()
	go func() { defer wg.Done(); recent, _ = h.auditSvc.GetLogs(5, "", "") }()
	wg.Wait()

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

	// Check all three Samba processes concurrently.
	var smbd, nmbd, winbindd bool
	var wg sync.WaitGroup
	wg.Add(3)
	go func() { defer wg.Done(); smbd = check("smbd") }()
	go func() { defer wg.Done(); nmbd = check("nmbd") }()
	go func() { defer wg.Done(); winbindd = check("winbindd") }()
	wg.Wait()

	return c.JSON(fiber.Map{
		"smbd":     smbd,
		"nmbd":     nmbd,
		"winbindd": winbindd,
	})
}
