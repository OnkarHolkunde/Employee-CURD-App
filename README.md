# Excel Import & Employee CRUD API

A Golang + Gin service that imports employee data from an Excel file,
persists it in MySQL, caches it in Redis (5-minute TTL), and exposes a
full CRUD API to view and edit records — keeping MySQL and Redis in sync
on every write.

## Features

- **Async Excel import**: upload a `.xlsx`/`.xls` file, get a `job_id`
  back immediately (`202 Accepted`), and poll `/api/v1/upload/status/:job_id`
  for progress. Parsing + batch DB inserts happen in a background
  goroutine so the API stays responsive on large files.
- **Header validation**: uploads are rejected with a clear error if the
  column headers don't match the expected schema.
- **Cache-aside reads**: `GET` endpoints check Redis first and fall back to
  MySQL on a miss, repopulating the cache automatically. Cache entries
  expire after 5 minutes (configurable).
- **Write-through updates**: `PUT`/`DELETE` update MySQL and immediately
  refresh (or clear) the corresponding Redis entries.
- **Production concerns**: structured JSON logging, request IDs, panic
  recovery, CORS, graceful shutdown, health (`/health`) and readiness
  (`/ready`) endpoints, connection pooling, batched inserts.

## Tech Stack

| Concern         | Choice                                   |
|-----------------|-------------------------------------------|
| HTTP framework  | [Gin](https://github.com/gin-gonic/gin)   |
| ORM             | [GORM](https://gorm.io) + MySQL driver    |
| Cache           | [go-redis](https://github.com/redis/go-redis) |
| Excel parsing   | [excelize](https://github.com/qax-os/excelize) |
| Config          | [.env](https://github.com/joho/godotenv) via env vars |

## Project Layout

```
.
├── cmd/
│   └── server/            # main.go — composition root, graceful shutdown
├── internal/
│   ├── config/            # env-driven configuration
│   ├── database/          # MySQL (GORM) + Redis client setup
│   ├── models/             # Employee, ImportJob structs
│   ├── services/           # business logic: excel parsing, employee CRUD, job tracking
│   ├── handlers/            # Gin HTTP handlers (thin — delegate to services)
│   ├── middleware/          # request ID, structured logging, recovery, CORS
│   ├── response/            # standard JSON response envelope
│   └── routes/               # route registration + health/readiness checks
├── sample_data/             # the sample Excel file used for testing
├── uploads/                  # scratch space for in-flight uploads (gitignored)
├── Dockerfile
├── docker-compose.yml        # app + MySQL + Redis, one command to run everything
├── Makefile
├── .env.example
└── go.mod / go.sum
```

`internal/` is used deliberately so nothing outside this module can import
these packages — this is a self-contained service, not a shared library.

## Data Model

The Excel file is expected to have these column headers (case-insensitive,
order-independent):

```
first_name | last_name | company_name | address | city | county | postal | phone | email | web
```

This matches `sample_data/Sample_Employee_data.xlsx`.

## Running Locally

### Option A — Docker Compose (recommended, zero local setup)

```bash
make docker-up      # builds the app image, starts MySQL + Redis + app
make docker-logs     # tail the app's logs
make docker-down     # stop and remove everything (including volumes)
```

The API will be available at `http://localhost:8080`.

### Option B — Run natively

Prerequisites: Go 1.22+, a running MySQL instance, a running Redis instance.

```bash
cp .env.example .env     # edit MYSQL_*/REDIS_* to point at your instances
go mod tidy               # fetch dependencies (needs normal internet access)
make run                  # or: go run ./cmd/server
```

> **Note on `go.mod`:** this repo's `go.mod` includes `replace` directives
> that redirect a few transitive dependencies (`golang.org/x/*`,
> `gorm.io/*`, `gopkg.in/*`, `google.golang.org/protobuf`) to their GitHub
> mirror repositories. That was only necessary because this project was
> built inside a sandboxed environment with a restricted network
> allowlist that couldn't reach those vanity domains for module
> resolution. On a normal machine with full internet access, you can
> simply delete those `replace` lines and run `go mod tidy` to resolve
> everything from the canonical module paths — the code itself doesn't
> depend on the redirect in any way.

## API Reference

All responses use a consistent envelope:

```json
{ "success": true, "message": "...", "data": { ... }, "meta": { ... } }
```

### Upload

| Method | Path                          | Description                                   |
|--------|-------------------------------|------------------------------------------------|
| POST   | `/api/v1/upload`               | Multipart form upload, field name `file`       |
| GET    | `/api/v1/upload/status/:job_id`| Poll import progress/result                    |

**Example (Postman/curl):**

```bash
curl -X POST http://localhost:8080/api/v1/upload \
  -F "file=@sample_data/Sample_Employee_data.xlsx"
```

Response:
```json
{
  "success": true,
  "message": "file accepted, processing started",
  "data": { "job_id": "b1f2...", "status_url": "/api/v1/upload/status/b1f2..." }
}
```

Then poll:
```bash
curl http://localhost:8080/api/v1/upload/status/b1f2...
```

### Employees (CRUD)

| Method | Path                    | Description                                  |
|--------|-------------------------|-----------------------------------------------|
| GET    | `/api/v1/employees`      | List (paginated), Redis-first with MySQL fallback |
| GET    | `/api/v1/employees/:id`  | Get one record                                |
| POST   | `/api/v1/employees`      | Create a record manually                      |
| PUT    | `/api/v1/employees/:id`  | Partial update (send only changed fields)     |
| DELETE | `/api/v1/employees/:id`  | Delete a record                               |

`GET /api/v1/employees` accepts `?page=1&page_size=50` (default page size
50, capped at 500).

### Health

| Method | Path       | Description                                          |
|--------|------------|-------------------------------------------------------|
| GET    | `/health`  | Liveness — process is up                              |
| GET    | `/ready`   | Readiness — MySQL and Redis are both reachable        |

## Design Notes

- **Why async upload instead of synchronous parse-and-insert?** A 2000+
  row spreadsheet parsed and inserted inline would tie up a request
  goroutine and an HTTP client connection for the whole duration.
  Returning a job ID immediately keeps the API responsive and lets the
  client (or a webhook, in a fuller build) poll independently.
- **Why cache-aside instead of write-behind?** The assignment explicitly
  asks for "if Redis doesn't have the data, retrieve it from the table" —
  a classic cache-aside pattern. Writes update MySQL as the source of
  truth first, then refresh/invalidate the cache, so a crash mid-write
  never leaves Redis holding data that isn't actually in MySQL.
- **Job tracking is in-memory.** For a single-instance deployment (as
  described in the assignment) this is simplest and fastest. If this
  service needed to run multiple replicas, job state would move to Redis
  or a dedicated table so any instance could serve a status check.
- **Batch inserts** (`CreateInBatches`, 500 rows/batch) keep large imports
  from generating one giant SQL statement or opening thousands of
  round trips.

## Testing

```bash
make test
```

Includes unit tests for Excel header validation (case-insensitivity,
column reordering, missing/short header detection). Extending this with
handler-level tests (using `httptest` + a test MySQL/Redis, e.g. via
`testcontainers-go`) is the natural next step for a fuller test suite.
