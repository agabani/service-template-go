# Architecture & Style Guide

## Module

`github.com/agabani/service-template-go`, Go 1.26

## Directory Structure

```
cmd/server/              CLI (cobra): serve, migrate up/down, version
internal/
  config/                Env var parsing only
  domain/                Pure business logic — no adapter/config imports
    errors.go            Sentinel errors: ErrNotFound, ErrConflict, ErrValidation
    {resource}/          entity.go, service.go, repository.go, service_test.go, mocks/
  adapters/
    http/                handler/, jsonapi/, middleware/, openapi/, router.go
    postgres/            db.go, otel.go, migrations.go, {resource}_repository.go
    telemetry/           OTel TracerProvider init
  app/                   DI wiring + graceful shutdown
api/                     openapi.yaml + spec.go (go:embed)
migrations/              {seq}_{desc}.{up|down}.sql + embed.go
tests/integration/       testcontainers-go tests (build tag: integration)
```

## Hexagonal Rules (enforced by `tests/architecture/arch_test.go`)

- `internal/domain/**` — must NOT import `internal/adapters/**` or `internal/config`
- `internal/adapters/**` — may import domain and config; must NOT cross into other adapter sub-packages directly
- Never bypass the architecture tests; fix the architecture instead

## Naming

| Thing                           | Convention                                                       |
| ------------------------------- | ---------------------------------------------------------------- |
| Domain entity                   | `User`, `Account`                                                |
| Domain service impl / interface | `service` (unexported) / `Service`                               |
| Repository interface            | `Repository`                                                     |
| Create/Update inputs            | `CreateInput`, `UpdateInput` (pointer fields = optional)         |
| HTTP handler struct             | `UserHandler`                                                    |
| HTTP attribute DTOs             | `UserAttributes`, `CreateUserAttributes`, `UpdateUserAttributes` |
| Postgres repository             | `UserRepository`                                                 |
| Constructors                    | `New{Type}`                                                      |
| Receivers                       | `h` handler, `r` repository, `s` service                         |
| Context/error/HTTP params       | `ctx`, `err`, `w`/`r`                                            |
| Test functions                  | `Test{Type}_{method}_{scenario}`                                 |
| Generated mocks                 | `mocks/mock_repository.go`, `mocks/mock_service.go`              |

## Domain Patterns

- Entity fields: `uuid.UUID` IDs, `CreatedAt`/`UpdatedAt` timestamps, no JSON tags
- `UpdateInput` uses pointer fields (`*string`) — nil means leave unchanged
- Both `Repository` and `Service` interfaces carry `//go:generate mockgen` directives
- Input validation (required fields, business rules) lives in the service, not the handler
- Wrap domain errors: `fmt.Errorf("%w: email is required", domain.ErrValidation)`

## HTTP Adapter Patterns

- Decode requests via `decodeBody[Attrs](w, r)` (returns `(*Resource[T], bool)`)
- Parse path params via `pathUUID(w, r, "id", "invalid id")`
- Always pass `r.Context()` to service calls
- Early-return after every error: write response, then `return`
- Three private conversion helpers per resource: `{r}Document`, `{r}ToResource`, `{r}Attributes`
- Status codes: 200 list/get, 201 create, 204 delete, 400 decode error, 404/409/422 domain errors, 500 unexpected
- Router uses `spanName(pattern, handler)` wrapper on every route
- Middleware order (outer→inner): `otelhttp` → `Recovery` → `Logger` → `OpenAPI Validator` → mux

## Postgres Adapter Patterns

- `QueryRow` for INSERT/UPDATE RETURNING and SELECT by ID; `Query`+loop for lists; `Exec` for DELETE
- Always check `rows.Err()` after iteration loop; always `defer rows.Close()`
- Check `result.RowsAffected() == 0` on DELETE and map to `ErrNoRows`
- Partial updates: `COALESCE($n, column)` — pass pointer directly, pgx sends NULL for nil
- Private `scan{Resource}(rowScanner)` helper shared by QueryRow and Query paths
- Wrap all errors: `fmt.Errorf("create user: %w", mapError(err))`
- `poolCfg.ConnConfig.Tracer = newQueryTracer()` wired in `NewPool`

## Config

- One typed struct per concern: `HTTPConfig`, `DatabaseConfig`, `TelemetryConfig`
- CLI flags override config only when `cmd.Flags().Changed("flag-name")` is true

## Testing

- Unit tests: closed-box (`package <name>_test`), gomock + testify, mocked service/repo; never test unexported symbols directly
- Integration tests: `//go:build integration` tag, `package integration_test`, testcontainers postgres, real DB + migrations
- Never mock the database in integration tests
- Run mocks: `make generate` (`go generate ./...`)
- Every new non-test file must have a corresponding `_test.go`; tests must be minimalistic with no overlapping coverage between cases

## Code Style

- Avoid comments inside functions; in-function comments are a design smell that the function has grown too large — refactor into reusable functions, not one-off extractions
- Functions must not return more than 3 values; group related returns in a result struct

## Assistant Workflow

After every significant code change, proactively review all affected files against this style guide and fix any violations before considering the task complete. Check: hexagonal layer boundaries, naming conventions, receiver names, test function naming (`Test{Type}_{method}_{scenario}`), HTTP/Postgres/domain patterns.

Keep this file under 150 lines. Prefer concise bullet points; remove or compress content before adding new sections.

## Checklist: Adding a New Resource

1. `internal/domain/{r}/` — entity, repository interface, service interface+impl, service tests; run `go generate`
2. `migrations/` — up/down SQL (`gen_random_uuid()` PK, `created_at`, `updated_at`)
3. `internal/adapters/postgres/{r}_repository.go` — implement Repository with scan helper
4. `api/openapi.yaml` — add schemas, paths, operations
5. `internal/adapters/http/handler/{r}.go` — DTOs, handler, three conversion helpers
6. `internal/adapters/http/router.go` — local interface, RouterDeps field, routes with spanName
7. `internal/app/app.go` — wire repo → service → handler → RouterDeps
8. `tests/integration/{r}_test.go` — integration tests via `setupDatabase`
