package config

import (
	"log"
	"strconv"

	agollo "github.com/apolloconfig/agollo/v4"
	apconf "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
    "github.com/samber/lo"
)

// overrideFromApollo starts Apollo client and overrides config values if present.
// Returns a closer to stop the Apollo client.
func overrideFromApollo(cfg *Config, store *Store) (func(), error) {
	if cfg.Apollo.Addrs == "" || cfg.Apollo.AppID == "" {
		log.Println("apollo: missing APOLLO_ADDRS or APOLLO_APP_ID; skip")
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
    if v, err := cache.Get("app.env"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.AppEnv = s
        }
    }
    if v, err := cache.Get("server.addr"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.Server.Addr = s
        }
    }
    if v, err := cache.Get("log.level"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.Log.Level = s
        }
    }
    if v, err := cache.Get("log.format"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.Log.Format = s
        }
    }
    if v, err := cache.Get("pg.url"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.PG.URL = s
        }
    }
    if v, err := cache.Get("pg.max_open"); err == nil {
        if s, _ := v.(string); s != "" {
            if n, err := strconv.Atoi(s); err == nil {
                cfg.PG.MaxOpenConns = n
            }
        }
    }
    if v, err := cache.Get("pg.max_idle"); err == nil {
        if s, _ := v.(string); s != "" {
            if n, err := strconv.Atoi(s); err == nil {
                cfg.PG.MaxIdleConns = n
            }
        }
    }
    // Redis
    if v, err := cache.Get("redis.addr"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.Redis.Addr = s
        }
    }
    if v, err := cache.Get("redis.password"); err == nil {
        if s, _ := v.(string); true {
            cfg.Redis.Password = s
        }
    }
    if v, err := cache.Get("redis.db"); err == nil {
        if s, _ := v.(string); s != "" {
            if n, err := strconv.Atoi(s); err == nil {
                cfg.Redis.DB = n
            }
        }
    }
    // MQ
    if v, err := cache.Get("mq.url"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.MQ.URL = s
        }
    }
    // ES
    if v, err := cache.Get("es.addrs"); err == nil {
        if s, _ := v.(string); s != "" {
            cfg.ES.Addrs = s
        }
    }
    if v, err := cache.Get("es.username"); err == nil {
        if s, _ := v.(string); true {
            cfg.ES.Username = s
        }
    }
    if v, err := cache.Get("es.password"); err == nil {
        if s, _ := v.(string); true {
            cfg.ES.Password = s
        }
    }
}

type changeLogger struct {
	ns     string
	client agollo.Client
	store  *Store
}

func (c *changeLogger) OnChange(e *storage.ChangeEvent) {
    log.Printf("apollo change: namespace=%s, changes=%d", e.Namespace, len(e.Changes))
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
func (c *changeLogger) OnNewestChange(e *storage.FullChangeEvent) {
    // Apply overrides directly using latest cache snapshot.
    cur := c.store.Get()
    next := cloneConfig(cur)
    applyApolloOverrides(c.client, c.ns, next)
    _ = c.store.UpdateValidated(next, map[string]bool{"apollo.newest": true})
}
