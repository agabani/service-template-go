# service-template-go

A production-ready Go microservice template featuring hexagonal architecture, JSON:API, OpenTelemetry, and CRUD for `user` and `account` domains.

## Architecture

```
cmd/server/          CLI entry point (serve, migrate up/down, version)
internal/
  config/            Configuration (env vars, CLI flags, .env)
  domain/            Core business logic — no framework dependencies
    user/            User entity, repository port, service
    account/         Account entity, repository port, service
  adapters/
    http/            Primary adapter — REST API (JSON:API, OpenAPI validation)
    postgres/        Secondary adapter — pgx v5, migrations
    telemetry/       OpenTelemetry setup
  app/               Dependency injection and server lifecycle
api/                 OpenAPI 3.0 specification
migrations/          SQL migrations (embedded in binary)
tests/
  integration/         Integration tests using testcontainers
  architecture/        Architecture enforcement tests
```

Two rules keep the architecture clean, both enforced by `tests/architecture/arch_test.go`:

- **Domain isolation** (`TestDomainIsolation`): domain packages cannot import adapters or config — they must remain framework-free.
- **Adapter isolation** (`TestAdapterIsolation`): no adapter sub-package may import a sibling adapter; adapters must communicate through domain interfaces.

Both rules apply automatically to any new package added under the respective directories.

## Prerequisites

- Go 1.26+
- Docker (for local Postgres and integration tests)
- [golangci-lint](https://golangci-lint.run/usage/install/)

## Getting Started

### 1. Install dev tools

```bash
make setup
```

This installs `goimports` and `mockgen`.

### 2. Install dependencies

```bash
go mod download
```

### 3. Start Postgres

```bash
docker compose up -d postgres
```

### 4. Configure environment

```bash
cp .env.example .env
# Edit .env as needed — defaults work with docker-compose postgres
```

### 5. Run migrations

```bash
make migrate-up
```

### 6. Start the server

```bash
make run
# or: go run ./cmd/server serve
```

The API is available at `http://localhost:8080`.

## Available Commands

```
server serve          Start the HTTP server
server migrate up     Apply all pending migrations
server migrate down   Revert all migrations
server version        Print the version
server --help         Show help
```

Each command accepts flags that override environment variables:

```bash
go run ./cmd/server serve --port 9090 --database-url "postgres://..."
```

## Development

```bash
make check            # fmt + lint + tests — run before opening a PR
make fmt              # Format code
make lint             # Run golangci-lint
make test             # Run unit tests
make test-integration # Run integration tests (requires Docker)
make build            # Build binary to bin/server
make generate         # Regenerate mocks after interface changes
```

## Docker

```bash
# Build the Docker image
make docker-build

# Start all services (postgres + server)
make docker-up

# Stop all services and remove volumes
make docker-down

# With HyperDX for distributed tracing
make docker-observability
# HyperDX UI: http://localhost:8081
```

## API

Full OpenAPI spec: [`api/openapi.yaml`](api/openapi.yaml)

### Users

| Method   | Path          | Description    |
| -------- | ------------- | -------------- |
| `GET`    | `/users`      | List all users |
| `POST`   | `/users`      | Create a user  |
| `GET`    | `/users/{id}` | Get a user     |
| `PATCH`  | `/users/{id}` | Update a user  |
| `DELETE` | `/users/{id}` | Delete a user  |

### Accounts

| Method   | Path                       | Description                |
| -------- | -------------------------- | -------------------------- |
| `GET`    | `/users/{userID}/accounts` | List accounts for a user   |
| `POST`   | `/users/{userID}/accounts` | Create an account for user |
| `GET`    | `/accounts/{id}`           | Get an account             |
| `PATCH`  | `/accounts/{id}`           | Update an account          |
| `DELETE` | `/accounts/{id}`           | Delete an account          |

### Health

| Method | Path            | Description     |
| ------ | --------------- | --------------- |
| `GET`  | `/health/live`  | Liveness probe  |
| `GET`  | `/health/ready` | Readiness probe |

All resource endpoints use [JSON:API](https://jsonapi.org/) (`application/vnd.api+json`).

### Example

```bash
# Create a user
curl -s -X POST http://localhost:8080/users \
  -H "Content-Type: application/vnd.api+json" \
  -d '{"data":{"type":"users","attributes":{"email":"alice@example.com","name":"Alice"}}}' | jq

# Create an account
curl -s -X POST http://localhost:8080/users/<user-id>/accounts \
  -H "Content-Type: application/vnd.api+json" \
  -d '{"data":{"type":"accounts","attributes":{"name":"Savings","currency":"USD"}}}' | jq
```

## Observability

Set `OTEL_EXPORTER_OTLP_ENDPOINT` to export traces to any OTLP-compatible backend (HyperDX, Tempo, etc.).

The endpoint must include a URL scheme — the scheme selects both the protocol and TLS mode:

| Scheme  | Protocol | TLS      | Example                              |
| ------- | -------- | -------- | ------------------------------------ |
| `http`  | HTTP     | insecure | `http://localhost:4318`              |
| `https` | HTTP     | TLS      | `https://collector.example.com:4318` |
| `grpc`  | gRPC     | insecure | `grpc://localhost:4317`              |
| `grpcs` | gRPC     | TLS      | `grpcs://collector.example.com:4317` |

```bash
# HTTP (HyperDX, OpenTelemetry Collector default)
OTEL_EXPORTER_OTLP_ENDPOINT=http://localhost:4318 make run

# gRPC
OTEL_EXPORTER_OTLP_ENDPOINT=grpc://localhost:4317 make run
```

## Configuration Reference

| Variable                      | Default                                               | Description                                                       |
| ----------------------------- | ----------------------------------------------------- | ----------------------------------------------------------------- |
| `HTTP_HOST`                   | `` (all interfaces)                                   | HTTP bind address                                                 |
| `HTTP_PORT`                   | `8080`                                                | HTTP port                                                         |
| `HTTP_SHUTDOWN_TIMEOUT`       | `30s`                                                 | Graceful shutdown timeout                                         |
| `HTTP_READ_TIMEOUT`           | `5s`                                                  | HTTP read timeout                                                 |
| `HTTP_WRITE_TIMEOUT`          | `10s`                                                 | HTTP write timeout                                                |
| `DATABASE_URL`                | `postgres://postgres:postgres@localhost:5432/app?...` | PostgreSQL connection URL                                         |
| `DATABASE_MAX_CONNS`          | `10`                                                  | Max pool connections                                              |
| `DATABASE_MIN_CONNS`          | `2`                                                   | Min pool connections                                              |
| `DATABASE_CONN_MAX_LIFETIME`  | `1h`                                                  | Connection max lifetime                                           |
| `OTEL_EXPORTER_OTLP_ENDPOINT` | `` (disabled)                                         | OTLP endpoint (scheme required: `http`, `https`, `grpc`, `grpcs`) |
| `OTEL_SERVICE_NAME`           | `service-template-go`                                 | Service name for traces                                           |
| `OTEL_SERVICE_VERSION`        | `dev`                                                 | Service version for traces                                        |
