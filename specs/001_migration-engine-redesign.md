# Migration Engine Redesign (Open Specs)

**Date**: February 3, 2026
**Status**: Draft

## Problem

`migrate` depends on `github.com/rubenv/sql-migrate`, which adds unmaintained dependencies, a pgx workaround, and features we do not use.

## Goals

1. Remove sql-migrate and its transitive dependencies.
2. Keep the public API compatible with the current `migrate` package.
3. Preserve current functionality: embedded SQL, programmatic migrations, and transaction control.
4. Reduce complexity and operational risk.

## Non-Goals

- SQL templating
- External config files
- CLI tooling
- Advanced rollback beyond down migrations

## Design Summary

### Tracking Table

Same schema as sql-migrate for backward compatibility. A row with `up=1` means "applied"; `up=0` means "rolled back" (preserves history).

- Columns: `id`, `applied_at`, `up` (BOOLEAN)
- Keeps `up` column for backward compatibility and rollback history tracking
- Dialect-specific types: MySQL `DATETIME`, Postgres `TIMESTAMP`, SQLite `TEXT`, MSSQL `DATETIME2`
- "Simpler" refers to implementation (no gorp ORM), not schema

### Migration Interface (Compatibility)

Keep existing interface and semantics:

- `ID()` unique and sortable
- `UpSQL()` / `DownSQL()` return **slices of statements**
- Optional `UpFn()` / `DownFn()` for programmatic logic
- `TxDisabler` supported to disable transactions for specific migrations

### Loading SQL

- Keep `embed.FS` loader
- Parse `<id>.up.sql` / `<id>.down.sql` pairs
- Split files into individual statements for reliable execution and error reporting

### Execution Flow

1. Ensure migrations table exists
2. Read applied IDs
3. Compute pending migrations (up or down)
4. Execute each migration in order
5. Record or delete row in tracking table

### Concurrency

Preferred: advisory locks per dialect. Fallback: table-level locks.

### Compatibility Layer

`migrate` package delegates to `migrate/v2` implementation without API changes.

## Testing

- Unit tests for sorting, parsing, and execution
- Integration tests for all dialects
- Compatibility tests: existing examples must run unchanged

## Performance

Expect fewer allocations and less overhead than sql-migrate (no gorp). Benchmarking will verify no regression.

## Risks

- SQL parsing edge cases (multi-statement files)
- Concurrency correctness across dialects
- Ensuring full API compatibility

## Success Criteria

- [ ] sql-migrate removed from dependencies
- [ ] All existing tests pass unchanged
- [ ] Examples run unchanged
- [ ] Benchmarks show no regression
- [ ] All supported dialects pass integration tests

## References

- Current implementation: https://github.com/acronis/go-dbkit/blob/main/migrate/migrations.go
- sql-migrate issue: https://github.com/rubenv/sql-migrate/issues/211
