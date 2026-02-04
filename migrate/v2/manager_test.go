/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2_test

import (
	"database/sql"
	"testing"

	_ "github.com/mattn/go-sqlite3"

	"github.com/acronis/go-appkit/log"

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
	if err != nil {
		t.Fatalf("Failed to run migration up: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 migration applied, got %d", count)
	}

	// Verify table exists
	var tableName string
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	if err != nil {
		t.Fatalf("Table not created: %v", err)
	}

	// Run migration down
	count, err = mgr.Run(migrations, v2.DirectionDown)
	if err != nil {
		t.Fatalf("Failed to run migration down: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 migration rolled back, got %d", count)
	}

	// Verify table no longer exists
	err = db.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name='test_table'").Scan(&tableName)
	if err != sql.ErrNoRows {
		t.Errorf("Expected table to be dropped, got: %v", err)
	}
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
	if err != nil {
		t.Fatalf("Failed to run migrations up: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 migrations applied, got %d", count)
	}

	// Run again - should apply 0 migrations (already applied)
	count, err = mgr.Run(migrations, v2.DirectionUp)
	if err != nil {
		t.Fatalf("Failed to run migrations (second time): %v", err)
	}
	if count != 0 {
		t.Errorf("Expected 0 migrations applied on second run, got %d", count)
	}
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
	if err != nil {
		t.Fatalf("Failed to run limited migrations: %v", err)
	}
	if count != 1 {
		t.Errorf("Expected 1 migration applied, got %d", count)
	}

	// Apply next 2 migrations
	count, err = mgr.RunLimit(migrations, v2.DirectionUp, 2)
	if err != nil {
		t.Fatalf("Failed to run remaining migrations: %v", err)
	}
	if count != 2 {
		t.Errorf("Expected 2 migrations applied, got %d", count)
	}
}
