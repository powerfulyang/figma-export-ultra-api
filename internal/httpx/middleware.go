package httpx

import (
	"fiber-ent-apollo-pg/pkg"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/cors"
	"github.com/gofiber/fiber/v2/middleware/recover"
	"github.com/gofiber/fiber/v2/middleware/requestid"
	"go.uber.org/zap"

	"fiber-ent-apollo-pg/internal/httpx/kit"
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
		// expose timing to downstream handlers
		c.Locals("req_start", start)
		err := c.Next()
		latency := pkg.SmartDurationFormat(time.Since(start))
		c.Set("X-Response-Time", latency)
		// Add Server-Timing header in milliseconds for client observability
		durMs := time.Since(start).Milliseconds()
		c.Set("Server-Timing", fmt.Sprintf("app;dur=%d", durMs))
		rid := kit.RequestID(c)
		httpxLogger.Info("access",
			zap.String("method", c.Method()),
			zap.String("path", c.OriginalURL()),
			zap.Int("status", c.Response().StatusCode()),
			zap.String("latency", latency),
			zap.String("ip", c.IP()),
			zap.String("ua", c.Get("User-Agent")),
			zap.String("request_id", rid),
		)
		return err
	})
}
