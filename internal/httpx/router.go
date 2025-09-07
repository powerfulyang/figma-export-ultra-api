package httpx

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"

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
	// Swagger 文档路由
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// API v1 路由组
	v1 := app.Group("/api/v1")

	// 健康检查
	app.Get("/health", HealthHandler)

	// 用户相关路由
	v1.Get("/users", GetUsersHandler(client))
	v1.Post("/users", CreateUserHandler(client))
}
