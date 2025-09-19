package httpx

import (
	"github.com/gofiber/fiber/v2"
	fiberSwagger "github.com/swaggo/fiber-swagger"

	"fiber-ent-apollo-pg/ent"
	"fiber-ent-apollo-pg/internal/config"
	"fiber-ent-apollo-pg/internal/esx"
	"fiber-ent-apollo-pg/internal/httpx/admin"
	"fiber-ent-apollo-pg/internal/httpx/auth"
	"fiber-ent-apollo-pg/internal/httpx/configs"
	"fiber-ent-apollo-pg/internal/httpx/groups"
	"fiber-ent-apollo-pg/internal/httpx/mw"
	"fiber-ent-apollo-pg/internal/httpx/projects"
	"fiber-ent-apollo-pg/internal/httpx/users"
	"fiber-ent-apollo-pg/internal/mqx"
	"fiber-ent-apollo-pg/internal/redisx"
)

// Providers 外部依赖提供者
type Providers struct {
	MQ  mqx.Publisher
	ES  *esx.Client
	RDB *redisx.Client
}

// Register 注册所有 HTTP 路由
func Register(app *fiber.App, client *ent.Client, providers ...*Providers) {
	// Swagger 文档路由
	app.Get("/swagger/*", fiberSwagger.WrapHandler)

	// API v1 路由组
	v1 := app.Group("/api/v1")

	// JWT middleware (non-strict); handlers enforce when needed
	cfg, _, _, _ := config.Load()
	// Attach JWT middleware using auth parser
	app.Use(mw.JWTMiddlewareDynamic(func(token string) (string, string, []string, string, error) {
		claims, err := auth.ParseAndValidate(cfg, token)
		if err != nil {
			return "", "", nil, "", err
		}
		return claims.Subject, claims.Kind, claims.Roles, claims.DeviceID, nil
	}))
	var rdb *redisx.Client
	if len(providers) > 0 && providers[0] != nil {
		rdb = providers[0].RDB
	}

	// �������
	app.Get("/health", HealthHandler)

	// �û����·��
	v1.Get("/users", users.GetUsersHandler(client))
	v1.Post("/users", users.CreateUserHandler(client))

	// Auth routes
	v1.Post("/auth/register", mw.RateLimitDefault(rdb, cfg.RL.RegisterWindowSec, cfg.RL.RegisterMax), auth.RegisterHandler(cfg, client))
	v1.Post("/auth/anonymous/init", mw.RateLimitDefault(rdb, cfg.RL.AnonInitWindowSec, cfg.RL.AnonInitMax), auth.AnonymousInitHandler(cfg, client))
	v1.Post("/auth/login", mw.RateLimitDefault(rdb, cfg.RL.LoginWindowSec, cfg.RL.LoginMax), auth.LoginHandler(cfg, client))
	v1.Post("/auth/fp/sync", mw.RateLimitDefault(rdb, cfg.RL.FpSyncWindowSec, cfg.RL.FpSyncMax), auth.FpSyncHandler(client))
	v1.Post("/auth/refresh", mw.RateLimitDefault(rdb, cfg.RL.RefreshWindowSec, cfg.RL.RefreshMax), auth.RefreshHandler(cfg))
	v1.Post("/auth/logout", mw.RateLimitDefault(rdb, cfg.RL.LogoutWindowSec, cfg.RL.LogoutMax), auth.LogoutHandler())
	v1.Get("/auth/me", mw.RateLimitDefault(rdb, cfg.RL.MeWindowSec, cfg.RL.MeMax), auth.MeHandler())

	// Protected admin example (requires admin role)
	v1.Get("/admin/ping", mw.RequireUser(), mw.RequireRoles("admin"), admin.PingHandler())
	v1.Post("/admin/users/:id/promote", mw.RequireUser(), mw.RequireRoles("admin"), admin.PromoteUserHandler(client))

	// Configs & Groups
	v1.Get("/configs", mw.RequireUser(), configs.ListConfigsHandler(client))
	v1.Get("/configs/visible", mw.RequireUser(), configs.VisibleConfigsHandler(client))
	v1.Post("/configs", mw.RequireUser(), configs.CreateConfigHandler(client))
	v1.Put("/configs/:id", mw.RequireUser(), configs.UpdateConfigHandler(client))
	v1.Delete("/configs/:id", mw.RequireUser(), configs.DeleteConfigHandler(client))
	v1.Post("/configs/:id/share/groups", mw.RequireUser(), configs.ShareToGroupsHandler(client))
	v1.Post("/configs/:id/unshare/groups", mw.RequireUser(), configs.UnshareFromGroupsHandler(client))
	v1.Post("/configs/:id/share/user/:user_id", mw.RequireUser(), configs.ShareToUserHandler(client))
	v1.Post("/configs/:id/unshare/user/:user_id", mw.RequireUser(), configs.UnshareFromUserHandler(client))

	v1.Get("/groups", mw.RequireUser(), groups.ListMyGroupsHandler(client))
	v1.Post("/groups", mw.RequireUser(), groups.CreateGroupHandler(client))
	v1.Delete("/groups/:id", mw.RequireUser(), groups.DeleteGroupHandler(client))

	// Projects
	v1.Get("/projects", mw.RequireUser(), projects.ListProjectsHandler(client))
	v1.Post("/projects", mw.RequireUser(), projects.CreateProjectHandler(client))
	v1.Get("/projects/:id", mw.RequireUser(), projects.GetProjectHandler(client))
	v1.Put("/projects/:id", mw.RequireUser(), projects.UpdateProjectHandler(client))
	v1.Delete("/projects/:id", mw.RequireUser(), projects.DeleteProjectHandler(client))

	// Project Configs
	v1.Get("/projects/:id/configs", mw.RequireUser(), projects.ListProjectConfigsHandler(client))
	v1.Post("/projects/:id/configs", mw.RequireUser(), projects.AddConfigToProjectHandler(client))
	v1.Delete("/projects/:id/configs/:config_id", mw.RequireUser(), projects.RemoveConfigFromProjectHandler(client))
	v1.Put("/projects/:id/active-config", mw.RequireUser(), projects.SetActiveConfigHandler(client))
}
