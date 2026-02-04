/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2_test

import (
	"database/sql"
	"testing"

	"github.com/acronis/go-appkit/log"
	_ "github.com/mattn/go-sqlite3"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/acronis/go-dbkit"
	v2 "github.com/acronis/go-dbkit/migrate/v2"
)

func TestManager_BasicMigration(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger, loggerClose := log.NewLogger(&log.Config{Output: log.OutputStderr, Level: log.LevelDebug})
	defer loggerClose()

	mgr, err := v2.NewMigrationsManager(db, dbkit.DialectSQLite, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	// Create a simple migration
	migrations := []v2.Migration{
		v2.NewMigration(
			"0001_test",
			[]string{"CREATE TABLE test_table (id INTEGER PRIMARY KEY, name TEXT)"},
			[]string{"DROP TABLE test_table"},
			nil,
			nil,
		),
	}

	// Run migration up
	count, err := mgr.Run(migrations, v2.DirectionUp)
	require.NoError(t, err, "Failed to run migration up")
	assert.Equal(t, 1, count, "Expected 1 migration applied")

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	require.NoError(t, err, "Table not created")

	// Run migration down
	count, err = mgr.Run(migrations, v2.DirectionDown)
	require.NoError(t, err, "Failed to run migration down")
	assert.Equal(t, 1, count, "Expected 1 migration rolled back")

	// Verify table no longer exists
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	assert.Equal(t, sql.ErrNoRows, err, "Expected table to be dropped")
}

func TestManager_MultipleMigrations(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger, loggerClose := log.NewLogger(&log.Config{Output: log.OutputStderr, Level: log.LevelDebug})
	defer loggerClose()

	mgr, err := v2.NewMigrationsManager(db, dbkit.DialectSQLite, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	migrations := []v2.Migration{
		v2.NewMigration(
			"0001_users",
			[]string{"CREATE TABLE users (id INTEGER PRIMARY KEY, name TEXT)"},
			[]string{"DROP TABLE users"},
			nil,
			nil,
		),
		v2.NewMigration(
			"0002_posts",
			[]string{"CREATE TABLE posts (id INTEGER PRIMARY KEY, user_id INTEGER, title TEXT)"},
			[]string{"DROP TABLE posts"},
			nil,
			nil,
		),
	}

	// Run all migrations up
	count, err := mgr.Run(migrations, v2.DirectionUp)
	require.NoError(t, err, "Failed to run migrations up")
	assert.Equal(t, 2, count, "Expected 2 migrations applied")

	// Run again - should apply 0 migrations (already applied)
	count, err = mgr.Run(migrations, v2.DirectionUp)
	require.NoError(t, err, "Failed to run migrations (second time)")
	assert.Equal(t, 0, count, "Expected 0 migrations applied on second run")
}

func TestManager_RunLimit(t *testing.T) {
	db, err := sql.Open("sqlite3", ":memory:")
	if err != nil {
		t.Fatalf("Failed to open database: %v", err)
	}
	defer db.Close()

	logger, loggerClose := log.NewLogger(&log.Config{Output: log.OutputStderr, Level: log.LevelDebug})
	defer loggerClose()

	mgr, err := v2.NewMigrationsManager(db, dbkit.DialectSQLite, logger)
	if err != nil {
		t.Fatalf("Failed to create manager: %v", err)
	}

	migrations := []v2.Migration{
		v2.NewMigration("0001_first", []string{"CREATE TABLE first (id INTEGER)"}, []string{"DROP TABLE first"}, nil, nil),
		v2.NewMigration("0002_second", []string{"CREATE TABLE second (id INTEGER)"}, []string{"DROP TABLE second"}, nil, nil),
		v2.NewMigration("0003_third", []string{"CREATE TABLE third (id INTEGER)"}, []string{"DROP TABLE third"}, nil, nil),
	}

	// Apply only 1 migration
	count, err := mgr.RunLimit(migrations, v2.DirectionUp, 1)
	require.NoError(t, err, "Failed to run limited migrations")
	assert.Equal(t, 1, count, "Expected 1 migration applied")

	// Apply next 2 migrations
	count, err = mgr.RunLimit(migrations, v2.DirectionUp, 2)
	require.NoError(t, err, "Failed to run remaining migrations")
	assert.Equal(t, 2, count, "Expected 2 migrations applied")
}
