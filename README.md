# Chat Service

Backend scaffold for the chat system. This service is intentionally focused on the Go backend only and follows Clean Architecture boundaries so endpoint behavior can be implemented in focused follow-up tasks.

## Stack

- Go 1.25.9
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
docker compose up -d postgres redis
```

Apply migrations with your migration tool of choice. The SQL files live in `migrations/`.

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
docker compose up --build
```

The app listens on `http://localhost:8080`.

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

Business behavior is not fully implemented in this scaffold. The use cases currently define the application boundaries and return `501 Not Implemented` after handlers validate incoming requests. Follow-up endpoint tasks should implement:

- Stable alias generation and persistence using hashed client identifiers.
- Room join, owner assignment, destroy, leave, status, and TTL rules.
- Message persistence, retrieval, and Redis publish flow.
- SSE stream coordination and room-destroy shutdown behavior.
- Transaction boundaries around room and alias/message mutations.

The database schema, repositories, Redis pub/sub adapter, HTTP handlers, validation, error mapping, middleware, and app wiring are ready for those implementations.
