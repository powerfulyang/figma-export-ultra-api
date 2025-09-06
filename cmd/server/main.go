// Package main is the entry point for the API server
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

	"fiber-ent-apollo-pg/internal/config"
	"fiber-ent-apollo-pg/internal/db"
	"fiber-ent-apollo-pg/internal/esx"
	"fiber-ent-apollo-pg/internal/httpx"
	"fiber-ent-apollo-pg/internal/logx"
	"fiber-ent-apollo-pg/internal/mqx"
	"fiber-ent-apollo-pg/internal/redisx"
	"fiber-ent-apollo-pg/internal/server"
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

	// Init logger
	logx.Init(cfg.Log.Level, cfg.Log.Format)
	logx.L().Info("config loaded", "env", cfg.AppEnv, "addr", cfg.Server.Addr, "log.level", cfg.Log.Level, "log.format", cfg.Log.Format)

	// Open DB (Ent + pgx)
	client, closeDB, err := db.Open(cfg)
	if err != nil {
		logx.L().Error("open db error", "err", err)
		panic(err)
	}
	defer closeDB()

	// Auto-migrate (demo purpose)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := client.Schema.Create(ctx); err != nil {
		logx.L().Error("auto migrate error", "err", err)
		panic(err)
	}

	// Optional deps: Redis, MQ, ES
	var (
		redisClose func()
		mqClose    func() error
	)
	rdb, rclose, err := redisx.Open(cfg)
	if err != nil {
		logx.L().Warn("redis init failed", "err", err)
	} else {
		redisClose = rclose
		defer redisClose()
	}

	var publisher mqx.Publisher
	if cfg.MQ.URL != "" {
		if pub, err := mqx.NewRabbitPublisher(cfg.MQ.URL, "events"); err != nil {
			logx.L().Warn("mq init failed", "err", err)
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
		logx.L().Warn("es init failed", "err", err)
	} else {
		defer esClose()
	}

	// Fiber app and routes
	app := fiber.New(fiber.Config{ErrorHandler: httpx.ErrorHandler()})
	httpx.RegisterCommonMiddlewares(app)
	_ = rdb // reserved for future http handlers
	providers := &httpx.Providers{MQ: publisher, ES: esClient}
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
			logx.L().Info("db pool updated", "max_open", newCfg.PG.MaxOpenConns, "max_idle", newCfg.PG.MaxIdleConns)
		}
		if changed["pg.url"] {
			logx.L().Warn("pg.url changed; restart required to reconnect")
		}
		if changed["server.addr"] {
			logx.L().Warn("server.addr changed; restart required to take effect", "addr", newCfg.Server.Addr)
		}
		if changed["log.level"] || changed["log.format"] {
			logx.Init(newCfg.Log.Level, newCfg.Log.Format)
			logx.L().Info("logger reconfigured", "level", newCfg.Log.Level, "format", newCfg.Log.Format)
		}
	})

	// Graceful shutdown
	go func() {
		ln, err := server.GetListener(cfg.Server.Addr)
		if err != nil {
			logx.L().Error("listener error", "err", err)
			return
		}
		if err := app.Listener(ln); err != nil {
			logx.L().Info("fiber exit", "err", err)
		}
	}()
	logx.L().Info("server started", "addr", cfg.Server.Addr)

	sig := make(chan os.Signal, 1)
	signal.Notify(sig, syscall.SIGINT, syscall.SIGTERM)
	<-sig
	logx.L().Info("shutting down...")
	_ = app.Shutdown()
}
