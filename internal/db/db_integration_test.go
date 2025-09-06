//go:build integration
// +build integration

package db

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/testcontainers/testcontainers-go/modules/postgres"

	"fiber-ent-apollo-pg/internal/config"
)

func Test_Open_With_PostgresContainer(t *testing.T) {
	ctx := context.Background()
	pg, err := postgres.RunContainer(ctx,
		postgres.WithInitScripts(),
		postgres.WithDatabase("app"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	host, _ := pg.Host(ctx)
	port, _ := pg.MappedPort(ctx, "5432")
	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/app?sslmode=disable", host, port.Port())

	cfg := &config.Config{}
	cfg.PG.URL = dsn
	cfg.PG.MaxOpenConns = 5
	cfg.PG.MaxIdleConns = 2

	c, closeFn, err := Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer closeFn()

	ctx2, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	// simple ping via ent by issuing a trivial query
	if _, err := c.User.Query().Count(ctx2); err != nil {
		t.Fatalf("ent ping: %v", err)
	}
}
