# Team Development Guide

เอกสารนี้เป็นกติกากลางสำหรับทีมที่พัฒนา `chat-service` ใช้เป็นมาตรฐานเดียวกันเรื่องการตั้งชื่อ branch, commit, pull request, file, package, function, database migration และรูปแบบงานที่ควรทำในแต่ละ PR

เป้าหมายคือทำให้ repository อ่านง่าย ตรวจง่าย ขยายต่อได้ และลดความเสี่ยงจากการแก้โค้ดกระทบกันโดยไม่จำเป็น

## หลักการทำงานร่วมกัน

ก่อนเริ่มงานทุกครั้ง ให้ยึดหลักเหล่านี้:

- แก้เฉพาะ scope ของงานที่ได้รับ
- แยกงานใหญ่เป็น PR เล็กที่ review ได้จริง
- ไม่เปลี่ยน public API contract ถ้า task ไม่ได้สั่ง
- ไม่ย้ายโครงสร้าง folder โดยไม่มีเหตุผลทาง architecture
- ไม่ใส่ business logic ใน HTTP handler
- ไม่ใส่ database query ใน usecase หรือ handler โดยตรง
- ไม่ expose raw client identifier ให้ client อื่นเห็น
- ต้อง run validation ที่เกี่ยวข้องก่อนเปิดหรืออัปเดต PR

## Branch Naming

Branch name ต้องบอกประเภทงานและขอบเขตงานให้ชัดเจน

รูปแบบมาตรฐาน:

```text
<type>/<scope>-<short-description>
```

ตัวอย่าง:

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

| Type | ใช้เมื่อ |
| --- | --- |
| `feat` | เพิ่ม feature หรือ behavior ใหม่ |
| `fix` | แก้ bug |
| `docs` | แก้เอกสารเท่านั้น |
| `test` | เพิ่มหรือแก้ test |
| `refactor` | ปรับโครงสร้างโค้ดโดยไม่เปลี่ยน behavior |
| `chore` | งานดูแล repo เช่น Docker, dependency, config |
| `perf` | ปรับ performance |
| `build` | แก้ build system หรือ dependency ที่เกี่ยวกับ build |

### Branch Rules

ควรทำ:

```text
feat/chat-service-client-alias
fix/chat-service-invalid-room-status
docs/chat-service-naming-guide
```

ไม่ควรทำ:

```text
update
fix
backend
new-code
work
final
test123
```

เหตุผลคือชื่อ branch ที่กว้างเกินไปทำให้ reviewer ไม่รู้ว่า branch นี้มีไว้ทำอะไร และทำให้ค้นย้อนหลังยาก

## Commit Naming

ใช้รูปแบบ Conventional Commit

```text
<type>: <summary>
```

ตัวอย่าง:

```text
feat: implement room join use case
fix: return forbidden for non-owner room destroy
docs: add team development guide
test: add room use case tests
chore: add docker compose for local dependencies
refactor: split message repository mapping
```

### Commit Types

| Type | ใช้เมื่อ |
| --- | --- |
| `feat` | เพิ่ม behavior ใหม่ |
| `fix` | แก้ bug |
| `docs` | แก้เอกสาร |
| `test` | เพิ่มหรือแก้ test |
| `refactor` | refactor โดยไม่เปลี่ยน behavior |
| `chore` | งานดูแลทั่วไป |
| `perf` | ปรับ performance |
| `build` | แก้ build/dependency |

### Commit Rules

หนึ่ง commit ควรแทนหนึ่ง logical change

ควรทำ:

```text
feat: implement client alias lookup
test: add client alias use case tests
docs: document branch naming rules
```

ไม่ควรทำ:

```text
update code
fix bug
changes
wip
final
done
```

ถ้า commit มีหลายเรื่องปนกัน เช่น แก้ room logic, แก้ Dockerfile, และแก้ README ใน commit เดียว ควรแยก commit หรือแยก PR ตามความเหมาะสม

## Pull Request Naming

PR title ต้องสรุปสิ่งที่เปลี่ยนแบบอ่านแล้วเข้าใจทันที

รูปแบบที่แนะนำ:

```text
<type>: <clear summary>
```

ตัวอย่าง:

```text
feat: implement room join endpoint
fix: prevent non-owner room destroy
docs: add team development guide
test: add message use case tests
```

PR description ควรมีอย่างน้อย:

- Summary: เปลี่ยนอะไร
- Reason: ทำไปทำไม
- Validation: ตรวจอะไรแล้ว
- Notes: มีอะไรที่ยังไม่ทำหรือ intentionally left out

ตัวอย่าง:

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

ทุก PR เข้า `main` ต้องผ่าน review อย่างน้อย 1 คน

Reviewer ควรตรวจ:

- Scope ตรงกับ task หรือไม่
- API contract เปลี่ยนโดยไม่ได้ตั้งใจหรือไม่
- Layer boundary ถูกต้องหรือไม่
- Error response ตรงรูปแบบกลางหรือไม่
- มี test เพียงพอกับความเสี่ยงหรือไม่
- ไม่มี raw client identifier ถูกส่งออกไปใน response หรือ log
- ไม่มี secret หรือ credential ถูก commit

ถ้า PR ยังไม่พร้อม merge ให้ใช้ Draft PR

## Go Package Naming

Package name ต้องเป็นตัวเล็ก สั้น และชัดเจน

ควรทำ:

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

ไม่ควรทำ:

```text
Config
UseCase
room_handler
postgresRepository
commonUtils
helpers
```

กฎสำคัญ:

- package ใช้ lowercase เท่านั้น
- หลีกเลี่ยง underscore ในชื่อ package
- หลีกเลี่ยงชื่อกว้างเกินไป เช่น `common`, `utils`, `helpers`
- package name ควรบอก responsibility ไม่ใช่ implementation detail ที่ไม่จำเป็น

## Go File Naming

ชื่อไฟล์ใช้ snake_case และบอกหน้าที่ของไฟล์

ควรทำ:

```text
room_usecase.go
message_usecase.go
alias_handler.go
room_repository.go
message_repository.go
request_id.go
```

ไม่ควรทำ:

```text
RoomUseCase.go
messageUseCase.go
utils.go
helper.go
all.go
main2.go
```

ถ้าไฟล์เริ่มใหญ่เกินไป ให้แยกตาม responsibility ไม่ใช่แยกตามความสะดวกชั่วคราว

## Go Type Naming

Exported type ใช้ PascalCase

```go
type RoomUseCase struct {}
type MessageRepository interface {}
type AliasHandler struct {}
```

Private type ใช้ camelCase

```go
type requestIDKey struct {}
type statusRecorder struct {}
```

Interface name ควรตั้งตาม behavior

ควรทำ:

```go
type RoomRepository interface {}
type MessagePubSub interface {}
type MessageSubscription interface {}
```

ไม่ควรทำ:

```go
type IRoomRepository interface {}
type RoomRepositoryInterface interface {}
type Manager interface {}
```

ใน Go ไม่ต้องเติม `I` หน้า interface

## Function Naming

Function name ต้องบอก action ชัดเจน

ควรทำ:

```go
func NewRoomUseCase(...) *RoomUseCase
func FindByRoomID(ctx context.Context, roomID string) (*Room, error)
func MarkMemberLeft(ctx context.Context, roomID string, identifierHash string, leftAt time.Time) error
func roomToModel(room *domain.Room) RoomModel
```

ไม่ควรทำ:

```go
func Do(...)
func Process(...)
func HandleData(...)
func Manage(...)
func Convert(...)
```

ถ้า function ชื่อกว้างเกินไป มักเป็นสัญญาณว่า function นั้นทำหลายอย่างเกินไป

## Variable Naming

ใช้ชื่อสั้นได้เมื่อ scope แคบ แต่ต้องอ่านรู้เรื่อง

ควรทำ:

```go
ctx := r.Context()
roomID := req.RoomID
identifierHash := idgen.HashIdentifier(req.Identifier)
message := domain.Message{}
```

ไม่ควรทำ:

```go
x := req.RoomID
data := domain.Message{}
thing := idgen.HashIdentifier(req.Identifier)
resultObj := output
```

ชื่อตัวแปรควรสะท้อนความหมายทาง domain เช่น `roomID`, `identifierHash`, `senderName`, `expiresAt`

## Error Naming

Domain error กลางใช้รูปแบบ `Err<Name>`

```go
var (
    ErrInvalidInput = errors.New("invalid_input")
    ErrNotFound     = errors.New("not_found")
    ErrForbidden    = errors.New("forbidden")
)
```

Error message ที่ส่งให้ client ต้องเป็นภาษาอังกฤษ และไม่ควรเปิดเผย internal detail เช่น SQL query, Redis command, connection string หรือ secret

## Context Rules

Function ที่ทำ I/O หรือทำงานข้าม layer ต้องรับ `context.Context` เป็น parameter แรก

ควรทำ:

```go
func (r *RoomRepository) FindByRoomID(ctx context.Context, roomID string) (*domain.Room, error)
```

ไม่ควรทำ:

```go
func (r *RoomRepository) FindByRoomID(roomID string) (*domain.Room, error)
```

Handler ต้องส่ง `r.Context()` เข้า usecase เสมอ

```go
output, err := h.useCase.GetStatus(r.Context(), roomID)
```

ห้ามสร้าง `context.Background()` ใน handler, usecase หรือ repository เพื่อแทน request context ยกเว้นเป็น background job ที่ออกแบบไว้โดยเฉพาะ

## API Route Naming

Route ใช้ lowercase และ resource-based naming

ใช้รูปแบบ:

```text
GET    /health
POST   /api/client-alias
POST   /api/rooms
GET    /api/rooms/status
POST   /api/messages
GET    /api/messages
GET    /api/messages/stream
```

ไม่ควรเพิ่ม route ที่ชื่อกำกวม เช่น:

```text
POST /api/do-room
POST /api/action
GET  /api/getMessages
```

ถ้าต้องเพิ่ม endpoint ใหม่ ให้ยึดรูปแบบ resource เดิมของระบบ

## JSON Field Naming

JSON field ใช้ camelCase ตาม frontend contract

ตัวอย่าง:

```json
{
  "identifier": "client-id",
  "roomId": "room-id",
  "senderName": "Display Name",
  "sentAt": "2026-04-25T12:00:00.000Z"
}
```

Go struct field ใช้ PascalCase แต่ tag JSON ใช้ camelCase

```go
type CreateMessageRequest struct {
    Identifier string `json:"identifier" validate:"required"`
    RoomID     string `json:"roomId" validate:"required"`
    Body       string `json:"body" validate:"required"`
}
```

ห้ามเปลี่ยน JSON field name โดยไม่ตรวจ frontend contract

## Database Naming

Table name ใช้ plural snake_case

```text
rooms
room_members
client_aliases
messages
```

Column name ใช้ snake_case

```text
room_id
identifier_hash
owner_identifier_hash
is_destroyed
expires_at
created_at
updated_at
```

Index name ควรบอก table และ columns

```text
idx_messages_room_sent_at
idx_rooms_destroyed_expires_at
idx_client_aliases_room_id
```

Unique constraint name ควรขึ้นต้นด้วย `uq_`

```text
uq_room_members_room_identifier
uq_client_aliases_room_identifier
```

## Migration Naming

Migration file ใช้เลขลำดับและคำอธิบายสั้นๆ

```text
000001_create_chat_tables.up.sql
000001_create_chat_tables.down.sql
000002_add_room_destroyed_at.up.sql
000002_add_room_destroyed_at.down.sql
```

กฎ migration:

- ทุก `.up.sql` ต้องมี `.down.sql`
- ห้ามแก้ migration เก่าที่ merge ไปแล้ว ยกเว้นยังไม่เคย deploy
- migration ต้อง review ได้ง่าย
- หลีกเลี่ยง destructive change ถ้าไม่มี migration plan ชัดเจน
- production migration ต้องคำนึงถึง data เดิมเสมอ

## Environment Variable Naming

Environment variables ใช้ uppercase snake_case

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

กฎสำคัญ:

- ห้าม commit secret จริง
- `.env.example` ใส่ได้เฉพาะค่าตัวอย่าง
- ถ้าเพิ่ม env ใหม่ ต้อง update `.env.example` และ README
- ชื่อ env ต้องอ่านแล้วรู้ว่าควบคุม behavior อะไร

## Layer Boundary Rules

ระบบนี้แบ่ง layer ชัดเจน

```text
delivery/http -> usecase -> domain
infrastructure -> domain interfaces
bootstrap -> wires dependencies
```

### Handler

Handler ทำได้:

- decode request
- validate request
- call usecase
- map response

Handler ห้าม:

- query database
- call Redis โดยตรง
- เขียน business rule ซับซ้อน
- ใช้ GORM model

### Usecase

Usecase ทำได้:

- orchestrate business flow
- call repository interface
- enforce business rule
- return domain/application error

Usecase ห้าม:

- import GORM
- import Chi
- import HTTP package
- รู้จัก DTO ของ HTTP layer

### Domain

Domain ทำได้:

- define entity
- define repository interface
- define domain/application error

Domain ห้าม:

- import infrastructure
- import delivery/http
- import config
- import GORM
- import Redis client

### Infrastructure

Infrastructure ทำได้:

- connect external systems
- implement repository interface
- map domain entity กับ database model

Infrastructure ห้าม:

- รับ HTTP request object
- return HTTP response
- บังคับ business workflow ที่ควรอยู่ใน usecase

## Task Naming

Task ควรตั้งชื่อให้เล็กและทำจบใน PR เดียว

ควรทำ:

```text
Implement client alias use case
Implement room join action
Implement room destroy permission check
Implement message creation flow
Add tests for room use case
```

ไม่ควรทำ:

```text
Build chat backend
Finish all APIs
Improve system
Fix everything
```

Task ที่ดีควรมี:

- endpoint หรือ module ที่เกี่ยวข้อง
- expected behavior
- files/layers ที่คาดว่าจะต้องแก้
- validation ที่ต้องผ่าน

## Testing Rules

ก่อน push หรือเปิด PR ควรรันอย่างน้อย:

```sh
go test ./...
```

ถ้าแก้ build หรือ dependency:

```sh
go build ./cmd/api
```

ถ้าแก้ Docker Compose:

```sh
docker compose config
```

ถ้าแก้ API behavior ควรมี test ในระดับ usecase เป็นอย่างน้อย

## Documentation Rules

ถ้าแก้ behavior ที่กระทบ developer คนอื่น ต้อง update เอกสาร

ควร update README เมื่อ:

- เพิ่ม env variable
- เพิ่ม endpoint
- เปลี่ยนวิธี run service
- เพิ่ม dependency ใหม่
- เปลี่ยน migration workflow

เอกสารควรเขียนให้คนในทีมอ่านแล้วทำตามได้ทันที ไม่ควรเขียนเป็นข้อความกำกวม

## Security Rules

ห้าม commit สิ่งเหล่านี้:

- password จริง
- access token
- private key
- production connection string
- raw client identifier ใน log หรือ response
- ไฟล์ `.env` จริง

สิ่งที่ต้องระวังใน service นี้:

- client identifier ต้อง hash ก่อน persist
- response ต้องส่ง alias ไม่ใช่ raw identifier
- error response ต้องไม่ leak internal detail
- log ต้องไม่เก็บ secret หรือข้อมูลส่วนตัวที่ไม่จำเป็น

## Checklist ก่อนเปิด PR

ใช้ checklist นี้ก่อนเปิด PR ทุกครั้ง:

- Branch name ถูกต้อง
- Commit message ถูกต้อง
- Scope ของ PR ไม่กว้างเกินไป
- ไม่มีไฟล์ลับหรือ credential
- ไม่มี frontend change ถ้า task เป็น backend-only
- Layer boundary ถูกต้อง
- API contract ไม่เปลี่ยนโดยไม่ตั้งใจ
- `go test ./...` ผ่าน
- README หรือ docs ถูก update ถ้าจำเป็น
- PR description อธิบายสิ่งที่เปลี่ยนและ validation ครบ

## สรุป

มาตรฐานการตั้งชื่อและการแยกงานช่วยให้ทีม review ง่าย ลด conflict และทำให้ repository โตได้อย่างเป็นระบบ

ถ้าไม่แน่ใจว่าจะตั้งชื่อ branch, commit, file หรือ package อย่างไร ให้เลือกชื่อที่อ่านแล้วตอบคำถามได้ทันทีว่า:

- งานนี้ทำอะไร
- กระทบส่วนไหน
- เปลี่ยน behavior หรือไม่
- reviewer ต้องตรวจเรื่องอะไร

ชื่อที่ดีช่วยลดความผิดพลาดได้ตั้งแต่ก่อนเริ่ม review
