SHELL := /bin/sh

ifneq (,$(wildcard .env))
include .env
export
endif

MIGRATE_VERSION ?= v4.19.1
MIGRATE_PACKAGE := github.com/golang-migrate/migrate/v4/cmd/migrate@$(MIGRATE_VERSION)
MIGRATE := go run -tags 'postgres' $(MIGRATE_PACKAGE)

.PHONY: migrate-up migrate-down migrate-version migrate-force migrate-create require-database-url

require-database-url:
	@test -n "$(DATABASE_URL)" || (echo "DATABASE_URL is required. Create or load .env before running migrations." >&2; exit 1)

migrate-up: require-database-url
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" up

migrate-down: require-database-url
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" down 1

migrate-version: require-database-url
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" version

migrate-force: require-database-url
	@test -n "$(version)" || (echo "version is required. Example: make migrate-force version=1" >&2; exit 1)
	@$(MIGRATE) -path migrations -database "$(DATABASE_URL)" force "$(version)"

migrate-create:
	@test -n "$(name)" || (echo "name is required. Example: make migrate-create name=add_room_destroyed_at" >&2; exit 1)
	@$(MIGRATE) create -ext sql -dir migrations -seq "$(name)"
