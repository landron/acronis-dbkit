# go-dbkit AI Coding Agent Instructions

## Project Overview
`go-dbkit` is a Go library for SQL database operations, providing transaction management, automatic retry logic, distributed locking (`distrlock`), schema migrations (`migrate`), and query builder utilities (`dbrutil`, `goquutil`). Supports MySQL, PostgreSQL (lib/pq & pgx), SQLite, and MSSQL.

## Architecture Patterns

### Driver-Specific Retryable Error Registration
Each RDBMS package (`mysql/`, `postgres/`, `pgx/`, `sqlite/`, `mssql/`) registers database-specific retryable errors in its `init()` function using `dbkit.RegisterIsRetryableFunc()`. This enables automatic retry on transient errors (deadlocks, lock timeouts, serialization failures).

**Import pattern**: Use blank imports to register handlers:
```go
import _ "github.com/acronis/go-dbkit/mysql"  // Registers MySQL retryable errors
```

### Transaction Management
- Root package: `dbkit.DoInTx()` wraps transactions with automatic commit/rollback
- `dbrutil` package: `TxRunner.DoInTx()` for dbr query builder integration
- Both support retry policies via functional options (`WithRetryPolicy()`, `WithTxOptions()`)

### Configuration System
`dbkit.Config` struct centralizes connection parameters for all supported databases. Provides:
- DSN generation via `DriverNameAndDSN()` 
- Default values in `constants.go`: `DefaultMaxOpenConns=10`, `DefaultMaxIdleConns=2`, `DefaultConnMaxLifetime=10min`
- Supports environment variable binding through viper/mapstructure

## Development Workflow

### Testing & Quality
**Test execution** (`check.sh`):
```bash
# Fast tests (excludes slow integration tests)
go test $(go list ./... | grep -v distrlock | grep -v pgx | grep -v postgres)

# Full test suite with slow integration tests
SLOW=1 ./check.sh
```

**Environment setup**: Tests expect `MYSQL_USER`, `MYSQL_DATABASE`, `MYSQL_DSN` environment variables. Integration tests use `testcontainers-go` to spin up MariaDB/PostgreSQL containers (see `internal/testing/db.go`).

**Linting**: Uses `golangci-lint-v1` with 600s timeout. Primary config: `.golangci.yml`. Experimental stricter config exists in `.golangci.v2.yml` (currently disabled, 234 issues including 158 nlreturn violations).

### Package Structure
- **Root (`dbkit`)**: Core abstractions - `Config`, `Open()`, `DoInTx()`, DSN generation, retryable error registry
- **`dbrutil/`**: dbr query builder utilities with instrumentation (Prometheus metrics, slow query logging via `EventReceiver` pattern)
- **`distrlock/`**: SQL-based distributed locks using database tables. Use `DoExclusively()` for simple cases or `DBManager`/`DBLock` for advanced control
- **`migrate/`**: Schema migrations with two approaches: embedded SQL files or programmatic Go migrations. Uses `rubenv/sql-migrate` internally
- **`goquutil/`**: Helper functions for goqu query builder (no dedicated README, refer to source)
- **`internal/testing/`**: Test utilities for container-based databases

## Key Conventions

### Query Instrumentation Pattern
Annotate queries with `.Comment()` for metrics and slow query logging:
```go
tx.Select("column").From("table").
    Comment("query:operation_name").  // Prefix with "query:"
    LoadOne(&result)
```
Use `dbrutil.NewCompositeReceiver()` to combine `QueryMetricsEventReceiver` (Prometheus) and `SlowQueryLogEventReceiver`.

### Error Handling
Wrap transaction errors with typed wrappers: `TxBeginError`, `TxCommitError`, `TxRollbackError` (see `dbrutil/dbrutil.go`). These implement `Unwrap()` for retry logic compatibility.

### Dialect Constants
Use `dbkit.Dialect` type (`DialectMySQL`, `DialectPostgres`, `DialectPgx`, `DialectSQLite`, `DialectMSSQLServer`) consistently across config, DSN generation, and migrations.

## Code Style
- Go 1.20+ (see `go.mod`)
- Avoid `gochecknoinits` violations only for driver registration init functions
- Default transaction isolation levels in `constants.go`: `ReadCommitted` for all databases
- Use functional options pattern for API extensibility (`DoInTxOption`, `TxRunnerMiddlewareOpts`)

## Testing Patterns
- Example tests follow naming `Example()` or `ExampleFunctionName()` convention (see `example_test.go`, `distrlock/example_test.go`)
- Integration tests use `internal/testing.MustRunAndOpenTestDB()` for containerized databases
- Mock testing with `github.com/DATA-DOG/go-sqlmock` for unit tests without external dependencies

## Schema Migrations (`migrate/`)

### Two Migration Approaches

**1. Embedded SQL Migrations** (recommended for simple schemas):
- Store migrations as `.up.sql` and `.down.sql` file pairs in directories
- Embed with `//go:embed mysql/*.sql` or `//go:embed postgres/*.sql`
- Load with `migrate.LoadAllEmbedFSMigrations(migrationFS, "mysql")`
- File naming: `0001_create_users_table.up.sql` / `0001_create_users_table.down.sql`
- Migration ID derived from filename (without `.up.sql`/`.down.sql` suffix)

**2. Programmatic Go Migrations** (for dialect-specific logic):
```go
type Migration0001 struct {
    *migrate.NullMigration
    Dialect dbkit.Dialect
}
func (m *Migration0001) ID() string { return "0001_create_users_table" }
func (m *Migration0001) UpSQL() []string {
    switch m.Dialect {
    case dbkit.DialectMySQL:
        return []string{"CREATE TABLE users (id BIGINT AUTO_INCREMENT PRIMARY KEY, ...)"}
    case dbkit.DialectPostgres, dbkit.DialectPgx:
        return []string{"CREATE TABLE users (id BIGINT GENERATED ALWAYS AS IDENTITY PRIMARY KEY, ...)"}
    }
    return nil
}
```

### Migration Manager Pattern
```go
migrationManager, _ := migrate.NewMigrationsManager(dbConn, dialect, logger)
// Run all migrations up
migrationManager.Run(migrations, migrate.MigrationsDirectionUp)
// Run limited migrations
migrationManager.RunLimit(migrations, migrate.MigrationsDirectionDown, 1)
// Check status
status, _ := migrationManager.Status()
lastMig, exists := status.LastAppliedMigration()
```

### Key Migration Conventions
- Migrations tracked in `migrations` table (customizable via `NewMigrationsManagerWithOpts`)
- Migration IDs must be unique and sortable (use numeric prefixes: `0001_`, `0002_`)
- Both `.up.sql` and `.down.sql` required for each migration (enforced by loader)
- **Pgx dialect normalization**: `DialectPgx` converted to `DialectPostgres` internally (sql-migrate limitation)
- `NullMigration` embed pattern reduces boilerplate for Migration interface
- `CustomMigration` for simple cases: `migrate.NewCustomMigration(id, upSQL, downSQL, nil, nil)`

### Advanced Features
- **Transaction control**: Implement `TxDisabler` interface with `DisableTx() bool` for non-transactional migrations
- **Custom migration logic**: Implement `RawMigrator` interface for full control over sql-migrate structures
- **Selective loading**: `LoadEmbedFSMigrations(fs, dirName, []string{"0001_...", "0002_..."})` for specific migrations
- `MigrationsNoLimit` constant (value: 0) to run all pending migrations

### Testing Migrations
- Use SQLite for fast in-memory migration tests (see `migrate/migrations_test.go`)
- Test both up and down migrations to ensure rollback works
- Verify migration order with `Status().AppliedMigrations` slice

## Dependencies
Core: `go-appkit` (retry, logging), `dbr` (query builder), `goqu`, `rubenv/sql-migrate`, `testcontainers-go`
Drivers: `go-sql-driver/mysql`, `lib/pq`, `jackc/pgx/v5`, `mattn/go-sqlite3`, `microsoft/go-mssqldb`
