# Chat Service

Backend scaffold for the chat system. This service is intentionally focused on the Go backend only and follows Clean Architecture boundaries so endpoint behavior can be implemented in focused follow-up tasks.

## Stack

- Go 1.25.10
- Chi HTTP router
- GORM
- PostgreSQL
- Redis
- `log/slog` structured logging
- Environment-variable configuration

## Project Layout

```text
cmd/api                  application entrypoint
internal/config          environment configuration
internal/bootstrap       dependency wiring
internal/domain          entities, errors, repository and pub/sub interfaces
internal/usecase         application use cases
internal/infrastructure  PostgreSQL and Redis adapters
internal/delivery/http   router, handlers, DTOs, middleware, responses
migrations               SQL migration files
pkg                      small shared packages
```

## Team Guide

Read [docs/README.md](docs/README.md) before contributing. It defines the shared team rules for branch naming, commit naming, PR expectations, Go naming, migration naming, CI/CD, security, and the pre-PR checklist.

For backend unit test work, read [docs/unit-testing/README.md](docs/unit-testing/README.md). It defines the testing boundaries, file placement, fake dependency rules, and validation checklist for this service.

## Environment

Copy `.env.example` and export the values before running locally:

```sh
cp .env.example .env
set -a
. ./.env
set +a
```

Required variables:

| Variable | Description |
| --- | --- |
| `APP_ENV` | Runtime environment name |
| `APP_PORT` | HTTP port |
| `APP_NAME` | Service name |
| `DATABASE_URL` | PostgreSQL connection URL |
| `REDIS_ADDR` | Redis host and port |
| `REDIS_PASSWORD` | Redis password, empty for local development |
| `REDIS_DB` | Redis database number |
| `CORS_ALLOWED_ORIGINS` | Comma-separated allowed browser origins |
| `ROOM_DEFAULT_TTL_MINUTES` | Default room lifetime in minutes |
| `MAX_REQUEST_BODY_BYTES` | Maximum accepted HTTP request body size in bytes |
| `POSTGRES_USER` | Local Docker Compose PostgreSQL user |
| `POSTGRES_PASSWORD` | Local Docker Compose PostgreSQL password |
| `POSTGRES_DB` | Local Docker Compose PostgreSQL database |
| `POSTGRES_PORT` | Host port mapped to local Docker Compose PostgreSQL |
| `REDIS_PORT` | Host port mapped to local Docker Compose Redis |
| `COMPOSE_DATABASE_URL` | PostgreSQL URL injected into the app container by Docker Compose |
| `COMPOSE_REDIS_ADDR` | Redis address injected into the app container by Docker Compose |

`DATABASE_URL` and `REDIS_ADDR` are intended for running the service directly on the host with `go run ./cmd/api`. `COMPOSE_DATABASE_URL` and `COMPOSE_REDIS_ADDR` are used by Docker Compose because containers connect to service names such as `postgres` and `redis` instead of `localhost`.

Do not hardcode server connection settings, credentials, or ports in Docker files. Keep them in environment variables and keep real `.env` files out of Git.

## Run Locally

Start PostgreSQL and Redis:

```sh
docker compose --env-file .env up -d postgres redis
```

Apply migrations with the project Makefile. This is the only supported local migration path:

```sh
make migrate-up
```

The Makefile uses `golang-migrate` with the `DATABASE_URL` value from `.env`. Do not apply migrations manually through TablePlus for normal development, because it is easy to run the SQL against a different local PostgreSQL instance, port, database, or schema.

Verify the migration version when needed:

```sh
make migrate-version
```

Run the service:

```sh
go run ./cmd/api
```

Health check:

```sh
curl http://localhost:8080/health
```

Expected response:

```json
{"status":"ok"}
```

## Docker Compose

Build and run the app with its local dependencies:

```sh
docker compose --env-file .env up --build
```

The app listens on `http://localhost:8080`.

## Database Migrations

Migration files live in `migrations/` and are executed with `golang-migrate`.

Supported commands:

```sh
make migrate-up
make migrate-down
make migrate-version
make migrate-force version=1
make migrate-create name=add_room_destroyed_at
```

Rules:

- `DATABASE_URL` is required. The service no longer falls back to a default database URL.
- Load `.env` or keep `.env` in the project root before running migration commands.
- Run migrations against the same database that the backend uses.
- Use TablePlus only to inspect data, not as the standard migration runner.
- Do not edit old migration files after they are merged and used by other developers.

## CI/CD

GitHub Actions workflows live in `.github/workflows/`.

- `CI`: runs on pull requests and pushes to `main`; validates formatting, modules, `go vet`, race-enabled tests, API build, Docker Compose config, Docker image build, and `govulncheck`.
- `CodeQL`: runs on pull requests, pushes to `main`, a weekly schedule, and manual dispatch for static security analysis.
- `Container Publish`: runs on pushes to `main`, semantic version tags, and manual dispatch; builds and publishes the Docker image to GitHub Container Registry.

The current CD workflow publishes a container image only. It does not deploy to a production environment. A real production deployment should be added later when the target platform, environment protection rules, required secrets, rollback strategy, and approval process are defined.

## API Routes Registered

- `GET /health`
- `POST /api/v1/client-alias`
- `POST /api/v1/rooms`
- `GET /api/v1/rooms/status?roomId=...`
- `POST /api/v1/messages`
- `GET /api/v1/messages?roomId=...`
- `GET /api/v1/messages/stream?roomId=...`

## Intentional TODOs

The initial scaffold has been extended with minimal alias, room, and message behavior. The service is still not a complete production chat system. Follow-up endpoint tasks should harden:

- Unit and integration tests for alias, room, message, and SSE behavior.
- A transactional outbox or equivalent delivery guarantee for message persistence plus Redis publish.
- Redis-based room destroy events so SSE streams close immediately without relying on periodic status checks.
- Expired-room cleanup and operational jobs.
- Friendly alias generation rules if product wants human-readable names instead of deterministic technical aliases.

The database schema, repositories, Redis pub/sub adapter, HTTP handlers, validation, error mapping, middleware, and app wiring are ready for those follow-up implementations.
