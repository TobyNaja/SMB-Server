package main

import (
	"fmt"
	"log"

	"smb-server/backend/internal/audit"
	"smb-server/backend/internal/auth"
	"smb-server/backend/internal/config"
	"smb-server/backend/internal/httpapi"
	"smb-server/backend/internal/samba"

	"github.com/gofiber/fiber/v2"
	fiberlogger "github.com/gofiber/fiber/v2/middleware/logger"
	"github.com/gofiber/fiber/v2/middleware/recover"
)

func main() {
	cfg := config.Load()

	authSvc := auth.New(cfg.SecretKey, cfg.AdminCredsFile, cfg.TokenExpiryMinutes)
	auditSvc := audit.NewService(cfg.AuditLogPath)
	exec := samba.NewDockerExecutor(cfg.SambaContainer)

	app := fiber.New(fiber.Config{
		ErrorHandler: func(c *fiber.Ctx, err error) error {
			log.Printf("unhandled error: %v", err)
			return c.Status(500).JSON(fiber.Map{"detail": "Internal server error"})
		},
	})

	app.Use(recover.New())
	app.Use(fiberlogger.New(fiberlogger.Config{
		Format: "[${time}] ${status} - ${latency} ${method} ${path}\n",
	}))
	app.Use(httpapi.SecurityHeaders())

	// Register API routes (must come before static serving)
	httpapi.SetupRoutes(app, cfg, authSvc, exec, auditSvc)

	// Serve compiled SvelteKit SPA
	app.Static("/", "./frontend/build", fiber.Static{
		Compress: true,
		Browse:   false,
	})

	// SPA fallback: unknown paths serve index.html so client-side routing works
	app.Use(func(c *fiber.Ctx) error {
		return c.SendFile("./frontend/build/index.html")
	})

	addr := fmt.Sprintf("%s:%d", cfg.APIHost, cfg.APIPort)
	log.Printf("SMB Manager starting on %s", addr)
	log.Fatal(app.Listen(addr))
}
