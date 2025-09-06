## Fiber + Ent + Apollo 配置中心 + Postgres 脚手架

一个开箱即用的 Go 脚手架，集成了：
- Web 框架：Fiber
- ORM：Ent（含示例 Schema）
- 配置中心：携程 Apollo（可选，支持本地 .env 覆盖）
- 数据库：PostgreSQL（示例使用 pgx 驱动）

### 快速开始

1) 准备环境
- Go 1.21+
- Docker（用于本地启动 Postgres）

2) 启动 Postgres

```bash
docker compose -f docker-compose.yml up -d
```

默认数据库连接：`postgres://postgres:postgres@localhost:5432/app?sslmode=disable`

3) 配置环境变量

复制 `.env.example` 为 `.env`，根据需要修改：

```bash
cp .env.example .env
```

关键变量：
- `APOLLO_ENABLE`：是否启用 Apollo（默认 false）
- `POSTGRES_URL`：数据库连接串

4) 生成 Ent 代码

首次使用需要生成 Ent 代码（修改了 `ent/schema` 后也需要重新生成）：

```bash
cd fiber-ent-apollo-pg
go mod tidy
go generate ./...
```

5) 运行服务

```bash
go run ./cmd/server
```

启动后可访问：
- 健康检查：`GET http://localhost:8080/health`
- 用户列表：`GET http://localhost:8080/users`
- 创建用户：`POST http://localhost:8080/users`，Body: `{ "name": "Alice" }`
- 文章列表：`GET http://localhost:8080/posts?limit=20&offset=0`
- 创建文章：`POST http://localhost:8080/posts`，Body: `{ "title": "Hello", "content": "...", "user_id": 1 }`
 - 搜索文章（ES）：`GET http://localhost:8080/search/posts?q=hello&limit=20&offset=0`

#### 快速验证（cURL 示例）
- 健康检查
  - `curl -s http://localhost:8080/health`
- 创建用户并列出
  - `curl -s -X POST http://localhost:8080/users -H "Content-Type: application/json" -d '{"name":"alice"}'`
  - `curl -s "http://localhost:8080/users?limit=10&offset=0&sort=created_at:desc"`
- 创建文章并列表（需替换 user_id）
  - `curl -s -X POST http://localhost:8080/posts -H "Content-Type: application/json" -d '{"title":"hello","content":"world","user_id":1}'`
  - `curl -s "http://localhost:8080/posts?limit=10&offset=0&sort=created_at:desc"`
  - 光标模式：`curl -s "http://localhost:8080/posts?mode=cursor&sort=id:asc&limit=10"`

### Apollo 配置中心

当 `APOLLO_ENABLE=true` 时，服务会连接 Apollo 并读取以下 Keys（在命名空间 `APOLLO_NAMESPACE` 中）：
- `app.env`：运行环境
- `server.addr`：服务地址（如 `:8080`）
- `log.level`：日志级别（debug/info/warn/error）
- `log.format`：日志格式（text/json）
- `pg.url`：Postgres 连接串
- `pg.max_open`：最大连接数
- `pg.max_idle`：最大空闲连接数

说明：
- 启动时会优先读取 `.env`，若启用 Apollo 则用 Apollo 值覆盖。
- 变更监听与“回滚”策略：配置变更会先通过验证器（例如 `PG_MAX_IDLE <= PG_MAX_OPEN`），若失败则拒绝更新并保持旧值；成功后触发 Watcher 执行副作用（如更新连接池）。支持运行时热更新：`pg.max_open`/`pg.max_idle` 立即生效；`log.level`/`log.format` 立即重载；`server.addr`、`pg.url` 提示重启。

### 项目结构

```
fiber-ent-apollo-pg/
├── cmd/server/main.go         # 入口
├── internal/
│   ├── config/                # 配置加载（env + Apollo）
│   │   ├── store.go           # 运行时配置存储与监听
│   ├── db/                    # DB 连接（Ent + pgx）
│   └── httpx/                 # Fiber 路由与处理器
│       ├── errors.go          # 统一错误响应（含错误码映射）
│       ├── middleware.go      # 中间件与结构化访问日志
│       ├── paging.go          # 通用分页解析/编码游标
│       ├── sort.go            # 排序白名单映射
│
├── internal/logx/             # 统一日志（基于 slog）
├── internal/redisx/           # Redis 客户端
├── internal/mqx/              # MQ（RabbitMQ）发布端
├── internal/esx/              # Elasticsearch 客户端与搜索/索引示例
├── internal/server/           # Listener 抽象（支持 systemd socket 激活）
├── ent/
│   ├── generate.go            # go:generate 入口
│   └── schema/                # Ent Schema（示例 User）
├── tools/tools.go             # ent 工具依赖
├── docker-compose.yml         # 本地 Postgres
├── .env.example               # 环境变量模板
└── README.md
```

### 常用命令

```bash
# 生成 Ent 代码（修改 schema 后执行）
go generate ./...

# 整体运行
go run ./cmd/server

# Lint 与测试
make lint           # 运行 golangci-lint
make test           # 运行单测与 e2e（不含集成）
make cover          # 输出覆盖率汇总

# 集成测试（需要 Docker，本项目使用 Testcontainers）
make test-integration  # -tags=integration 运行集成测试

# 仅本地数据库
docker compose up -d

# 使用 Makefile（可选）
make up        # 启动 Postgres
make gen       # 生成 Ent 代码
make run       # 运行服务
make down      # 关闭 Postgres
make build     # 构建二进制
make restart   # 构建并重启（脚本写入 PID 文件）

# 或使用 Task（需要安装 go-task）
task up
task gen
task run
task down
task build
task restart
```

### 备注

- 示例中启用了 Ent 的自动迁移（`client.Schema.Create`），生产环境请根据需要调整。
- 若需在 Apollo 中集中管理所有配置，可删去 `.env` 并把变量迁移到 Apollo。
- Apollo 为可选依赖；关闭时，服务完全由 `.env` 驱动。
- 已内置 Fiber 中间件：`recover`、`requestid`、`logger`、`cors`。
- 统一日志使用 Go 1.21 标准库 `log/slog`，支持 `text/json` 输出和动态调整级别。
- 脚本：`scripts/run.sh`（Linux/macOS）与 `scripts/restart.ps1`（Windows）可用于优雅重启（先 TERM 旧进程再启动新进程）。

### Redis / MQ / ES 配置

- Redis：`REDIS_ADDR`、`REDIS_PASSWORD`、`REDIS_DB`（可选）
- RabbitMQ：`RABBITMQ_URL`（如 `amqp://user:pass@host:5672/`）
- Elasticsearch：`ES_ADDRS`（逗号分隔）、`ES_USERNAME`、`ES_PASSWORD`

集成行为：
- 创建文章时：
  - MQ 发布事件：`routingKey=post.created`，payload 包含 `id`、`user_id`、`title`
  - ES 索引文档：索引 `posts`，字段：`id/title/content/user_id/created_at`
 - 搜索文章：`GET /search/posts?q=...` 基于 ES 的 multi_match 查询

### 错误响应规范

- 统一结构：
  - `code`: 业务错误码（如 `E_INVALID_PARAM`/`E_NOT_FOUND`/`E_INTERNAL`）
  - `message`: 错误信息
  - `details`: 可选，附加信息
  - `request_id`: 请求 ID（来自 `requestid` 中间件）
- 示例：
```
{
  "code": "E_INVALID_PARAM",
  "message": "invalid sort",
  "details": "name:sideways",
  "request_id": "3a2e..."
}
```

### 访问日志

- 字段：`method`、`path`、`status`、`latency_ms`、`ip`、`ua`、`request_id`
- 输出由 `internal/httpx/middleware.go` 中的结构化日志中间件完成，默认 `text`，可切换为 `json`。

### 零停机重启（可选）

- 提供 systemd Socket Activation 支持（Linux）：
  - 单元文件示例：`deploy/systemd/fiber-ent-apollo-pg.socket` 与 `deploy/systemd/fiber-ent-apollo-pg.service`
  - 启用：
    ```bash
    sudo cp -r deploy/systemd/* /etc/systemd/system/
    sudo systemctl daemon-reload
    sudo systemctl enable --now fiber-ent-apollo-pg.socket
    sudo systemctl enable --now fiber-ent-apollo-pg.service
    ```
  - 应用将从 systemd 注入的 FD 监听端口（设置 `SOCKET_ACTIVATION=1`）。
- 非 Linux 或无需零停机：使用 `make restart` 或 `scripts/restart.ps1` 即可。
### ����ģʽ��Air ���ط���

- ��װ Air��һ���Σ� `go install github.com/cosmtrek/air@latest`
- ��������
  - `make dev` �� `task dev` ��ֱ�� `air -c .air.toml`
- �Զ�Ϊ��
  - ��� `go generate ./ent` ���´��� Ent ���룻
  - ���������޸� `go/mod/sum/env/yaml` ʱ���Զ�����
  - ����Ŀ¼�� `tmp/server`

### 成功响应规范

- 统一结构：
  - `code`: 固定 `OK`
  - `message`: 固定 `success`
  - `data`: 业务数据（对象或数组）
  - `request_id`: 请求 ID
  - 可选 `meta`: 列表分页信息
- 列表响应示例：
```
{
  "code": "OK",
  "message": "success",
  "data": [ {"id":1,"name":"Alice"} ],
  "meta": { "limit":20, "offset":0, "count":1 },
  "request_id": "3a2e..."
}
```

### 分页与滚动

- Offset 模式（默认）：参数 `limit`、`offset`；响应 `meta` 包含 `next_offset`、`has_more`。
- 可选总数：`with_total=true` 时返回 `meta.total`（仅 offset 模式，为避免重查询开销）。
- Cursor 模式（兼容 infinite scroll）：
  - 参数：`cursor`（支持数字 id，或 base64 编码的 `{"id":number,"ts":RFC3339Nano}`），并指定 `sort=id:asc`（未指定时默认强制为 `id:asc`）。
  - 响应 `meta`：`cursor`、`next_cursor`、`cursor_enc`、`next_cursor_enc`、`has_more`、`mode=cursor`。
  - 适用接口：`GET /users`、`GET /posts`。

- 固定快照（强一致滚动窗口）：
  - 用途：滚动加载期间避免新写入导致的穿插/重复/漏读。
  - 使用方式：
    1) 首次请求带 `fixed=true`（可选 `limit`，可选过滤参数）。服务端会按 `created_at desc, id desc` 排序，并在响应 `meta.snapshot` 返回快照时间戳。
    2) 后续请求带上：`snapshot=<上次meta.snapshot>`，并使用游标：
       - 简洁方式：`cursor=<上次 meta.next_cursor_enc>`（推荐）
       - 兼容方式：`cursor=<上次 meta.next_cursor>&cursor_ts=<上次 meta.next_cursor_ts>`
    3) 直到 `meta.has_more=false`，快照窗口读取完毕。
  - 也可自行指定 `snapshot=<RFC3339Nano>`，在任意时间点冻结快照。
  - 注意：固定快照模式下排序强制为 `created_at desc, id desc`，过滤条件始终包含 `created_at <= snapshot`；keyset 条件为 `(created_at < cursor_ts) OR (created_at = cursor_ts AND id < cursor)`。
