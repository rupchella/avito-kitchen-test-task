GO := go

MIGRATIONS_DIR ?= migrations
MIGRATION_NAME ?= migration
DATABASE_DRIVER ?= postgres
DATABASE_URL ?= postgres://kitchen:kitchen@localhost:5432/kitchen?sslmode=disable

.PHONY: fmt vet test check run-kitchen run-restaurant migrate-create migrate-validate migrate-up migrate-down migrate-status

fmt:
	$(GO) fmt ./...

vet:
	$(GO) vet ./...

test:
	$(GO) test ./...

check: fmt vet test

run-kitchen:
	$(GO) run ./cmd/kitchen

run-restaurant:
	$(GO) run ./cmd/restaurant

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