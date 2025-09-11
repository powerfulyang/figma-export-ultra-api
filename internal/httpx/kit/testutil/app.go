package testutil

import (
	"github.com/gofiber/fiber/v2"

	"fiber-ent-apollo-pg/internal/httpx/kit"
)

// NewApp creates a Fiber app with the standard error handler and applies
// the given mount functions to register selective routes. Useful for tests.
func NewApp(mounts ...func(*fiber.App)) *fiber.App {
	app := fiber.New(fiber.Config{ErrorHandler: kit.ErrorHandler()})
	for _, m := range mounts {
		if m != nil {
			m(app)
		}
	}
	return app
}
