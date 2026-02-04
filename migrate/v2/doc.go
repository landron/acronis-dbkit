/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

// Package v2 provides a self-contained database migration engine without external dependencies.
//
// This package replaces github.com/rubenv/sql-migrate with a minimal, efficient implementation
// that maintains full backward compatibility with the original migrate package API.
//
// Key features:
//   - Zero external migration dependencies (no gorp, no sql-migrate)
//   - Support for embedded SQL migrations (embed.FS)
//   - Support for programmatic Go migrations
//   - Per-migration transaction control (TxDisabler interface)
//   - Concurrent migration protection via database locks
//   - Multi-dialect support (MySQL, PostgreSQL, pgx, SQLite, MSSQL)
//
// Basic usage:
//
//	//go:embed migrations/*.sql
//	var migrationFS embed.FS
//
//	func applyMigrations(db *sql.DB) error {
//	    logger := log.NewLogger(...)
//	    mgr, err := v2.NewManager(db, dbkit.DialectPostgres, logger)
//	    if err != nil {
//	        return err
//	    }
//
//	    migrations, err := v2.LoadAllEmbedFSMigrations(migrationFS, "migrations")
//	    if err != nil {
//	        return err
//	    }
//
//	    applied, err := mgr.Run(migrations, v2.DirectionUp)
//	    return err
//	}
//
// See package-level examples for more advanced usage patterns.
package v2
