# ADR-0009. Integration tests with testcontainers-go (ephemeral PostgreSQL)

- **Status:** Accepted
- **Date:** 2026-08-11
- **Supersedes:** —
- **Superseded by:** —

## Context

Integration tests need a real PostgreSQL to verify claims the database makes: unique constraints, atomic `FOR UPDATE SKIP LOCKED`, and conditional `WHERE lease_token = $1` fencing. A fake database would approve an implementation that races in production because in-memory locks behave differently from PostgreSQL's MVCC under concurrency.

Two approaches were considered: a shared PostgreSQL instance managed by `docker compose` (one schema per test), and ephemeral containers managed by `testcontainers-go` (one container per suite).

## Decision

`testcontainers-go` with `modules/postgres`. Each test suite starts a fresh PostgreSQL 16 Alpine container, applies embedded migrations, runs its tests, and terminates the container in `t.Cleanup`.

**Why testcontainers and not a shared docker compose database.**

- **Zero environment setup.** `go test ./...` runs everywhere Docker is available. No need to `make db-up` first, no port conflicts, no leftover state between runs.
- **Isolation without schemas.** A shared database requires `CREATE SCHEMA` per test to isolate state, but Flyway/migrate run against `public` by default. Schema-per-test is possible but adds complexity to the migration harness. A fresh container is complete isolation with zero schema management overhead.
- **No port allocation.** A shared database on port 5432 conflicts with anything else using that port. Testcontainers allocates a random port, so tests never collide with a development database.
- **No skip-when-down logic.** A shared database test either skips (masking failures in CI) or fails with an unhelpful connection error. A testcontainers test always has a database, so it always runs and always reports real results.

**Cost.** Each suite adds ~5 seconds of container startup. With one suite (`internal/postgres`) this is negligible. A slow integration suite that grows to thousands of tests would warrant a shared pool, but that is a problem for a future ADR.

**Alternative considered: schema-per-test with docker compose.** This was the approach documented in early drafts. It was rejected because:
1. It requires `make db-up` before `go test`, breaking the single-command contract.
2. Schema-per-test needs a migration runner that targets specific schemas — `golang-migrate` targets the database, not a schema.
3. Port conflicts with the development database force the developer to choose which one runs.

## Consequences

**Positive**

- `go test ./...` is the only command needed. No pre-steps.
- Complete isolation: no state leaks between suites or runs.
- No skip logic means integration tests always execute and always report truth.
- Random ports mean no conflict with `docker compose up` for development.

**Negative**

- ~5 seconds overhead per suite for container startup.
- Requires Docker running (already a project requirement).
- Disk usage: each run pulls a fresh PostgreSQL image layer (cached after first pull).

## Compliance

Integration tests live in the same package as the code they test (`internal/postgres/repository_test.go`). Each test file calls a shared `setupDB(t)` helper that starts the container, applies migrations, and registers cleanup. No test creates its own container.

## Notes

This decision reverses an earlier draft that described integration tests as "skipping when no database is reachable." The testcontainers approach makes that skip logic unnecessary: a database is always reachable because the test creates it.