.PHONY: build test migrate docker-up docker-down tag-next tag-patch tag-minor tag-major

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

tag-next:
	@if [ -z "$(TYPE)" ]; then \
		echo "TYPE is required (patch|minor|major)"; \
		echo "example: make tag-next TYPE=patch"; \
		exit 2; \
	fi
	@./scripts/tag-next-version.sh "$(TYPE)"

tag-patch:
	@./scripts/tag-next-version.sh patch

tag-minor:
	@./scripts/tag-next-version.sh minor

tag-major:
	@./scripts/tag-next-version.sh major
