# Architecture Decision Records

| ADR | Status | Title |
|---|---|---|
| [0001](adr-0001-record-architecture-decisions.md) | Accepted | Record architecture decisions |
| [0002](adr-0002-lean-hexagonal-architecture.md) | Accepted | Lean hexagonal architecture with `internal/` layout |
| [0003](adr-0003-postgresql-pgx-hand-written-sql.md) | Accepted | PostgreSQL via `pgx/v5`, hand-written SQL, `golang-migrate` |
| [0004](adr-0004-uuidv7-identifiers.md) | Accepted | UUIDv7 identifiers, prefixed at the HTTP boundary |
| [0005](adr-0005-idempotency-fingerprint.md) | Accepted | Idempotency via unique constraint with payload fingerprint |
| [0006](adr-0006-in-process-polling-worker.md) | Accepted | In-process polling worker with `SKIP LOCKED` |
| [0007](adr-0007-polymorphic-recipient-jsonb.md) | Accepted | Polymorphic recipient as JSONB with search index |
| [0008](adr-0008-basic-auth-technical-accounts.md) | Accepted | Basic Auth with single technical account per environment |
| [0009](adr-0009-integration-tests-testcontainers.md) | Accepted | Integration tests with testcontainers-go (ephemeral PostgreSQL) |
| [0010](adr-0010-callback-fields-event-correlation.md) | Accepted | Callback fields for event correlation in future messaging |