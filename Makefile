SHELL := /bin/sh

.PHONY: tidy gen run build dev lint test cover test-integration compose-up compose-down

tidy:
	go mod tidy

gen:
	go generate ./...

run:
	go run ./cmd/server

build:
	go build -o bin/server ./cmd/server

dev:
	air -c .air.toml

lint:
	golangci-lint run

test:
	go test ./...

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -n 1

# Integration tests require Docker; guarded by build tag
test-integration:
	go test -tags=integration ./...

compose-up:
	docker compose up -d

compose-down:
	docker compose down -v
