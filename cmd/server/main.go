// Package main is the entry point for the API server
//
//	@title			Figma Export Ultra API
//	@version		1.0
//	@description	这是一个用于 Figma 数据导出的 API 服务
//	@termsOfService	http://swagger.io/terms/
//
//	@contact.name	API Support
//	@contact.url	http://www.swagger.io/support
//	@contact.email	support@swagger.io
//
//	@license.name	Apache 2.0
//	@license.url	http://www.apache.org/licenses/LICENSE-2.0.html
//
//	@host		localhost:8080
//	@BasePath	/api/v1
//
//	@schemes		http https
//
//	@securityDefinitions.apikey	BearerAuth
//	@in						header
//	@name					Authorization
//
//	@security			BearerAuth
//
//	@externalDocs.description	OpenAPI
//	@externalDocs.url			https://swagger.io/resources/open-api/
package main

import (
	"context"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/joho/godotenv"
	"go.uber.org/zap"

	"fiber-ent-apollo-pg/internal/config"
	"fiber-ent-apollo-pg/internal/db"
	"fiber-ent-apollo-pg/internal/esx"
	"fiber-ent-apollo-pg/internal/httpx"
	"fiber-ent-apollo-pg/internal/logx"
	"fiber-ent-apollo-pg/internal/mqx"
	"fiber-ent-apollo-pg/internal/redisx"
	"fiber-ent-apollo-pg/internal/server"

	_ "fiber-ent-apollo-pg/docs" // swagger docs
)

func main() {
	// Load .env if present
	_ = godotenv.Load()

	// Load config (env first; optional Apollo override)
	cfg, store, apClose, err := config.Load()
	if err != nil {
		panic(err)
	}
	if apClose != nil {
		defer apClose()
	}

	// Init global logger first
	logx.Init(cfg.Log.Level, cfg.Log.Format)

	// Create the main scope logger using the convenient GetScope function
	mainLogger := logx.GetScope("main")

	mainLogger.Info("config loaded",
		zap.String("env", cfg.AppEnv),
		zap.String("addr", cfg.Server.Addr),
		zap.String("log.level", cfg.Log.Level),
		zap.String("log.format", cfg.Log.Format),
	)

	// Open DB (Ent + pgx)
	client, closeDB, err := db.Open(cfg)
	if err != nil {
		mainLogger.Sugar().Error("open db error", "err", err)
		panic(err)
	}
	defer closeDB()

	// Auto-migrate (demo purpose)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Schema.Create(ctx); err != nil {
		mainLogger.Sugar().Error("auto migrate error", "err", err)
		panic(err)
	}

	// Optional deps: Redis, MQ, ES
	var (
		redisClose func()
		mqClose    func() error
	)
	rdb, rclose, err := redisx.Open(cfg)
	if err != nil {
		mainLogger.Sugar().Warn("redis init failed", "err", err)
	} else {
		redisClose = rclose
		defer redisClose()
	}

	var publisher mqx.Publisher
	if cfg.MQ.URL != "" {
		if pub, err := mqx.NewRabbitPublisher(cfg.MQ.URL, "events"); err != nil {
			mainLogger.Sugar().Warn("mq init failed", "err", err)
		} else {
			publisher = pub
			mqClose = pub.Close
			defer func() {
				if mqClose != nil {
					_ = mqClose()
				}
			}()
		}
	}

	esClient, esClose, err := esx.Open(cfg)
	if err != nil {
		mainLogger.Sugar().Warn("es init failed", "err", err)
	} else {
		defer esClose()
	}

	// Fiber app and routes
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler()})
	httpx.RegisterCommonMiddlewares(app)
	_ = rdb // reserved for future http handlers
	providers := &httpx.Providers{MQ: publisher, ES: esClient, RDB: rdb}
	httpx.Register(app, client, providers)

	// Watch for dynamic config changes (Apollo)
	// Validators: rollback strategy for invalid config
	store.AddValidator(func(newCfg *config.Config, changed map[string]bool) error {
		if changed["pg.max_open"] || changed["pg.max_idle"] {
			if newCfg.PG.MaxIdleConns > newCfg.PG.MaxOpenConns {
				return fmt.Errorf("PG_MAX_IDLE cannot exceed PG_MAX_OPEN")
			}
		}
		return nil
	})

	store.Watch(func(newCfg *config.Config, changed map[string]bool) {
		if changed["pg.max_open"] || changed["pg.max_idle"] {
			db.UpdatePool(newCfg.PG.MaxOpenConns, newCfg.PG.MaxIdleConns)
			mainLogger.Info("db pool updated",
				zap.Int("max_open", newCfg.PG.MaxOpenConns),
				zap.Int("max_idle", newCfg.PG.MaxIdleConns),
			)
		}
		if changed["pg.url"] {
			mainLogger.Warn("pg.url changed; restart required to reconnect")
		}
		if changed["server.addr"] {
			mainLogger.Warn("server.addr changed; restart required to take effect",
				zap.String("addr", newCfg.Server.Addr),
			)
		}
		if changed["log.level"] || changed["log.format"] {
			logx.Init(newCfg.Log.Level, newCfg.Log.Format)
			mainLogger.Info("logger reconfigured",
				zap.String("level", newCfg.Log.Level),
				zap.String("format", newCfg.Log.Format),
			)
		}
	})

	// Graceful shutdown
	go func() {
		ln, err := server.GetListener(cfg.Server.Addr)
		if err != nil {
			mainLogger.Sugar().Errorf("listener error: %v", err)
			return
		}
		if err := app.Listener(ln); err != nil {
			mainLogger.Sugar().Infof("fiber exit: %v", err)
		}
	}()
	mainLogger.Sugar().Infof("server started on %s", cfg.Server.Addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	mainLogger.Sugar().Info("shutting down...")
	_ = app.Shutdown()
}
