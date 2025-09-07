package httpx

import "github.com/gofiber/fiber/v2"

// HealthHandler 处理健康检查请求
//
//	@Summary		健康检查
//	@Description	检查API服务的健康状态
//	@Tags			health
//	@Accept			json
//	@Produce		json
//	@Success		200	{object}	map[string]string	"服务健康"
//	@Router			/health [get]
func HealthHandler(c *fiber.Ctx) error {
	return OK(c, fiber.Map{"status": "ok"})
}
