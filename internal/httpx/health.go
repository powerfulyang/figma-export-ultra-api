package httpx

import "github.com/gofiber/fiber/v2"
import "fiber-ent-apollo-pg/internal/httpx/kit"

// HealthHandler returns liveness status
//
//	@Summary      Health Check
//	@Description  API health/liveness probe
//	@Tags         health
//	@Accept       json
//	@Produce      json
//	@Success      200  {object}  map[string]string  "ok"
//	@Router       /health [get]
func HealthHandler(c *fiber.Ctx) error { return kit.OK(c, fiber.Map{"status": "ok"}) }
