.PHONY: build test migrate docker-up docker-down

GO := go
BIN := bin/kao

build:
	$(GO) build -o $(BIN) ./cmd/kao

test:
	$(GO) test ./...

docker-up:
	docker compose up -d

docker-down:
	docker compose down

migrate:
	docker compose exec -T postgres psql -U kao -d kao < migrations/001_init.sql
