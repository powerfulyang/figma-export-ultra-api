// Package config provides application configuration management using Apollo
package config

import (
	"strconv"

	agollo "github.com/apolloconfig/agollo/v4"
	agcache "github.com/apolloconfig/agollo/v4/agcache"
	apconf "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
	"github.com/samber/lo"
)

// overrideFromApollo starts Apollo client and overrides config values if present.
// Returns a closer to stop the Apollo client.
func overrideFromApollo(cfg *Config, store *Store) (func(), error) {
	if cfg.Apollo.Addrs == "" || cfg.Apollo.AppID == "" {
		configLogger.Sugar().Info("apollo: missing APOLLO_ADDRS or APOLLO_APP_ID; skip")
		return nil, nil
	}

	ns := lo.Ternary(cfg.Apollo.Namespace != "", cfg.Apollo.Namespace, "application")

	appCfg := &apconf.AppConfig{
		AppID:         cfg.Apollo.AppID,
		Cluster:       cfg.Apollo.Cluster,
		NamespaceName: ns,
		IP:            cfg.Apollo.Addrs, // 支持逗号分隔
		Secret:        cfg.Apollo.AccessKey,
	}

	client, err := agollo.StartWithConfig(func() (*apconf.AppConfig, error) { return appCfg, nil })
	if err != nil {
		return nil, err
	}

	// Initial override
	applyApolloOverrides(client, ns, cfg)
	_ = store.UpdateValidated(cfg, map[string]bool{"apollo.init": true})

	// Listen changes: update store with changed keys
	client.AddChangeListener(&changeLogger{ns: ns, client: client, store: store})

	closer := func() {
		// agollo v4 没有公开 Stop 接口，这里保留为空
	}
	return closer, nil
}

func applyApolloOverrides(client agollo.Client, namespace string, cfg *Config) {
	cache := client.GetConfigCache(namespace)
	if cache == nil {
		return
	}

	// Simple string assignments
	setStringFromCache(cache, "app.env", &cfg.AppEnv)
	setStringFromCache(cache, "server.addr", &cfg.Server.Addr)
	setStringFromCache(cache, "log.level", &cfg.Log.Level)
	setStringFromCache(cache, "log.format", &cfg.Log.Format)
	setStringFromCache(cache, "pg.url", &cfg.PG.URL)
	setStringFromCache(cache, "redis.addr", &cfg.Redis.Addr)
	setStringFromCache(cache, "redis.password", &cfg.Redis.Password)
	setStringFromCache(cache, "mq.url", &cfg.MQ.URL)
	setStringFromCache(cache, "es.addrs", &cfg.ES.Addrs)
	setStringFromCache(cache, "es.username", &cfg.ES.Username)
	setStringFromCache(cache, "es.password", &cfg.ES.Password)

	// Integer assignments
	setIntFromCache(cache, "pg.max_open", &cfg.PG.MaxOpenConns)
	setIntFromCache(cache, "pg.max_idle", &cfg.PG.MaxIdleConns)
	setIntFromCache(cache, "redis.db", &cfg.Redis.DB)
}

func setStringFromCache(cache agcache.CacheInterface, key string, target *string) {
	if v, err := cache.Get(key); err == nil {
		if s, _ := v.(string); s != "" {
			*target = s
		}
	}
}

func setIntFromCache(cache agcache.CacheInterface, key string, target *int) {
	if v, err := cache.Get(key); err == nil {
		if s, _ := v.(string); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				*target = n
			}
		}
	}
}

type changeLogger struct {
	ns     string
	client agollo.Client
	store  *Store
}

func (c *changeLogger) OnChange(e *storage.ChangeEvent) {
	configLogger.Sugar().Infof("apollo change: namespace=%s, changes=%d", e.Namespace, len(e.Changes))
	// Build new config based on current and apply overrides
	cur := c.store.Get()
	next := cloneConfig(cur)
	applyApolloOverrides(c.client, c.ns, next)
	changed := map[string]bool{}
	for k := range e.Changes {
		changed[k] = true
	}
	_ = c.store.UpdateValidated(next, changed)
}

// OnNewestChange is required by agollo v4 ChangeListener interface.
func (c *changeLogger) OnNewestChange(_ *storage.FullChangeEvent) {
	// Apply overrides directly using latest cache snapshot.
	cur := c.store.Get()
	next := cloneConfig(cur)
	applyApolloOverrides(c.client, c.ns, next)
	_ = c.store.UpdateValidated(next, map[string]bool{"apollo.newest": true})
}
