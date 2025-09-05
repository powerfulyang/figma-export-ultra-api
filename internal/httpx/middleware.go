package httpx

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"

	"fiber-ent-apollo-pg/internal/logx"
)

// RegisterCommonMiddlewares registers common middlewares and a structured access log.
func RegisterCommonMiddlewares(app *fiber.App) {
	app.Use(recover.New())
	app.Use(requestid.New())
	app.Use(cors.New())

	// Structured access log
	app.Use(func(c *fiber.Ctx) error {
		start := time.Now()
		err := c.Next()
		latency := time.Since(start)
		rid := c.GetRespHeader("X-Request-ID")
		if rid == "" {
			rid = c.Get("X-Request-ID")
		}
		logx.L().Info("access",
			"method", c.Method(),
			"path", c.OriginalURL(),
			"status", c.Response().StatusCode(),
			"latency_ms", latency.Milliseconds(),
			"ip", c.IP(),
			"ua", c.Get("User-Agent"),
			"request_id", rid,
		)
		return err
	})
}
