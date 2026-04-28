# Chat Service

Backend scaffold for the chat system. This service is intentionally focused on the Go backend only and follows Clean Architecture boundaries so endpoint behavior can be implemented in focused follow-up tasks.

## Stack

- Go 1.25
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

## API Routes Registered

- `GET /health`
- `POST /api/client-alias`
- `POST /api/rooms`
- `GET /api/rooms/status?roomId=...`
- `POST /api/messages`
- `GET /api/messages?roomId=...`
- `GET /api/messages/stream?roomId=...`

## Intentional TODOs

Business behavior is not fully implemented in this scaffold. The use cases currently define the application boundaries and return `501 Not Implemented` after handlers validate incoming requests. Follow-up endpoint tasks should implement:

- Stable alias generation and persistence using hashed client identifiers.
- Room join, owner assignment, destroy, leave, status, and TTL rules.
- Message persistence, retrieval, and Redis publish flow.
- SSE stream coordination and room-destroy shutdown behavior.
- Transaction boundaries around room and alias/message mutations.

The database schema, repositories, Redis pub/sub adapter, HTTP handlers, validation, error mapping, middleware, and app wiring are ready for those implementations.
