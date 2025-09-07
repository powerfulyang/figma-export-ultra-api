package httpx

import "github.com/gofiber/fiber/v2"

// HealthHandler 处理健康检查请求
func HealthHandler(c *fiber.Ctx) error {
	return OK(c, fiber.Map{"status": "ok"})
}
