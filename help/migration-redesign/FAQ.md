# FAQ: Migration Engine Redesign - Clarifications

**Created**: February 3, 2026

## Question 1: What does "Simpler than sql-migrate's schema (no `Up` column tracking)" mean?

### The Problem with sql-migrate Schema

sql-migrate's approach uses **bidirectional tracking** in a single table:

```sql
-- sql-migrate migrations table
CREATE TABLE gorp_migrations (
    id TEXT PRIMARY KEY,
    applied_at DATETIME,
    up BOOLEAN  -- <-- This column tracks: are we "up" or "down"?
);

-- Example rows:
-- id='0001_create_users', applied_at=2024-01-01 10:00:00, up=TRUE   (migrated up)
-- id='0002_create_posts', applied_at=2024-01-01 10:05:00, up=TRUE   (migrated up)
-- id='0003_add_comments', applied_at=2024-01-01 10:10:00, up=FALSE  (rolled back)
```

**Complexity**: To check "which migrations are applied?", you need to query both `id` AND `up` column:
```sql
SELECT id FROM gorp_migrations WHERE up = TRUE;  -- Get applied migrations
SELECT id FROM gorp_migrations WHERE up = FALSE; -- Get rolled back migrations
```

### Our Approach: Keep the `up` Column (Like sql-migrate)

We actually **do need state tracking** via the `up` column. Here's why:

```sql
-- go-dbkit/migrate/v2 migrations table
CREATE TABLE migrations (
    id VARCHAR(255) PRIMARY KEY,
    applied_at TIMESTAMP NOT NULL,
    up BOOLEAN NOT NULL  -- Critical: tracks state to preserve history
);

-- Example rows:
-- id='0001_create_users', applied_at=2024-01-01 10:00:00, up=1  (applied, still up)
-- id='0002_create_posts', applied_at=2024-01-01 10:05:00, up=1  (applied, still up)
-- id='0003_add_comments', applied_at=2024-01-01 10:10:00, up=0  (was applied, then rolled back)
```

**Why the `up` column is essential**:

1. **History preservation**: Rolled back migrations stay in table with `up=0` (not deleted)
   - Can reconstruct: "When was migration 0003 rolled back?"
   - Can track: "Who rolled back which migrations?"

2. **State clarity**:
   - `up=1` → Migration is currently applied
   - `up=0` AND row exists → Migration was deliberately rolled back (not just unapplied)
   - Row doesn't exist → Migration was never attempted

3. **Rollback verification**:
```sql
-- Check if rollback succeeded
SELECT * FROM migrations WHERE id = '0003_add_comments';
-- If up=0 and row exists → successful rollback with history
-- If row doesn't exist → ambiguous (rolled back or never applied?)
```

4. **Backward compatibility**: sql-migrate tools expect `up` column; keeping it enables gradual migration

### Improvements Over sql-migrate

Instead of replacing the `up` column, we improve elsewhere:

| Aspect | sql-migrate | v2 (our design) |
|--------|-------------|-----------------|
| **Schema** | Same (keep `up` column) | ✅ Same - necessary for compatibility |
| **Removed** | gorp ORM overhead | ✅ Direct SQL (faster, fewer allocations) |
| **SQL parsing** | Limited | ✅ Better error reporting (statement-level) |
| **Removed** | File templating feature | ✅ Simpler, focused feature set |
| **Removed** | gorp transitive dependency | ✅ Eliminates binary bloat |
| **Code size** | ~5000 LOC | ✅ ~800 LOC (cleaner implementation) |

The "simpler" part is not the schema—it's the **implementation** (less code, fewer dependencies).

---

## Question 2: What do `UpSQL()` and `DownSQL()` return?

### The Intent

These methods return **a slice of individual SQL statements**, not a file path or entire file blob:

```go
type Migration interface {
    UpSQL() []string    // Returns: []string{"statement1", "statement2", ...}
    DownSQL() []string  // Returns: []string{"statement1", "statement2", ...}
}
```

### Example: What They Return

**User writes SQL file**:
```sql
-- migrations/0001_create_users.up.sql
CREATE TABLE users (
    id BIGINT PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) UNIQUE NOT NULL,
    created_at TIMESTAMP NOT NULL
);

CREATE INDEX idx_users_email ON users(email);
CREATE INDEX idx_users_created_at ON users(created_at);
```

**After parsing by our loader**:
```go
migration := LoadEmbedFSMigrations(fsys, "migrations", []string{"0001_create_users"})

// migration[0].UpSQL() returns:
[]string{
    "CREATE TABLE users (id BIGINT PRIMARY KEY, name VARCHAR(255) NOT NULL, email VARCHAR(255) UNIQUE NOT NULL, created_at TIMESTAMP NOT NULL)",
    "CREATE INDEX idx_users_email ON users(email)",
    "CREATE INDEX idx_users_created_at ON users(created_at)",
}
```

### Why Split Into Multiple Statements?

**Reason 1: Database Limitations**
```go
// This works:
db.Exec("CREATE TABLE t1 (id INT)");
db.Exec("CREATE TABLE t2 (id INT)");

// This fails (semicolon-separated statements need separate Exec calls):
db.Exec("CREATE TABLE t1 (id INT); CREATE TABLE t2 (id INT)");
// Error: syntax error near semicolon
```

**Reason 2: Better Error Reporting**
```
❌ Bad error:  "migration 0001_create_users failed"
✅ Good error: "migration 0001_create_users failed at statement 2/5 (CREATE INDEX idx_users_email...)"
```

**Reason 3: Database-Specific Statement Limits**
- PostgreSQL can't run `BEGIN`, `COMMIT` inside a transaction
- MySQL has different rules for certain statements
- Splitting allows framework to handle dialect-specific edge cases

### From User Perspective

Users still write simple, natural SQL files:

```sql
-- migrations/0001_create_users.up.sql (one big file)
CREATE TABLE users (...);
CREATE INDEX idx1 ON users(...);
CREATE INDEX idx2 ON users(...);
```

**Behind the scenes**:
1. Framework loads file
2. Framework splits by semicolons into statements
3. Framework executes each statement separately
4. If any fails, user gets precise error

---

## Question 3: Are migrations kept as SQL GO strings? Why not SQL files?

### Answer: Both! We Support Both Approaches

#### **Approach 1: Embedded SQL Files** (via embed.FS)

```go
//go:embed migrations/*.sql
var migrationFS embed.FS

func main() {
    // Load from .sql files embedded in binary
    migrations, err := v2.LoadAllEmbedFSMigrations(migrationFS, "migrations")
    // Each migration loaded from:
    // - migrations/0001_create_users.up.sql
    // - migrations/0001_create_users.down.sql
}
```

**Benefits**:
- ✅ SQL lives in `.sql` files (version control friendly)
- ✅ DBAs can review pure SQL, no Go knowledge needed
- ✅ Embedded in binary (no external file dependencies)
- ✅ Easy for large migrations (separated from code)

**User experience**:
```
migrations/
  ├── 0001_create_users.up.sql
  ├── 0001_create_users.down.sql
  ├── 0002_create_posts.up.sql
  └── 0002_create_posts.down.sql
```

#### **Approach 2: Programmatic Go Strings**

```go
func main() {
    migrations := []v2.Migration{
        v2.NewMigration(
            "0001_create_users",
            []string{"CREATE TABLE users (id BIGINT)"},  // ← Go strings
            []string{"DROP TABLE users"},
            nil, nil,
        ),
    }
}
```

**Benefits**:
- ✅ All code in one place
- ✅ Type safe (Go compiler checks syntax)
- ✅ Inline logic possible
- ✅ Good for simple, inline migrations

**User experience**: Migrate small schema changes without creating files

#### **Approach 3: Fully Custom (Your Choice)**

```go
type DialectAwareMigration struct { }

func (m *DialectAwareMigration) ID() string { return "0001" }

func (m *DialectAwareMigration) UpSQL() []string {
    // Generate SQL based on dialect, environment, etc.
    if os.Getenv("USE_PARTITIONING") == "true" {
        return []string{
            "CREATE TABLE events (id BIGINT) PARTITION BY RANGE (id)",
        }
    }
    return []string{
        "CREATE TABLE events (id BIGINT)",
    }
}

func (m *DialectAwareMigration) DownSQL() []string {
    return []string{"DROP TABLE events"}
}
```

**Benefits**:
- ✅ Conditional logic (environment, dialect-specific)
- ✅ Computed migrations
- ✅ Full Go power

### Recommendation

| Use Case | Approach |
|----------|----------|
| Team has DBAs who review migrations | **Embedded SQL files** |
| Simple/inline changes | **Go strings** |
| Conditional logic (e.g., multi-tenant) | **Custom implementation** |
| Large/complex schema | **Embedded SQL files** |

---

## Question 4: How to Test Performance ("Performance neutral or improved")?

### Performance Metrics We'll Measure

**1. Memory Allocations**

```bash
go test -bench=BenchmarkRunMigration -benchmem ./migrate/v2

# Output example:
# BenchmarkRunMigration100-8    1000    1234567 ns/op    45678 B/op    123 allocs/op
#                               ↑        ↑               ↑              ↑
#                            iterations time per op    bytes per op    allocs per op
```

**What we expect**:
- sql-migrate: ~500KB per 100 migrations (gorp ORM overhead)
- v2: ~300KB per 100 migrations (direct SQL)
- **Goal**: 30-40% fewer allocations

**Why**:
- sql-migrate uses gorp reflection to build queries
- We use plain `database/sql` with hard-coded queries
- Each reflection = allocation

**2. Execution Time**

```bash
# Benchmark with real database
go test -bench=BenchmarkRunMigration_Real -benchmem -timeout=5m ./migrate/v2

# Compare:
# sql-migrate: 1,234,567 ns/op (1.2ms per migration)
# v2:          1,100,000 ns/op (1.1ms per migration)
# Improvement: ~10%
```

**3. Binary Size**

```bash
# Check dependency impact
go build -o app-with-sqlmigrate .  # Build with sql-migrate
go build -o app-with-v2 .          # Build with v2

ls -lh app-with-*
# app-with-sqlmigrate: 45MB
# app-with-v2:         43MB (2MB smaller)
```

**4. Test Plan Implementation**

**Phase: After basic functionality works**

```go
// migrate/v2/benchmark_test.go (to create)

package v2

import (
    "database/sql"
    "testing"
    _ "github.com/mattn/go-sqlite3"
)

func BenchmarkRunMigration_Single(b *testing.B) {
    db := setupTestDB()
    defer db.Close()

    mgr := setupManager(db)
    migration := &BaseMigration{
        id:      "0001",
        upSQL:   []string{"CREATE TABLE t (id INT)"},
        downSQL: []string{"DROP TABLE t"},
    }

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        mgr.Run([]Migration{migration}, DirectionUp)
        mgr.Run([]Migration{migration}, DirectionDown)
    }
}

func BenchmarkRunMigration_Batch100(b *testing.B) {
    // 100 migrations per iteration
    // Measure: time, memory, allocations
}

func BenchmarkParseSQL_Large(b *testing.B) {
    // Parse 10KB SQL file
    // Measure: parsing overhead
}
```

**Compare Against**:
```bash
# Optional: Run same benchmark against current sql-migrate
go test -bench=. ./migrate -benchmem  # Current implementation
go test -bench=. ./migrate/v2 -benchmem  # v2 implementation

# Use benchcmp or benchstat to compare:
go install golang.org/x/perf/cmd/benchstat@latest
benchstat old.txt new.txt
```

### Acceptance Criteria

✅ **Acceptable performance**:
- Memory: Same or better (-30% is excellent)
- Speed: Within ±10% of sql-migrate
- Binary: 1-2MB reduction

❌ **Not acceptable**:
- Significantly slower (>20% regression)
- Allocates more memory (>10% regression)
- Unless there's a documented reason (e.g., added feature)

---

## Summary

| Question | Answer |
|----------|--------|
| **Schema simpler?** | Yes - no `up` column, uses row existence for state |
| **UpSQL/DownSQL?** | Returns slice of SQL statements (split from files), not file paths |
| **SQL in Go strings?** | Both - supports embedded `.sql` files OR hardcoded Go strings |
| **Performance testing?** | Benchmark memory allocs, execution time, binary size; 30% improvement expected |

