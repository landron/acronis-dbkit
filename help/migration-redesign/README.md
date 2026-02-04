# go-dbkit Migration Engine Redesign - Document Index

**Project**: Replace `github.com/rubenv/sql-migrate` with native implementation  
**Branch**: `feature/migrate-v2` (recommended)  
**Status**: Specs complete, prototype implementation started  
**Timeline**: 2-3 weeks to production

---

## 📋 Documentation Files

### 1. **[specs/001_migration-engine-redesign.md](001_migration-engine-redesign.md)** ⭐ START HERE
**Comprehensive technical specification (410 lines)**

Contains:
- Problem statement & goals
- Architecture & design decisions
- Database compatibility for all 5 dialects
- Migration execution flow (with diagrams)
- Testing strategy
- Concurrency control approach
- Performance testing methodology

**Key sections**:
- ✅ Schema comparison (simpler, no `up` column)
- ✅ Migration interface design (UpSQL/DownSQL explained)
- ✅ File loading from embed.FS
- ✅ Performance benchmarking approach

---

### 2. **[specs/plan.md](plan.md)**
**7-phase implementation plan (400+ lines)**

**Timeline breakdown**:
- Phase 1 (Days 1-3): Package structure & schema DDL
- Phase 2 (Days 4-7): Core migration engine
- Phase 3 (Days 8-10): Transaction control & locking
- Phase 4 (Days 11-13): Backward compatibility wrapper
- Phase 5 (Days 14-15): Integration testing (all 5 databases)
- Phase 6 (Days 16-17): Documentation
- Phase 7 (Days 18-19): Cleanup & release

**Includes**:
- Detailed task checklists per phase
- Success criteria & metrics
- Rollback plan
- Testing strategy for each phase

---

### 3. **[specs/FAQ.md](FAQ.md)** 📖 ANSWERS YOUR QUESTIONS
**In-depth clarifications (250+ lines)**

Addresses your specific questions:

1. **"What does 'simpler schema' mean?"**
   - Shows sql-migrate's `up BOOLEAN` column
   - Explains our row-existence approach
   - Visual comparison table

2. **"What do UpSQL/DownSQL return?"**
   - Returns slice of individual SQL statements
   - Example: 5KB file → 3 statements
   - Why split: error reporting, database limits, dialect support

3. **"Are migrations SQL strings? Why not files?"**
   - **Answer**: BOTH supported
   - Embedded .sql files via embed.FS (preferred for teams)
   - Go strings via `NewMigration()` (preferred for inline)
   - Custom implementations (preferred for complex logic)

4. **"How to test performance?"**
   - Memory allocation benchmarks (expect 30-40% improvement)
   - Execution time benchmarks (expect 5-15% faster)
   - Binary size reduction (expect 1-2MB)
   - Test plan with Go benchmark code samples

---

### 4. **[specs/VISUAL_COMPARISON.md](VISUAL_COMPARISON.md)** 🎨 VISUAL GUIDE
**Side-by-side comparisons with ASCII diagrams**

Contains:
- Schema comparison (tables with example data)
- Migration interface: sql-migrate vs v2
- File organization examples
- Execution flow diagrams
- SQL parsing example
- Dependency tree before/after
- Performance metrics comparison
- API compatibility matrix

---

## 🗂️ Prototype Implementation

### Location: `migrate/v2/`

**Core files**:
- `doc.go` - Package documentation
- `migration.go` - Interfaces: Migration, TxDisabler, BaseMigration
- `schema.go` - Dialect-specific DDL generation
- `manager.go` - Main execution engine (~250 LOC)
- `loader.go` - embed.FS file loading & SQL parsing

**Test files**:
- `manager_test.go` - Manager functionality tests
- `loader_test.go` - File loading tests  
- `schema_test.go` - Schema DDL tests

**Test data**:
- `testdata/0001_create_users.up.sql`
- `testdata/0001_create_users.down.sql`
- `testdata/0002_create_posts.up.sql`
- `testdata/0002_create_posts.down.sql`

---

## 🎯 Quick Answers to Your Questions

| Question | Answer | See |
|----------|--------|-----|
| Why "simpler schema"? | No `up BOOLEAN` column. Row existence = state. | FAQ #1, VISUAL #1 |
| What UpSQL/DownSQL return? | Slice of individual SQL statements, not files. | FAQ #2, VISUAL #5 |
| Strings or files? | Both! Embedded .sql files OR Go strings. | FAQ #3, VISUAL #3 |
| Performance testing? | Benchmark allocs, time, binary size. | FAQ #4, spec section |

---

## 🚀 Next Steps

### To Review
1. Read [specs/001_migration-engine-redesign.md](001_migration-engine-redesign.md)
2. Review visual comparisons in [VISUAL_COMPARISON.md](VISUAL_COMPARISON.md)
3. Check implementation plan in [specs/plan.md](specs/plan.md)

### To Implement
1. Create feature branch: `git checkout -b feature/migrate-v2`
2. Follow Phase 1-2 in [specs/plan.md](specs/plan.md)
3. Run tests: `go test ./migrate/v2/...`
4. Performance benchmark: `go test -bench=. -benchmem ./migrate/v2`

### Key Decisions Made
✅ Zero external dependencies (no gorp, no sql-migrate)  
✅ Backward compatible API  
✅ Support both SQL files (embed.FS) and Go strings  
✅ Simpler schema (no `up` column)  
✅ 30-40% fewer memory allocations expected  

---

## 📊 Success Metrics

- [ ] All existing tests pass without modification
- [ ] Examples run unchanged
- [ ] 30-40% fewer memory allocations
- [ ] 5-15% faster execution
- [ ] 1-2MB binary size reduction
- [ ] Integration tests for all 5 databases (MySQL, PostgreSQL, pgx, SQLite, MSSQL)
- [ ] 90%+ code coverage in v2 package
- [ ] Zero external migration dependencies

---

## 💡 Key Design Decisions

1. **Schema**: Simpler (no `up` column), row existence = state
2. **SQL**: Support both files (embed.FS) and Go strings
3. **Parser**: Split SQL into statements (better errors, compatibility)
4. **Transactions**: Per-migration control via TxDisabler interface
5. **Concurrency**: Advisory locks (fallback to row locks)
6. **Compatibility**: Wrapper in current `migrate` package (drop-in replacement)

---

## 📞 Questions?

Check [FAQ.md](FAQ.md) for detailed answers to:
- Schema simplification
- UpSQL/DownSQL design
- SQL files vs Go strings
- Performance testing approach

