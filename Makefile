SHELL := /bin/sh

.PHONY: tidy gen run up down build start stop restart dev

tidy:
	go mod tidy

gen:
	go generate ./...

run:
	go run ./cmd/server

up:
	docker compose up -d

down:
	docker compose down -v

build:
	go build -o bin/server ./cmd/server

start:
	bash scripts/run.sh

stop:
	@if [ -f .run/server.pid ]; then kill `cat .run/server.pid` || true; fi

restart: build start

dev:
	air -c .air.toml
