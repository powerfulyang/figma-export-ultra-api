package config

import (
	"log"
	"os"
	"strconv"
)

type Config struct {
	AppEnv string
	Server struct {
		Addr string
	}
	Log struct {
		Level  string // debug, info, warn, error
		Format string // text, json
	}
	PG struct {
		URL          string
		MaxOpenConns int
		MaxIdleConns int
	}
	Redis struct {
		Addr     string
		Password string
		DB       int
	}
	MQ struct {
		URL string // RabbitMQ URL
	}
	ES struct {
		Addrs    string // comma separated
		Username string
		Password string
	}
	Apollo struct {
		Enable    bool
		AppID     string
		Cluster   string
		Namespace string
		Addrs     string
		AccessKey string
	}
}

// Load loads config from env, and if enabled, overrides with Apollo values.
// Returns config, optional apollo closer, and error.
func Load() (*Config, *Store, func(), error) {
	cfg := &Config{}

	// env defaults
	cfg.AppEnv = getEnv("APP_ENV", "dev")
	cfg.Server.Addr = getEnv("SERVER_ADDR", ":8080")
	cfg.Log.Level = getEnv("LOG_LEVEL", "info")
	cfg.Log.Format = getEnv("LOG_FORMAT", "text")
	cfg.PG.URL = getEnv("POSTGRES_URL", "postgres://postgres:postgres@localhost:5432/app?sslmode=disable")
	cfg.PG.MaxOpenConns = getInt("PG_MAX_OPEN", 10)
	cfg.PG.MaxIdleConns = getInt("PG_MAX_IDLE", 5)

	// Redis
	cfg.Redis.Addr = getEnv("REDIS_ADDR", "")
	cfg.Redis.Password = getEnv("REDIS_PASSWORD", "")
	cfg.Redis.DB = getInt("REDIS_DB", 0)

	// RabbitMQ
	cfg.MQ.URL = getEnv("RABBITMQ_URL", "")

	// Elasticsearch
	cfg.ES.Addrs = getEnv("ES_ADDRS", "")
	cfg.ES.Username = getEnv("ES_USERNAME", "")
	cfg.ES.Password = getEnv("ES_PASSWORD", "")

	cfg.Apollo.Enable = getBool("APOLLO_ENABLE", false)
	cfg.Apollo.AppID = getEnv("APOLLO_APP_ID", "")
	cfg.Apollo.Cluster = getEnv("APOLLO_CLUSTER", "default")
	cfg.Apollo.Namespace = getEnv("APOLLO_NAMESPACE", "application")
	cfg.Apollo.Addrs = getEnv("APOLLO_ADDRS", "")
	cfg.Apollo.AccessKey = getEnv("APOLLO_ACCESS_KEY", "")

	store := NewStore(cfg)

	if cfg.Apollo.Enable {
		closer, err := overrideFromApollo(cfg, store)
		if err != nil {
			log.Printf("apollo override failed: %v", err)
			return cfg, store, closer, err
		}
		return cfg, store, closer, nil
	}

	return cfg, store, nil, nil
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func getBool(key string, def bool) bool {
	if v := os.Getenv(key); v != "" {
		if b, err := strconv.ParseBool(v); err == nil {
			return b
		}
	}
	return def
}
