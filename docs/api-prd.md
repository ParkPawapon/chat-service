# API Endpoint PRD

This PRD documents the current `chat-service` HTTP API contract. All JSON fields use camelCase. Error responses use:

```json
{
  "error": {
    "code": "invalid_request",
    "message": "request validation failed",
    "fields": {
      "RoomID": "is required"
    }
  }
}
```

Common error codes: `invalid_request`, `forbidden`, `not_found`, `gone`, `conflict`, `not_implemented`, `internal_error`.

## GET /health

### Purpose

Allow clients, local developers, and deployment infrastructure to verify that the HTTP service is running.

### Users

- Deployment health checks
- Developers running the service locally

### Request

No request body or query parameters.

### Success Response

Status: `200 OK`

```json
{
  "status": "ok"
}
```

### Acceptance Criteria

- Returns `200 OK` when the HTTP server is reachable.
- Does not require database or Redis access.

## POST /api/v1/client-alias

### Purpose

Return a stable room-scoped display alias for a client identifier without exposing the raw identifier.

### Users

- Chat clients that need to display sender names
- Message creation flow

### Request

```json
{
  "identifier": "client-local-storage-id",
  "roomId": "room-a"
}
```

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `identifier` | Yes | 512 chars | Trimmed and hashed before persistence |
| `roomId` | Yes | 128 chars | Trimmed |

### Success Response

Status: `200 OK`

```json
{
  "alias": "Client-ABC12345"
}
```

### Business Rules

- Same `identifier` and `roomId` must return the same alias.
- Same `identifier` in a different room may return a different alias.
- Raw identifiers must not be stored or returned.

### Error Cases

- `400 invalid_request` for invalid JSON, unknown fields, missing fields, empty trimmed values, or length violations.

### Acceptance Criteria

- Creates an alias when none exists.
- Returns the existing alias on repeat requests.
- Response never includes the raw client identifier.

## POST /api/v1/rooms

### Purpose

Perform room lifecycle actions through one endpoint: join, destroy, or leave.

### Users

- Chat clients entering, leaving, or ending a room

### Request

```json
{
  "action": "join",
  "identifier": "client-local-storage-id",
  "roomId": "room-a"
}
```

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `action` | Yes | `join`, `destroy`, `leave` | Unsupported values are rejected |
| `identifier` | Yes | 512 chars | Trimmed and hashed before persistence |
| `roomId` | Yes | 128 chars | Trimmed |

### Join Success Response

Status: `200 OK`

```json
{
  "isDestroyed": false,
  "isOwner": true
}
```

### Destroy Success Response

Status: `200 OK`

```json
{
  "isDestroyed": true
}
```

### Leave Success Response

Status: `200 OK`

```json
{
  "hasLeft": true
}
```

### Business Rules

- `join` creates the room if it does not exist; the first joiner becomes owner.
- `join` adds or reactivates a room member when the room is active.
- `join` on a destroyed room returns `isDestroyed: true` and `isOwner: false`.
- `destroy` is allowed only for the room owner.
- `destroy` is idempotent for an already destroyed room.
- `leave` marks an existing member as left and is idempotent after the member has left.
- Expired rooms cannot be joined, destroyed, or left.

### Error Cases

- `400 invalid_request` for invalid JSON, unknown fields, missing fields, unsupported action, empty trimmed values, or length violations.
- `403 forbidden` when a non-owner attempts `destroy`.
- `404 not_found` when `destroy` or `leave` targets a missing room or missing membership.
- `410 gone` when the room is expired.

### Acceptance Criteria

- Correct response shape is returned for each action.
- Owner-only room destruction is enforced.
- Client identifiers are hashed before persistence.

## GET /api/v1/rooms/status

### Purpose

Return current room state for polling clients and operational UI state.

### Users

- Chat clients checking expiration, destruction, and message count

### Query Parameters

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `roomId` | Yes | 128 chars | Trimmed |

### Success Response

Status: `200 OK`

```json
{
  "expiresAt": "2026-01-02T03:04:05.000Z",
  "isDestroyed": false,
  "messageCount": 3,
  "serverTime": "2026-01-02T03:00:00.000Z"
}
```

### Business Rules

- `expiresAt` and `serverTime` are UTC timestamps formatted as `YYYY-MM-DDTHH:mm:ss.sssZ`.
- `messageCount` is the persisted message count for the room.

### Error Cases

- `400 invalid_request` when `roomId` is missing or invalid.
- `404 not_found` when the room does not exist.

### Acceptance Criteria

- Returns room destruction state and expiry time.
- Returns server time so clients can calculate remaining room lifetime.

## POST /api/v1/messages

### Purpose

Persist a chat message and publish it to active room subscribers.

### Users

- Chat clients sending messages

### Request

```json
{
  "identifier": "client-local-storage-id",
  "roomId": "room-a",
  "body": "hello world"
}
```

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `identifier` | Yes | 512 chars | Trimmed and hashed before persistence |
| `roomId` | Yes | 128 chars | Trimmed |
| `body` | Yes | 4096 chars | Trimmed before persistence |

### Success Response

Status: `200 OK`

```json
{
  "id": "message-id",
  "body": "hello world",
  "senderName": "Client-ABC12345",
  "sentAt": "2026-01-02T03:04:05.000Z"
}
```

### Business Rules

- Room must exist, be active, and not be expired.
- Sender alias is created or reused automatically.
- Message is persisted before publishing to Redis.
- Redis publish failure is logged but does not fail the HTTP request.

### Error Cases

- `400 invalid_request` for invalid JSON, unknown fields, missing fields, empty trimmed values, or length violations.
- `404 not_found` when the room does not exist.
- `410 gone` when the room is destroyed or expired.

### Acceptance Criteria

- Persists message with sender alias and timestamp.
- Returns the message contract used by list and stream APIs.
- Does not expose raw client identifiers.

## GET /api/v1/messages

### Purpose

Return persisted messages for a room in chronological order.

### Users

- Chat clients loading room history

### Query Parameters

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `roomId` | Yes | 128 chars | Trimmed |

### Success Response

Status: `200 OK`

```json
{
  "messages": [
    {
      "id": "message-id",
      "body": "hello world",
      "senderName": "Client-ABC12345",
      "sentAt": "2026-01-02T03:04:05.000Z"
    }
  ]
}
```

### Business Rules

- Room must exist, be active, and not be expired.
- Messages are returned oldest to newest.

### Error Cases

- `400 invalid_request` when `roomId` is missing or invalid.
- `404 not_found` when the room does not exist.
- `410 gone` when the room is destroyed or expired.

### Acceptance Criteria

- Returns an empty `messages` array when the room has no messages.
- Returns only the public message fields.

## GET /api/v1/messages/stream

### Purpose

Stream new room messages to clients using Server-Sent Events.

### Users

- Chat clients listening for realtime message updates

### Query Parameters

| Field | Required | Limit | Notes |
| --- | --- | --- | --- |
| `roomId` | Yes | 128 chars | Trimmed |

### Success Response

Status: `200 OK`

Headers:

```text
Content-Type: text/event-stream
Cache-Control: no-cache
Connection: keep-alive
```

Event frame:

```text
event: message
data: {"id":"message-id","body":"hello world","senderName":"Client-ABC12345","sentAt":"2026-01-02T03:04:05.000Z"}
```

### Business Rules

- Room must exist, be active, and not be expired before subscription starts.
- Stream closes when the client disconnects.
- Stream checks room status periodically and closes when the room becomes unavailable.
- Each event payload uses the same public message contract as `POST /api/v1/messages`.

### Error Cases

- `400 invalid_request` when `roomId` is missing or invalid.
- `404 not_found` when the room does not exist.
- `410 gone` when the room is destroyed or expired before streaming begins.
- `500 internal_error` when streaming is not supported by the HTTP writer.

### Acceptance Criteria

- Sends `message` SSE events for published room messages.
- Does not send events for other rooms.
- Closes cleanly on client disconnect.
