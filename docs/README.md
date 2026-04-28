# Team Development Guide

This document defines the shared engineering rules for `chat-service`. Use it as the team standard for branch names, commit messages, pull requests, file names, package names, function names, database migrations, CI/CD, security, and review checklists.

The goal is to keep the repository easy to read, easy to review, safe to operate, and predictable as the service grows.

## Working Principles

Follow these rules before starting any task:

- Change only the scope required by the task.
- Keep pull requests small enough to review properly.
- Do not change public API contracts unless the task explicitly requires it.
- Do not move folders or rewrite structure without a clear architecture reason.
- Do not put business logic in HTTP handlers.
- Do not put database queries in handlers or use cases directly.
- Do not expose raw client identifiers to other clients.
- Run the relevant validation commands before opening or updating a pull request.

## Branch Naming

Branch names must describe the type of work and the scope.

Standard format:

```text
<type>/<scope>-<short-description>
```

Examples:

```text
feat/chat-service-room-join
feat/chat-service-message-create
fix/chat-service-cors
fix/chat-service-room-destroy-permission
chore/chat-service-docker
test/chat-service-room-usecase
docs/chat-service-team-guide
refactor/chat-service-message-repository
```

### Branch Types

| Type | Use when |
| --- | --- |
| `feat` | Adding a new feature or behavior |
| `fix` | Fixing a bug |
| `docs` | Changing documentation only |
| `test` | Adding or changing tests |
| `refactor` | Restructuring code without changing behavior |
| `chore` | Repository maintenance, Docker, dependency, or config work |
| `perf` | Improving performance |
| `build` | Changing build tooling or build-related dependencies |

### Branch Rules

Good examples:

```text
feat/chat-service-client-alias
fix/chat-service-invalid-room-status
docs/chat-service-naming-guide
```

Bad examples:

```text
update
fix
backend
new-code
work
final
test123
```

Generic branch names make review and historical lookup harder. The name should tell another developer what the branch is for without opening the diff.

## Commit Naming

Use Conventional Commit style:

```text
<type>: <summary>
```

Examples:

```text
feat: implement room join use case
fix: return forbidden for non-owner room destroy
docs: add team development guide
test: add room use case tests
chore: add docker compose for local dependencies
refactor: split message repository mapping
```

### Commit Types

| Type | Use when |
| --- | --- |
| `feat` | Adding new behavior |
| `fix` | Fixing a bug |
| `docs` | Changing documentation |
| `test` | Adding or changing tests |
| `refactor` | Refactoring without changing behavior |
| `chore` | Maintenance work |
| `perf` | Improving performance |
| `build` | Changing build or dependency behavior |

### Commit Rules

One commit should represent one logical change.

Good examples:

```text
feat: implement client alias lookup
test: add client alias use case tests
docs: document branch naming rules
```

Bad examples:

```text
update code
fix bug
changes
wip
final
done
```

If one commit mixes room logic, Docker changes, and README edits, split it into separate commits or separate pull requests when appropriate.

## Pull Request Naming

PR titles must summarize the change clearly.

Recommended format:

```text
<type>: <clear summary>
```

Examples:

```text
feat: implement room join endpoint
fix: prevent non-owner room destroy
docs: add team development guide
test: add message use case tests
```

Every PR description should include:

- Summary: what changed
- Reason: why the change is needed
- Validation: what was checked
- Notes: what is intentionally left out, if anything

Example:

```md
## Summary

- Implement room join use case.
- Persist first joining client as room owner.
- Return destroyed state for destroyed rooms.

## Validation

- go test ./...

## Notes

- Room destroy flow will be implemented in a separate PR.
```

## Review Rules

Every PR into `main` must have at least one review approval unless a repository owner intentionally uses the allowed admin bypass.

Reviewers should check:

- Whether the scope matches the task
- Whether the API contract changed unexpectedly
- Whether layer boundaries are respected
- Whether error responses use the shared format
- Whether test coverage is reasonable for the risk
- Whether raw client identifiers are exposed in responses or logs
- Whether secrets or credentials were committed

Use a Draft PR when the change is not ready to merge.

## CI/CD Rules

Every PR must allow GitHub Actions to finish before merge.

Main workflows:

- `CI`: checks formatting, module files, `go vet`, tests, build, Docker Compose config, Docker image build, and vulnerability scan.
- `CodeQL`: runs static security analysis.
- `Container Publish`: builds and publishes the Docker image to GitHub Container Registry after pushes to `main` or semantic version tags.

Rules:

- Do not merge a failing CI run without understanding the failure.
- If CI fails because of the code change, fix it in the same branch and push again.
- If CI fails because of external infrastructure, document the reason in the PR.
- Current CD publishes a container image only. It does not deploy to production.
- Production deployment requires environment protection, secrets, approval, rollback strategy, and a clear deployment target.
- Never hardcode secrets in workflow files. Use GitHub Secrets or protected environment secrets.

## CI/CD Rules

ทุก PR ต้องปล่อยให้ GitHub Actions ทำงานจนจบก่อน merge

Workflow หลักของ repository นี้:

- `CI`: ตรวจ formatting, module files, `go vet`, tests, build, Docker Compose config, Docker image build และ vulnerability scan
- `CodeQL`: ตรวจ static security analysis
- `Container Publish`: build และ publish Docker image ไป GitHub Container Registry เมื่อมี push เข้า `main` หรือ tag แบบ semantic version

กฎสำคัญ:

- ห้าม merge ถ้า CI fail โดยไม่เข้าใจสาเหตุ
- ถ้า CI fail จาก code change ต้องแก้ใน branch เดิมและ push เพิ่ม
- ถ้า CI fail จาก infrastructure ภายนอก ให้ระบุใน PR ว่าเกิดจากอะไร
- CD ปัจจุบัน publish image เท่านั้น ยังไม่ deploy production จริง
- Production deployment ต้องมี environment protection, secrets, approval และ rollback plan ก่อนเพิ่ม workflow
- ห้ามใส่ secret ลง workflow file โดยตรง ให้ใช้ GitHub Secrets หรือ environment secrets เท่านั้น

## Go Package Naming

Package names must be lowercase, short, and clear.

Good examples:

```text
config
bootstrap
domain
usecase
postgres
redis
handler
middleware
response
validator
logger
```

Bad examples:

```text
Config
UseCase
room_handler
postgresRepository
commonUtils
helpers
```

Rules:

- Use lowercase package names.
- Avoid underscores in package names.
- Avoid broad names such as `common`, `utils`, and `helpers`.
- A package name should describe responsibility, not convenience.

## Go File Naming

Use snake_case file names that describe responsibility.

Good examples:

```text
room_usecase.go
message_usecase.go
alias_handler.go
room_repository.go
message_repository.go
request_id.go
```

Bad examples:

```text
RoomUseCase.go
messageUseCase.go
utils.go
helper.go
all.go
main2.go
```

If a file becomes too large, split it by responsibility.

## Go Type Naming

Exported types use PascalCase.

```go
type RoomUseCase struct {}
type MessageRepository interface {}
type AliasHandler struct {}
```

Private types use camelCase.

```go
type requestIDKey struct {}
type statusRecorder struct {}
```

Interfaces should be named by behavior.

Good examples:

```go
type RoomRepository interface {}
type MessagePubSub interface {}
type MessageSubscription interface {}
```

Bad examples:

```go
type IRoomRepository interface {}
type RoomRepositoryInterface interface {}
type Manager interface {}
```

Do not prefix Go interfaces with `I`.

## Function Naming

Function names must describe the action.

Good examples:

```go
func NewRoomUseCase(...) *RoomUseCase
func FindByRoomID(ctx context.Context, roomID string) (*Room, error)
func MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error
func roomToModel(room *domain.Room) RoomModel
```

Bad examples:

```go
func Do(...)
func Process(...)
func HandleData(...)
func Manage(...)
func Convert(...)
```

Broad function names usually mean the function is doing too much.

## Variable Naming

Short names are acceptable in small scopes, but names must still be meaningful.

Good examples:

```go
ctx := r.Context()
roomID := req.RoomID
identifierHash := idgen.HashIdentifier(req.Identifier)
message := domain.Message{}
```

Bad examples:

```go
x := req.RoomID
data := domain.Message{}
thing := idgen.HashIdentifier(req.Identifier)
resultObj := output
```

Variable names should reflect domain meaning, such as `roomID`, `identifierHash`, `senderName`, and `expiresAt`.

## Error Naming

Shared domain errors use the `Err<Name>` format.

```go
var (
    ErrInvalidInput = errors.New("invalid_input")
    ErrNotFound     = errors.New("not_found")
    ErrForbidden    = errors.New("forbidden")
)
```

Client-facing error messages must be in English and must not expose internal details such as SQL queries, Redis commands, connection strings, or secrets.

## Context Rules

Functions that perform I/O or cross layer boundaries must accept `context.Context` as the first parameter.

Good example:

```go
func (r *RoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error)
```

Bad example:

```go
func (r *RoomRepository) FindByRoomID(roomID string) (*domain.Room, error)
```

Handlers must pass `r.Context()` into use cases.

```go
output, err := h.useCase.GetStatus(r.Context(), roomID)
```

Do not create `context.Background()` inside handlers, use cases, or repositories to replace request context. The exception is a deliberate background job with its own lifecycle.

## API Route Naming

Routes use lowercase, resource-based names.

Business APIs must live under `/api/v1` so a future `/api/v2` can be introduced for breaking changes. `/health` stays unversioned because it is an operational endpoint, not a business API contract.

Use this pattern:

```text
GET    /health
POST   /api/v1/client-alias
POST   /api/v1/rooms
GET    /api/v1/rooms/status
POST   /api/v1/messages
GET    /api/v1/messages
GET    /api/v1/messages/stream
```

Avoid ambiguous routes:

```text
POST /api/v1/do-room
POST /api/v1/action
GET  /api/v1/getMessages
```

New endpoints should follow the existing resource style.

## JSON Field Naming

JSON fields use camelCase to match frontend contracts.

Example:

```json
{
  "identifier": "client-id",
  "roomId": "room-id",
  "senderName": "Display Name",
  "sentAt": "2026-04-25T12:00:00.000Z"
}
```

Go struct fields use PascalCase, while JSON tags use camelCase.

```go
type CreateMessageRequest struct {
    Identifier string `json:"identifier" validate:"required"`
    RoomID     string `json:"roomId" validate:"required"`
    Body       string `json:"body" validate:"required"`
}
```

Do not change JSON field names without checking the frontend contract.

## Database Naming

Table names use plural snake_case.

```text
rooms
room_members
client_aliases
messages
```

Column names use snake_case.

```text
room_id
identifier_hash
owner_identifier_hash
is_destroyed
expires_at
created_at
updated_at
```

Index names should identify the table and columns.

```text
idx_messages_room_sent_at
idx_rooms_destroyed_expires_at
idx_client_aliases_room_id
```

Unique constraint names should start with `uq_`.

```text
uq_room_members_room_identifier
uq_client_aliases_room_identifier
```

## Migration Naming

Migration files use a sequence number and a short description.

```text
000001_create_chat_tables.up.sql
000001_create_chat_tables.down.sql
000002_add_room_destroyed_at.up.sql
000002_add_room_destroyed_at.down.sql
```

Migration rules:

- Every `.up.sql` file must have a matching `.down.sql` file.
- Do not edit old migrations after they are merged, unless they have never been deployed anywhere.
- Migrations must be easy to review.
- Avoid destructive changes without a clear migration plan.
- Production migrations must account for existing data.

## Environment Variable Naming

Environment variables use uppercase snake_case.

```text
APP_ENV
APP_PORT
APP_NAME
DATABASE_URL
REDIS_ADDR
REDIS_PASSWORD
REDIS_DB
CORS_ALLOWED_ORIGINS
ROOM_DEFAULT_TTL_MINUTES
```

Rules:

- Do not commit real secrets.
- `.env.example` may contain sample local values only.
- If a new environment variable is added, update `.env.example` and README.
- Environment variable names should clearly describe the behavior they control.

## Docker And Environment Configuration

Docker files must not hardcode server connection settings, credentials, or ports.

Use environment variables for:

- Application port
- PostgreSQL user, password, database, and host port
- Redis host port
- Application `DATABASE_URL`
- Application `REDIS_ADDR`
- CORS origins
- Room TTL

Local Docker Compose may read these values from a local `.env` file, but the real `.env` file must never be committed. Keep only `.env.example` in Git.

Use separate values when the app runs on the host versus inside Docker Compose:

- Host run: `DATABASE_URL` points to `localhost`, and `REDIS_ADDR` points to `localhost`.
- Compose app run: `COMPOSE_DATABASE_URL` points to the `postgres` service, and `COMPOSE_REDIS_ADDR` points to the `redis` service.

## Layer Boundary Rules

The service has clear layer boundaries.

```text
delivery/http -> usecase -> domain
infrastructure -> domain interfaces
bootstrap -> wires dependencies
```

### Handler

Handlers may:

- Decode requests
- Validate requests
- Call use cases
- Map responses

Handlers must not:

- Query the database
- Call Redis directly
- Implement complex business rules
- Use GORM models

### Usecase

Use cases may:

- Orchestrate business flow
- Call repository interfaces
- Enforce business rules
- Return domain or application errors

Use cases must not:

- Import GORM
- Import Chi
- Import HTTP packages
- Know HTTP DTO types

### Domain

Domain may:

- Define entities
- Define repository interfaces
- Define domain and application errors

Domain must not:

- Import infrastructure
- Import delivery/http
- Import config
- Import GORM
- Import Redis clients

### Infrastructure

Infrastructure may:

- Connect to external systems
- Implement repository interfaces
- Map domain entities to database models

Infrastructure must not:

- Accept HTTP request objects
- Return HTTP responses
- Own business workflows that belong in use cases

## Task Naming

Tasks should be small enough to complete in one PR.

Good examples:

```text
Implement client alias use case
Implement room join action
Implement room destroy permission check
Implement message creation flow
Add tests for room use case
```

Bad examples:

```text
Build chat backend
Finish all APIs
Improve system
Fix everything
```

A good task should include:

- The endpoint or module involved
- Expected behavior
- Expected files or layers to change
- Required validation

## Testing Rules

Before pushing or opening a PR, run at least:

```sh
go test ./...
```

When build or dependency behavior changes, run:

```sh
go build ./cmd/api
```

When Docker Compose changes, run:

```sh
docker compose --env-file .env.example config
```

When API behavior changes, add use case tests at minimum.

## Documentation Rules

Update documentation when behavior affects other developers.

Update README when:

- Adding an environment variable
- Adding an endpoint
- Changing the service run flow
- Adding a dependency
- Changing the migration workflow

Documentation should be direct and actionable.

## Security Rules

Never commit:

- Real passwords
- Access tokens
- Private keys
- Production connection strings
- Raw client identifiers in logs or responses
- Real `.env` files

Service-specific security rules:

- Hash client identifiers before persistence.
- Return aliases, not raw identifiers.
- Do not leak internal details in error responses.
- Do not log secrets or unnecessary personal data.

## Pre-PR Checklist

Use this checklist before opening every PR:

- Branch name follows the naming rule.
- Commit message follows the naming rule.
- PR scope is not too broad.
- No secrets or credentials are committed.
- No frontend changes are included in backend-only tasks.
- Layer boundaries are respected.
- API contract did not change unexpectedly.
- `go test ./...` passes.
- README or docs are updated when needed.
- PR description explains the change and validation.

## Summary

Consistent naming and small scoped changes make reviews easier, reduce conflicts, and help the repository grow safely.

When naming a branch, commit, file, package, or task, choose a name that clearly answers:

- What does this do?
- Which area does it affect?
- Does it change behavior?
- What should the reviewer focus on?

Good names reduce mistakes before review starts.
