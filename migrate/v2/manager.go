/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2

import (
	"context"
	"database/sql"
	"fmt"
	"sort"
	"time"

	"github.com/acronis/go-appkit/log"

	"github.com/acronis/go-dbkit"
)

// Manager handles database migration execution and tracking.
type Manager struct {
	db        *sql.DB
	dialect   dbkit.Dialect
	logger    log.FieldLogger
	tableName string
}

// ManagerOption is a functional option for Manager configuration.
// Use NewMigrationsManager to create a new Manager instance.
type ManagerOption func(*Manager)

// WithTableName sets a custom migrations table name.
func WithTableName(name string) ManagerOption {
	return func(m *Manager) {
		m.tableName = name
	}
}

// NewMigrationsManager creates a new migration manager.
func NewMigrationsManager(db *sql.DB, dialect dbkit.Dialect, logger log.FieldLogger, opts ...ManagerOption) (*Manager, error) {
	if db == nil {
		return nil, fmt.Errorf("db cannot be nil")
	}
	if logger == nil {
		return nil, fmt.Errorf("logger cannot be nil")
	}

	m := &Manager{
		db:        db,
		dialect:   dialect,
		logger:    logger,
		tableName: DefaultTableName,
	}

	for _, opt := range opts {
		opt(m)
	}

	return m, nil
}

// Run executes all pending migrations in the specified direction.
func (m *Manager) Run(migrations []Migration, direction Direction) (int, error) {
	return m.RunLimit(migrations, direction, NoLimit)
}

// RunLimit executes up to 'limit' migrations in the specified direction.
// Use NoLimit (0) to apply all pending migrations.
func (m *Manager) RunLimit(migrations []Migration, direction Direction, limit int) (int, error) {
	ctx := context.Background()

	// Ensure migrations table exists
	if err := ensureTable(ctx, m.db, m.dialect, m.tableName); err != nil {
		return 0, fmt.Errorf("ensure migrations table: %w", err)
	}

	// Get already applied migrations
	applied, err := m.getAppliedMigrations(ctx)
	if err != nil {
		return 0, fmt.Errorf("get applied migrations: %w", err)
	}

	// Determine which migrations to apply
	toApply := m.filterMigrations(migrations, applied, direction, limit)

	m.logger.Info(fmt.Sprintf("Applying %d migration(s) (%s)", len(toApply), direction))

	// Execute migrations
	count := 0
	for _, mig := range toApply {
		if err := m.executeMigration(ctx, mig, direction); err != nil {
			return count, fmt.Errorf("execute migration %s: %w", mig.ID(), err)
		}
		count++
		m.logger.Info(fmt.Sprintf("Applied migration: %s", mig.ID()))
	}

	return count, nil
}

// getAppliedMigrations returns a map of migration IDs that have been applied.
func (m *Manager) getAppliedMigrations(ctx context.Context) (map[string]time.Time, error) {
	query := fmt.Sprintf("SELECT id, applied_at FROM %s WHERE up = 1", m.tableName)
	rows, err := m.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("query applied migrations: %w", err)
	}
	defer rows.Close()

	applied := make(map[string]time.Time)
	for rows.Next() {
		var id string
		var appliedAtStr string
		if err := rows.Scan(&id, &appliedAtStr); err != nil {
			return nil, fmt.Errorf("scan migration row: %w", err)
		}

		// Parse time string based on dialect
		var appliedAt time.Time
		switch m.dialect {
		case dbkit.DialectSQLite:
			// SQLite stores as TEXT, needs parsing
			t, err := time.Parse("2006-01-02 15:04:05", appliedAtStr)
			if err != nil {
				return nil, fmt.Errorf("parse applied_at time: %w", err)
			}
			appliedAt = t
		default:
			// Other dialects return time.Time directly via driver
			t, err := time.Parse(time.RFC3339, appliedAtStr)
			if err != nil {
				return nil, fmt.Errorf("parse applied_at time: %w", err)
			}
			appliedAt = t
		}

		applied[id] = appliedAt
	}

	return applied, rows.Err()
}

// filterMigrations determines which migrations to apply based on direction and current state.
func (m *Manager) filterMigrations(migrations []Migration, applied map[string]time.Time, direction Direction, limit int) []Migration {
	// Sort migrations by ID
	sorted := make([]Migration, len(migrations))
	copy(sorted, migrations)
	sort.Slice(sorted, func(i, j int) bool {
		return sorted[i].ID() < sorted[j].ID()
	})

	var toApply []Migration

	if direction == DirectionUp {
		// Apply migrations not yet applied
		for _, mig := range sorted {
			if _, exists := applied[mig.ID()]; !exists {
				toApply = append(toApply, mig)
				if limit > 0 && len(toApply) >= limit {
					break
				}
			}
		}
	} else {
		// Rollback applied migrations in reverse order
		for i := len(sorted) - 1; i >= 0; i-- {
			mig := sorted[i]
			if _, exists := applied[mig.ID()]; exists {
				toApply = append(toApply, mig)
				if limit > 0 && len(toApply) >= limit {
					break
				}
			}
		}
	}

	return toApply
}

// executeMigration executes a single migration in the specified direction.
func (m *Manager) executeMigration(ctx context.Context, mig Migration, direction Direction) error {
	// Check if transaction should be disabled
	disableTx := false
	if txDisabler, ok := mig.(TxDisabler); ok {
		disableTx = txDisabler.DisableTx()
	}

	if disableTx {
		return m.executeWithoutTx(ctx, mig, direction)
	}

	return m.executeWithTx(ctx, mig, direction)
}

// executeWithTx executes a migration within a transaction.
func (m *Manager) executeWithTx(ctx context.Context, mig Migration, direction Direction) error {
	tx, err := m.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback() // nolint: errcheck

	if err := m.executeMigrationSteps(ctx, tx, mig, direction); err != nil {
		return err
	}

	if err := m.recordMigration(ctx, tx, mig.ID(), direction == DirectionUp); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}

	return nil
}

// executeWithoutTx executes a migration without a transaction.
func (m *Manager) executeWithoutTx(ctx context.Context, mig Migration, direction Direction) error {
	if err := m.executeMigrationStepsNoTx(ctx, m.db, mig, direction); err != nil {
		return err
	}

	// Record migration outside transaction
	if err := m.recordMigrationNoTx(ctx, mig.ID(), direction == DirectionUp); err != nil {
		return fmt.Errorf("record migration: %w", err)
	}

	return nil
}

// executeMigrationSteps executes the SQL and function steps of a migration (within tx).
func (m *Manager) executeMigrationSteps(ctx context.Context, tx *sql.Tx, mig Migration, direction Direction) error {
	var statements []string
	var fn func(tx *sql.Tx) error

	if direction == DirectionUp {
		statements = mig.UpSQL()
		fn = mig.UpFn()
	} else {
		statements = mig.DownSQL()
		fn = mig.DownFn()
	}

	// Execute SQL statements
	for i, stmt := range statements {
		if stmt == "" {
			continue
		}
		if _, err := tx.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute statement %d: %w", i+1, err)
		}
	}

	// Execute function if provided
	if fn != nil {
		if err := fn(tx); err != nil {
			return fmt.Errorf("execute function: %w", err)
		}
	}

	return nil
}

// executeMigrationStepsNoTx executes the SQL and function steps without a transaction.
func (m *Manager) executeMigrationStepsNoTx(ctx context.Context, db *sql.DB, mig Migration, direction Direction) error {
	var statements []string

	if direction == DirectionUp {
		statements = mig.UpSQL()
	} else {
		statements = mig.DownSQL()
	}

	// Execute SQL statements
	for i, stmt := range statements {
		if stmt == "" {
			continue
		}
		if _, err := db.ExecContext(ctx, stmt); err != nil {
			return fmt.Errorf("execute statement %d: %w", i+1, err)
		}
	}

	// Note: Functions not supported in non-tx mode
	// This is a limitation we accept for simplicity

	return nil
}

// recordMigration records a migration as applied or unapplied (within tx).
func (m *Manager) recordMigration(ctx context.Context, tx *sql.Tx, id string, applied bool) error {
	if applied {
		query := fmt.Sprintf("INSERT INTO %s (id, applied_at) VALUES (?, ?)", m.tableName)
		if m.dialect == dbkit.DialectPostgres || m.dialect == dbkit.DialectPgx {
			query = fmt.Sprintf("INSERT INTO %s (id, applied_at) VALUES ($1, $2)", m.tableName)
		}
		if _, err := tx.ExecContext(ctx, query, id, time.Now()); err != nil {
			return fmt.Errorf("insert migration record: %w", err)
		}
	} else {
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.tableName)
		if m.dialect == dbkit.DialectPostgres || m.dialect == dbkit.DialectPgx {
			query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", m.tableName)
		}
		if _, err := tx.ExecContext(ctx, query, id); err != nil {
			return fmt.Errorf("delete migration record: %w", err)
		}
	}

	return nil
}

// recordMigrationNoTx records a migration without a transaction.
func (m *Manager) recordMigrationNoTx(ctx context.Context, id string, applied bool) error {
	if applied {
		query := fmt.Sprintf("INSERT INTO %s (id, applied_at) VALUES (?, ?)", m.tableName)
		if m.dialect == dbkit.DialectPostgres || m.dialect == dbkit.DialectPgx {
			query = fmt.Sprintf("INSERT INTO %s (id, applied_at) VALUES ($1, $2)", m.tableName)
		}
		if _, err := m.db.ExecContext(ctx, query, id, time.Now()); err != nil {
			return fmt.Errorf("insert migration record: %w", err)
		}
	} else {
		query := fmt.Sprintf("DELETE FROM %s WHERE id = ?", m.tableName)
		if m.dialect == dbkit.DialectPostgres || m.dialect == dbkit.DialectPgx {
			query = fmt.Sprintf("DELETE FROM %s WHERE id = $1", m.tableName)
		}
		if _, err := m.db.ExecContext(ctx, query, id); err != nil {
			return fmt.Errorf("delete migration record: %w", err)
		}
	}

	return nil
}
