package config

import (
	"log"
	"strconv"

	agollo "github.com/apolloconfig/agollo/v4"
	apconf "github.com/apolloconfig/agollo/v4/env/config"
	"github.com/apolloconfig/agollo/v4/storage"
)

// overrideFromApollo starts Apollo client and overrides config values if present.
// Returns a closer to stop the Apollo client.
func overrideFromApollo(cfg *Config, store *Store) (func(), error) {
	if cfg.Apollo.Addrs == "" || cfg.Apollo.AppID == "" {
		log.Println("apollo: missing APOLLO_ADDRS or APOLLO_APP_ID; skip")
		return nil, nil
	}

	ns := cfg.Apollo.Namespace
	if ns == "" {
		ns = "application"
	}

	appCfg := &apconf.AppConfig{
		AppID:              cfg.Apollo.AppID,
		Cluster:            cfg.Apollo.Cluster,
		NamespaceName:      ns,
		ApolloConfigServer: cfg.Apollo.Addrs, // 支持逗号分隔
		Secret:             cfg.Apollo.AccessKey,
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
	if v, ok := cache.Get("app.env"); ok {
		if s, _ := v.(string); s != "" {
			cfg.AppEnv = s
		}
	}
	if v, ok := cache.Get("server.addr"); ok {
		if s, _ := v.(string); s != "" {
			cfg.Server.Addr = s
		}
	}
	if v, ok := cache.Get("log.level"); ok {
		if s, _ := v.(string); s != "" {
			cfg.Log.Level = s
		}
	}
	if v, ok := cache.Get("log.format"); ok {
		if s, _ := v.(string); s != "" {
			cfg.Log.Format = s
		}
	}
	if v, ok := cache.Get("pg.url"); ok {
		if s, _ := v.(string); s != "" {
			cfg.PG.URL = s
		}
	}
	if v, ok := cache.Get("pg.max_open"); ok {
		if s, _ := v.(string); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				cfg.PG.MaxOpenConns = n
			}
		}
	}
	if v, ok := cache.Get("pg.max_idle"); ok {
		if s, _ := v.(string); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				cfg.PG.MaxIdleConns = n
			}
		}
	}
	// Redis
	if v, ok := cache.Get("redis.addr"); ok {
		if s, _ := v.(string); s != "" {
			cfg.Redis.Addr = s
		}
	}
	if v, ok := cache.Get("redis.password"); ok {
		if s, _ := v.(string); true {
			cfg.Redis.Password = s
		}
	}
	if v, ok := cache.Get("redis.db"); ok {
		if s, _ := v.(string); s != "" {
			if n, err := strconv.Atoi(s); err == nil {
				cfg.Redis.DB = n
			}
		}
	}
	// MQ
	if v, ok := cache.Get("mq.url"); ok {
		if s, _ := v.(string); s != "" {
			cfg.MQ.URL = s
		}
	}
	// ES
	if v, ok := cache.Get("es.addrs"); ok {
		if s, _ := v.(string); s != "" {
			cfg.ES.Addrs = s
		}
	}
	if v, ok := cache.Get("es.username"); ok {
		if s, _ := v.(string); true {
			cfg.ES.Username = s
		}
	}
	if v, ok := cache.Get("es.password"); ok {
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
func (c *changeLogger) OnNewNamespace(e *storage.NewNamespaceEvent) {
	log.Printf("apollo new namespace: %s", e.Namespace)
}
func (c *changeLogger) OnDelete(e *storage.DeleteEvent) {
	log.Printf("apollo delete namespace: %s", e.Namespace)
}
