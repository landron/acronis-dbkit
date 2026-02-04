/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2

import (
	"database/sql"
)

// Direction defines the direction of database migrations.
type Direction string

// Migration directions.
const (
	DirectionUp   Direction = "up"
	DirectionDown Direction = "down"
)

// NoLimit contains a special value that will not limit the number of migrations to apply.
const NoLimit = 0

// Migration is the interface for all database migrations.
// Migrations can provide SQL statements, Go functions, or both for up/down operations.
type Migration interface {
	// ID returns a unique identifier for the migration.
	// Must be unique across all migrations and sortable (use numeric prefixes like "0001_").
	ID() string

	// UpSQL returns SQL statements to execute when applying the migration.
	// Each statement will be executed in order within a transaction (unless disabled).
	UpSQL() []string

	// DownSQL returns SQL statements to execute when rolling back the migration.
	// Each statement will be executed in order within a transaction (unless disabled).
	DownSQL() []string

	// UpFn returns a function to execute when applying the migration.
	// Called after UpSQL statements (if any).
	UpFn() func(tx *sql.Tx) error

	// DownFn returns a function to execute when rolling back the migration.
	// Called after DownSQL statements (if any).
	DownFn() func(tx *sql.Tx) error
}

// TxDisabler is an optional interface that migrations can implement to disable
// transactional execution. Some database operations (like CREATE INDEX CONCURRENTLY
// in PostgreSQL) cannot run within a transaction.
type TxDisabler interface {
	DisableTx() bool
}

// BaseMigration is a basic implementation of Migration that can be embedded in
// custom migrations to reduce boilerplate.
type BaseMigration struct {
	id      string
	upSQL   []string
	downSQL []string
	upFn    func(tx *sql.Tx) error
	downFn  func(tx *sql.Tx) error
}

// NewMigration creates a new BaseMigration with the given parameters.
func NewMigration(id string, upSQL, downSQL []string, upFn, downFn func(tx *sql.Tx) error) *BaseMigration {
	return &BaseMigration{
		id:      id,
		upSQL:   upSQL,
		downSQL: downSQL,
		upFn:    upFn,
		downFn:  downFn,
	}
}

// ID returns the migration identifier.
func (m *BaseMigration) ID() string {
	return m.id
}

// UpSQL returns the up SQL statements.
func (m *BaseMigration) UpSQL() []string {
	return m.upSQL
}

// DownSQL returns the down SQL statements.
func (m *BaseMigration) DownSQL() []string {
	return m.downSQL
}

// UpFn returns the up function.
func (m *BaseMigration) UpFn() func(tx *sql.Tx) error {
	return m.upFn
}

// DownFn returns the down function.
func (m *BaseMigration) DownFn() func(tx *sql.Tx) error {
	return m.downFn
}
