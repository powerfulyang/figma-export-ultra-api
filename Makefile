SHELL := /bin/sh

.PHONY: tidy gen run build dev lint test test-unit cover test-integration compose-up compose-down swagger-gen swagger-install swagger-serve swagger-validate

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

# Unit tests focusing on httpx subpackages (exclude e2e tests package)
test-unit:
	GO111MODULE=on go test $(shell go list ./internal/httpx/... | grep -v '/tests$$')

cover:
	go test -coverprofile=coverage.out ./... && go tool cover -func=coverage.out | tail -n 1

# Integration tests require Docker; guarded by build tag
test-integration:
	go test -tags=integration ./...

compose-up:
	docker compose up -d

compose-down:
	docker compose down -v

# Generate Swagger documentation
swagger-gen:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o ./docs

# Install swag tool globally (optional)
swagger-install:
	go install github.com/swaggo/swag/cmd/swag@latest

# Serve Swagger UI (requires swag to be installed)
swagger-serve:
	swag init -g cmd/server/main.go -o ./docs && \
	echo "Swagger docs generated. Start your server with 'make run' and visit http://localhost:8080/swagger/index.html"

# Validate Swagger documentation
swagger-validate:
	go run github.com/swaggo/swag/cmd/swag@latest init -g cmd/server/main.go -o ./docs --parseVendor
