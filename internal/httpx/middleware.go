package httpx

import (
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"

	"fiber-ent-apollo-pg/internal/logx"
)

var httpxLogger = logx.GetScope("httpx")

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
		rid := requestID(c)
		httpxLogger.Info("access",
			zap.String("method", c.Method()),
			zap.String("path", c.OriginalURL()),
			zap.Int("status", c.Response().StatusCode()),
			zap.Int64("latency_ms", latency.Milliseconds()),
			zap.String("ip", c.IP()),
			zap.String("ua", c.Get("User-Agent")),
			zap.String("request_id", rid),
		)
		return err
	})
}
