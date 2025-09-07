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

	// 启动 PostgreSQL 容器，添加等待策略
	pg, err := postgres.RunContainer(ctx,
		postgres.WithDatabase("app"),
		postgres.WithUsername("postgres"),
		postgres.WithPassword("postgres"),
		postgres.WithSQLDriver("pgx"),
		postgres.BasicWaitStrategies(),
	)
	if err != nil {
		t.Fatalf("start postgres: %v", err)
	}
	t.Cleanup(func() { _ = pg.Terminate(ctx) })

	// 获取连接信息
	host, err := pg.Host(ctx)
	if err != nil {
		t.Fatalf("get container host: %v", err)
	}
	port, err := pg.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatalf("get container port: %v", err)
	}

	// 构建数据库连接字符串
	dsn := fmt.Sprintf("postgres://postgres:postgres@%s:%s/app?sslmode=disable", host, port.Port())

	// 配置数据库连接
	cfg := &config.Config{}
	cfg.PG.URL = dsn
	cfg.PG.MaxOpenConns = 5
	cfg.PG.MaxIdleConns = 2

	// 打开数据库连接
	c, closeFn, err := Open(cfg)
	if err != nil {
		t.Fatalf("open db: %v", err)
	}
	defer closeFn()

	// 运行数据库模式迁移
	ctx2, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()

	if err := c.Schema.Create(ctx2); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	// 测试数据库连接是否正常
	if _, err := c.User.Query().Count(ctx2); err != nil {
		t.Fatalf("ent ping: %v", err)
	}

	// 测试创建用户
	user, err := c.User.Create().
		SetUsername("test_user").
		SetIsActive(true).
		SetCreatedAt(time.Now()).
		SetUpdatedAt(time.Now()).
		Save(ctx2)
	if err != nil {
		t.Fatalf("create user: %v", err)
	}

	// 验证用户创建成功
	if user.Username != "test_user" {
		t.Errorf("expected user name 'test_user', got '%s'", user.Username)
	}

	// 测试查询用户
	count, err := c.User.Query().Count(ctx2)
	if err != nil {
		t.Fatalf("count users: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 user, got %d", count)
	}

	t.Logf("Database integration test passed successfully")
}
