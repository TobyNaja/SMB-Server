package httpapi

import "github.com/gofiber/fiber/v2"

func healthHandler(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{"status": "healthy"})
}
