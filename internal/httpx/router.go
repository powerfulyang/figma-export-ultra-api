package httpx

import (
	"github.com/gofiber/fiber/v2"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/internal/esx"
	"fiber-ent-apollo-pg/internal/mqx"
)

// Providers 包含外部服务的提供者
type Providers struct {
	MQ mqx.Publisher
	ES *esx.Client
}

// Register 注册所有的HTTP路由
func Register(app *fiber.App, client *ent.Client, _ ...*Providers) {
	app.Get("/health", HealthHandler)
	app.Get("/users", GetUsersHandler(client))
	app.Post("/users", CreateUserHandler(client))
}
