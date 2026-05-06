# Unit Testing Guide

This guide explains how we write unit tests in `chat-service`.

The goal is simple: a developer should be able to change one endpoint, run the tests, and know whether the important behavior still works. Tests should make the code safer to change, not harder to maintain.

This document is written for backend work only. Do not use it as a frontend testing guide.

## What We Mean by Unit Test

A unit test checks one small piece of code in isolation.

In this service, that usually means one of these:

- A use case method, such as `GetOrCreateAlias`.
- An HTTP handler method, such as `AliasHandler.GetOrCreate`.
- A small pure helper, such as an ID or time formatting helper.

A unit test must not require:

- PostgreSQL
- Redis
- Docker
- Network calls
- Real migrations
- A running HTTP server
- GitHub Actions
- A specific test order

If a test needs PostgreSQL or Redis, it is an integration test, not a unit test.

## Why Unit Tests Matter Here

`chat-service` uses Clean Architecture. That means business behavior should live in the use case layer, while HTTP and database code stay in their own layers.

Good unit tests protect that design:

- Handler tests verify HTTP contracts.
- Use case tests verify business rules.
- Repository tests are usually integration tests because they verify real database behavior.
- Domain tests verify pure domain logic when domain behavior exists.

When tests follow these boundaries, a bug is easier to locate. A failing handler test usually means request parsing or response mapping is wrong. A failing use case test usually means business behavior is wrong.

## Testing Boundaries

Use this table when deciding where a test belongs.

| Code being tested | Test location | Test type | External services |
| --- | --- | --- | --- |
| `internal/usecase/alias_usecase.go` | `internal/usecase/alias_usecase_test.go` | Unit | No |
| `internal/delivery/http/handler/alias_handler.go` | `internal/delivery/http/handler/alias_handler_test.go` | Unit | No |
| `internal/infrastructure/postgres/*_repository.go` | Same package or integration test package | Integration | PostgreSQL |
| `internal/infrastructure/redis/pubsub.go` | Same package or integration test package | Integration | Redis |
| `cmd/api/main.go` wiring | Smoke or integration test | Integration | Usually yes |

Do not test a use case by starting the whole app. That makes the test slower and hides the real failure point.

## File Placement

Put test files next to the code they test.

Good:

```text
internal/usecase/
  alias_usecase.go
  alias_usecase_test.go
```

Good:

```text
internal/delivery/http/handler/
  alias_handler.go
  alias_handler_test.go
```

Avoid:

```text
tests/
  alias_usecase_test.go
```

A top-level `tests/` folder makes sense in some projects, but not for this service right now. Go already organizes tests by package. Keeping tests near the code is easier to read and review.

## Test File Naming

Use this pattern:

```text
<file_under_test>_test.go
```

Examples:

```text
alias_usecase.go
alias_usecase_test.go

room_handler.go
room_handler_test.go

message_usecase.go
message_usecase_test.go
```

For helper files used only by tests, use a clear name:

```text
alias_test_repository_test.go
room_usecase_test_helpers.go
message_test_pubsub_test.go
```

The file must end in `_test.go` if it contains test-only code that should not be compiled into the normal application binary.

## Package Naming in Tests

Most tests in this service should use the same package as the code being tested.

Example:

```go
package usecase
```

This is useful when testing behavior close to the package and using small package-local helpers.

Use an external test package only when you intentionally want to test public behavior from the outside:

```go
package usecase_test
```

For this service, prefer same-package tests unless there is a clear reason to use an external package.

## Test Function Naming

Use this format:

```go
func Test<TypeOrFunction><Behavior>(t *testing.T)
```

Good examples:

```go
func TestAliasUseCaseGetOrCreateAlias(t *testing.T)
func TestAliasHandlerGetOrCreate(t *testing.T)
func TestRoomUseCaseJoinRoom(t *testing.T)
func TestMessageUseCaseCreateMessage(t *testing.T)
```

Avoid vague names:

```go
func TestAlias(t *testing.T)
func TestHandler(t *testing.T)
func TestCase1(t *testing.T)
```

The test name should tell the reviewer what behavior is being protected.

## Subtest Naming

Use subtests to describe behavior in plain English.

Good:

```go
t.Run("returns existing alias without creating a new record", func(t *testing.T) {
    // ...
})
```

Good:

```go
t.Run("returns 400 when roomId is missing", func(t *testing.T) {
    // ...
})
```

Avoid:

```go
t.Run("success", func(t *testing.T) {})
t.Run("fail", func(t *testing.T) {})
t.Run("case 1", func(t *testing.T) {})
```

Good subtest names are useful in CI logs. When a test fails, the name should already explain the broken rule.

## What to Test in a Use Case

Use case tests should focus on business behavior.

For `AliasUseCase.GetOrCreateAlias`, the important behavior is:

- Missing `identifier` returns `ErrInvalidInput`.
- Missing `roomId` returns `ErrInvalidInput`.
- Existing alias is returned without creating a duplicate.
- New alias is deterministic.
- Same `identifier + roomId` returns the same alias.
- Same identifier in a different room is room-scoped.
- Raw client identifier is not stored.
- Repository dependency errors are returned.
- Create conflict can fall back to reading the alias created by a concurrent request.

Use case tests should not check:

- HTTP status codes.
- JSON response shape.
- GORM behavior.
- SQL indexes.
- Redis publish behavior unless the use case depends on a pub/sub interface and the test uses a fake.

## What to Test in a Handler

Handler tests should focus on the HTTP contract.

For `POST /api/v1/client-alias`, the important behavior is:

- Valid JSON returns `200`.
- Response body contains `alias`.
- Missing `identifier` returns `400`.
- Missing `roomId` returns `400`.
- Non-string `identifier` returns `400`.
- Non-string `roomId` returns `400`.
- Unknown JSON fields return `400`.
- Raw client identifier is not exposed in the alias response.

Handler tests should not check:

- Database implementation.
- Redis implementation.
- SQL migrations.
- GORM model tags.

The handler should call a use case or use case backed by a fake repository. It should not talk to a real database.

## What Not to Unit Test

Do not write unit tests for framework behavior.

Avoid tests like:

- Testing that `json.Unmarshal` works.
- Testing that `chi` routes requests correctly.
- Testing that GORM can insert records without a database.
- Testing that Redis pub/sub works without Redis.

Test our behavior, not the standard library or third-party libraries.

## Fakes, Stubs, and Mocks

Use simple fakes first.

A fake is a small in-memory implementation of an interface. It behaves enough like the real dependency for the test.

Example:

```go
type memoryAliasRepository struct {
    mu      sync.Mutex
    records map[string]domain.ClientAlias
}
```

This is good for use case tests because `AliasUseCase` depends on the `domain.AliasRepository` interface, not PostgreSQL.

Use scripted stubs when you need exact behavior.

Example:

```go
repo := &scriptedAliasRepository{
    findFunc: func(ctx context.Context, roomID string, identifierHash string) (*domain.ClientAlias, error) {
        return nil, domain.NewAppError(domain.ErrNotFound, "client alias not found")
    },
}
```

This is useful for testing dependency errors, conflicts, or edge cases.

Avoid heavy mocking frameworks unless the standard library approach becomes painful. For this project, plain Go fakes are enough.

## Test Helpers

Keep helpers small and boring.

Good helpers:

- Build a request.
- Decode a response.
- Create an in-memory repository.
- Assert that a fake repository contains a value.

Bad helpers:

- Hide the whole test flow.
- Contain business logic.
- Return unclear data.
- Make the test harder to read than the code being tested.

Use `t.Helper()` inside helper functions so failures point to the caller.

Example:

```go
func decodeAliasResponse(t *testing.T, recorder *httptest.ResponseRecorder) string {
    t.Helper()

    var responseBody struct {
        Alias string `json:"alias"`
    }
    if err := json.Unmarshal(recorder.Body.Bytes(), &responseBody); err != nil {
        t.Fatalf("failed to decode alias response: %v", err)
    }
    return responseBody.Alias
}
```

## Parallel Tests

Use `t.Parallel()` when the test does not share mutable state with other tests.

Good:

```go
func TestAliasUseCaseGetOrCreateAlias(t *testing.T) {
    t.Parallel()

    t.Run("validates required input", func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

Be careful with shared fakes. If a fake repository can be used by parallel tests, protect its map with a mutex.

Good:

```go
type memoryAliasRepository struct {
    mu      sync.Mutex
    records map[string]domain.ClientAlias
}
```

Avoid package-level mutable test state.

## Context Usage

Production code uses `context.Context`, so tests should pass a context too.

Use this for normal unit tests:

```go
ctx := context.Background()
```

Use a canceled context only when the behavior being tested depends on cancellation.

Do not use `context.TODO()` in committed tests. It tells the reader that the test is unfinished.

## Error Assertions

Use `errors.Is` for application error kinds.

Good:

```go
if !errors.Is(err, domain.ErrInvalidInput) {
    t.Fatalf("expected ErrInvalidInput, got %v", err)
}
```

Avoid string matching unless there is no better option.

Bad:

```go
if err.Error() != "roomId is required" {
    t.Fatal("wrong error")
}
```

The error message can change. The error kind is the contract.

## HTTP Test Style

Use `httptest` for handler tests.

Example:

```go
request := httptest.NewRequest(http.MethodPost, "/api/v1/client-alias", bytes.NewBufferString(body))
request.Header.Set("Content-Type", "application/json")

recorder := httptest.NewRecorder()
handler.GetOrCreate(recorder, request)
```

Do not start a real server for a handler unit test.

Use a real server only for integration or smoke tests.

## Table-Driven Tests

Use table-driven tests when multiple cases share the same setup.

Good:

```go
tests := []struct {
    name string
    body string
}{
    {name: "missing identifier", body: `{"roomId":"room-a"}`},
    {name: "missing room id", body: `{"identifier":"client-local-storage-id"}`},
}

for _, tt := range tests {
    tt := tt
    t.Run(tt.name, func(t *testing.T) {
        t.Parallel()
        // ...
    })
}
```

Keep each case readable. If every row needs many fields and special rules, split it into separate tests.

## Time in Tests

Avoid real time when possible.

For future use cases that depend heavily on time, prefer injecting a clock function.

Example:

```go
type RoomUseCase struct {
    now func() time.Time
}
```

Then tests can use a fixed time.

Do not use `time.Sleep` in unit tests unless there is no reasonable alternative. Sleeping tests are slow and flaky.

## Random IDs in Tests

Avoid randomness when the value is part of the assertion.

Good:

```go
roomID := "room-a"
identifier := "client-local-storage-id"
```

Use generated IDs only when the exact value does not matter.

Tests should be repeatable. A test should not pass only because random data happened to avoid a collision.

## Unit Tests by Layer

### Domain

Test domain code when it contains real behavior.

Current domain entities are mostly data structures, so there is not much to unit test there yet.

When domain behavior is added, test it without use cases, HTTP, database, or Redis.

### Use Case

Use case tests are the most important tests for business behavior.

Use fake repositories and fake pub/sub implementations through domain interfaces.

Use case tests should answer this question:

```text
Does the application rule work?
```

Examples:

- First room join makes the requester the owner.
- Destroyed room join returns `isOwner: false`.
- Non-owner destroy returns forbidden.
- Message creation uses the stable alias.
- Message listing returns oldest to newest.

### Handler

Handler tests protect request and response behavior.

Handler tests should answer this question:

```text
Does the HTTP API contract work?
```

Examples:

- Missing required JSON field returns `400`.
- Query parameter missing returns `400`.
- Domain `ErrForbidden` maps to `403`.
- Domain `ErrGone` maps to `410`.
- Success response has the correct JSON field names.

### Infrastructure

Do not fake GORM and call it a unit test.

Repository behavior is tied to PostgreSQL constraints, indexes, transactions, and SQL behavior. Test repositories with integration tests against PostgreSQL.

Redis pub/sub behavior should be tested with Redis in an integration test.

## Current Example in This Repository

The first complete unit test example is for:

```text
POST /api/v1/client-alias
```

Files:

```text
internal/usecase/alias_usecase_test.go
internal/usecase/alias_test_repository_test.go
internal/delivery/http/handler/alias_handler_test.go
```

Use this as the reference for the next endpoint tasks.

The example shows:

- Use case tests with fake repositories.
- Handler tests with `httptest`.
- Required field validation.
- Type validation.
- Unknown field validation.
- Stable alias behavior.
- Room-scoped alias behavior.
- No raw identifier exposure.
- Race fallback after create conflict.
- Dependency error propagation.

## Commands

Run all tests:

```sh
go test ./...
```

Run tests with race detection:

```sh
go test -race ./...
```

Run one package:

```sh
go test ./internal/usecase
```

Run one test:

```sh
go test ./internal/usecase -run TestAliasUseCaseGetOrCreateAlias
```

Run one subtest:

```sh
go test ./internal/usecase -run 'TestAliasUseCaseGetOrCreateAlias/validates_required_input'
```

Run tests with verbose output:

```sh
go test -v ./internal/usecase
```

Run vet:

```sh
go vet ./...
```

Build the API:

```sh
go build -o /tmp/chat-service-api ./cmd/api
```

Before opening a PR that adds tests, run:

```sh
go test -race ./...
go vet ./...
go build -o /tmp/chat-service-api ./cmd/api
git diff --check
```

## Coverage

Coverage is useful, but it is not the goal by itself.

Good coverage means important behavior is protected:

- Success path.
- Validation errors.
- Permission errors.
- Not found and gone behavior.
- Dependency failures.
- Concurrency-sensitive cases.
- Response mapping.

Bad coverage means many lines are executed but the test does not assert business behavior.

Do not write weak tests just to increase a percentage.

## Pull Request Checklist for Unit Tests

Before asking for review, check this list:

- The test is in the same package area as the code being tested.
- The test name describes the behavior.
- The test does not require PostgreSQL or Redis.
- The test does not depend on test order.
- The test does not use real network calls.
- The test uses fake dependencies through interfaces.
- The test checks both success and failure behavior.
- The test uses `errors.Is` for domain error kinds.
- The test does not assert unstable implementation details.
- The test does not expose or commit secrets.
- `go test -race ./...` passes.
- `go vet ./...` passes.
- `go build -o /tmp/chat-service-api ./cmd/api` passes.
- `git diff --check` passes.

## Review Checklist

When reviewing a test PR, ask these questions:

- Does the test protect a real rule from the API contract?
- Is this a unit test, or should it be an integration test?
- Is the setup easy to read?
- Are the assertions meaningful?
- Does the test fail for the right reason if the behavior breaks?
- Is the fake dependency smaller than the production dependency?
- Are we testing our code instead of testing Go, Chi, GORM, or Redis?
- Can the next developer copy this pattern safely?

## Common Mistakes

Avoid these:

- Putting database setup in a unit test.
- Starting the whole server to test one handler.
- Testing only the happy path.
- Checking error strings instead of error kinds.
- Using global mutable state.
- Sharing one fake repository across parallel subtests without a mutex.
- Hiding important behavior inside large helpers.
- Writing tests that pass even when the assertion is wrong.
- Adding sleeps to make concurrency tests pass.
- Using random data when fixed data would be clearer.

## Task Template for the Next Endpoint

Use this template when assigning the next unit test task.

```text
Task: Add unit tests for <endpoint or use case>.

Scope:
- Test only <specific endpoint/use case>.
- Do not touch frontend code.
- Do not use PostgreSQL or Redis.
- Use fake dependencies through domain interfaces.
- Keep tests close to the code under test.

Required coverage:
- Success response.
- Required field validation.
- Invalid type validation where applicable.
- Not found / forbidden / gone behavior where applicable.
- Dependency error behavior.
- Any endpoint-specific business rule.

Validation:
- go test -race ./...
- go vet ./...
- go build -o /tmp/chat-service-api ./cmd/api
- git diff --check
```

## Endpoint Test Targets

Use this as a guide for future tasks.

### `POST /api/v1/client-alias`

Already has the first unit test example.

Important rules:

- Same `identifier + roomId` returns the same alias.
- Raw identifier is not exposed.
- Missing or invalid fields return `400`.

### `POST /api/v1/rooms`

Future tests should cover:

- First join returns owner.
- Later join returns non-owner.
- Destroyed room join returns `isDestroyed: true` and `isOwner: false`.
- Non-owner destroy returns `403`.
- Owner destroy returns success.
- Leave returns success.
- Missing or invalid fields return `400`.
- Concurrent first join does not create two owners.

### `GET /api/v1/rooms/status`

Future tests should cover:

- Missing `roomId` returns `400`.
- Unknown room returns `404`.
- Destroyed room returns `isDestroyed: true`.
- `expiresAt` and `serverTime` use the API time format.
- `messageCount` is returned.

### `POST /api/v1/messages`

Future tests should cover:

- Missing or invalid fields return `400`.
- Unknown room returns `404`.
- Destroyed or expired room returns `410`.
- Sender name comes from alias use case.
- Message is persisted through the message repository interface.
- Pub/sub failure behavior follows the agreed contract.

### `GET /api/v1/messages`

Future tests should cover:

- Missing `roomId` returns `400`.
- Unknown room returns `404`.
- Destroyed or expired room returns `410`.
- Messages are returned oldest to newest.
- Response field names match the API contract.

### `GET /api/v1/messages/stream`

Future tests should be careful.

SSE tests can become integration-like quickly. Keep unit tests focused on:

- Missing `roomId` returns `400`.
- Unknown room maps to the correct status.
- Destroyed or expired room maps to `410`.
- A message event is formatted correctly.

Use integration tests later for real Redis pub/sub behavior and long-running stream behavior.

## Final Rule

A good unit test should be clear enough that a reviewer can understand the business rule without opening the frontend, the database, or the running service.

If the test is hard to read, make the test simpler before adding more coverage.
