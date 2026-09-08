GO := go

MIGRATIONS_DIR ?= migrations
MIGRATION_NAME ?= migration
DATABASE_DRIVER ?= postgres
DATABASE_URL ?= postgres://kitchen:kitchen@localhost:5432/kitchen?sslmode=disable

.PHONY: fmt vet lint lint-fix test check run-kitchen run-restaurant \
	up down logs migrate-create migrate-validate migrate-up migrate-down \
	migrate-status

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

lint:
	$(GO) tool golangci-lint run ./...

lint-fix:
	$(GO) tool golangci-lint run --fix ./...

test:
	$(GO) test ./...

check: fmt vet lint test

run-kitchen:
	$(GO) run ./cmd/kitchen

run-restaurant:
	$(GO) run ./cmd/restaurant

up:
	docker compose up --build -d

down:
	docker compose down -v

logs:
	docker compose logs -f kitchen restaurant

migrate-create:
	$(GO) tool goose -s -dir $(MIGRATIONS_DIR) create $(MIGRATION_NAME) sql

migrate-validate:
	$(GO) tool goose -dir $(MIGRATIONS_DIR) validate

migrate-up:
	$(GO) tool goose -dir $(MIGRATIONS_DIR) $(DATABASE_DRIVER) "$(DATABASE_URL)" up

migrate-down:
	$(GO) tool goose -dir $(MIGRATIONS_DIR) $(DATABASE_DRIVER) "$(DATABASE_URL)" down

migrate-status:
	$(GO) tool goose -dir $(MIGRATIONS_DIR) $(DATABASE_DRIVER) "$(DATABASE_URL)" status
