# Repository Guidelines

## Project Structure & Module Organization
- `cmd/server/main.go`: API entrypoint (Fiber), wiring for Ent, Redis, MQ, ES.
- `internal/`: application modules
  - `config/` (env + optional Apollo), `db/` (Ent + pgx), `httpx/` (routes, middleware, errors, health), `logx/`, `redisx/`, `mqx/`, `server/` (listener).
- `ent/`: Ent ORM (schemas in `ent/schema`, generated code in `ent/*`). Run generators after schema changes.
- `docs/`: Swagger artifacts (`docs.go`, `swagger.{json,yaml}`).
- `tools/tools.go`: tool deps (e.g., ent codegen). `.air.toml` for live reload.

## Build, Test, and Development Commands
- `make tidy`: sync Go modules.
- `make gen`: run `go generate` (Ent codegen).
- `make run`: start server (`go run ./cmd/server`).
- `make build`: build binary to `bin/server`.
- `make dev`: live reload via Air (reads `.air.toml`).
- `make lint`: run `golangci-lint`.
- `make test`: unit/E2E tests.
- `make cover`: show coverage summary.
- `make test-integration`: integration tests (Testcontainers, `-tags=integration`).
Notes: Compose targets expect a `docker-compose.yml` if you add one locally.

## Coding Style & Naming Conventions
- Use `gofmt`/`goimports`; default Go tab indentation.
- Follow Go naming: packages lower-case, exported identifiers `CamelCase`.
- Keep HTTP handlers and middleware in `internal/httpx`.
- Run `make lint`; `golangci-lint` enables `govet`, `staticcheck`, `revive`, etc.

## Testing Guidelines
- Framework: standard `go test`.
- Unit/E2E: `_test.go` colocated (see `internal/httpx/*_test.go`).
- Integration tests: guarded by `//go:build integration` (see `internal/db/db_integration_test.go`). Run with `make test-integration`.
- Aim to cover routing envelopes, config validation, and DB operations.

## Commit & Pull Request Guidelines
- Commits: follow Conventional Commits (e.g., `feat:`, `fix:`, `chore:`). Keep messages scoped and imperative.
- PRs: include
  - Purpose and scope, linked issues.
  - Test evidence (commands/coverage). For API changes, update Swagger in `docs/` (e.g., `swag init -g cmd/server/main.go -o ./docs`).
  - For Ent schema changes, run `make gen` and commit generated code.

## Security & Configuration Tips
- Config via env (`.env`) and optional Apollo overrides. Key vars: `POSTGRES_URL`, `SERVER_ADDR`, `LOG_LEVEL/FORMAT`, Redis/MQ/ES creds.
- Do not commit secrets. Prefer local `.env`; use Apollo in non-dev.
- Some config changes require restart (e.g., `server.addr`, `pg.url`); pool sizes update live.

