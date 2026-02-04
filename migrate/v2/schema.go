/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/acronis/go-dbkit"
)

// DefaultTableName is the default name for the migrations tracking table.
const DefaultTableName = "schema_migrations"

// getCreateTableSQL returns the dialect-specific DDL for creating the migrations table.
func getCreateTableSQL(dialect dbkit.Dialect, tableName string) (string, error) {
	switch dialect {
	case dbkit.DialectMySQL:
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			applied_at DATETIME NOT NULL,
			up BOOLEAN NOT NULL DEFAULT 1
		)`, tableName), nil

	case dbkit.DialectPostgres, dbkit.DialectPgx:
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			applied_at TIMESTAMP NOT NULL,
			up BOOLEAN NOT NULL DEFAULT true
		)`, tableName), nil

	case dbkit.DialectSQLite:
		return fmt.Sprintf(`CREATE TABLE IF NOT EXISTS %s (
			id VARCHAR(255) NOT NULL PRIMARY KEY,
			applied_at TEXT NOT NULL,
			up BOOLEAN NOT NULL DEFAULT 1
		)`, tableName), nil

	case dbkit.DialectMSSQL:
		// MSSQL doesn't support CREATE TABLE IF NOT EXISTS, use conditional check
		return fmt.Sprintf(`IF NOT EXISTS (SELECT * FROM sys.tables WHERE name = '%s')
			CREATE TABLE %s (
				id VARCHAR(255) NOT NULL PRIMARY KEY,
				applied_at DATETIME2 NOT NULL,
				up BIT NOT NULL DEFAULT 1
			)`, tableName, tableName), nil

	default:
		return "", fmt.Errorf("unsupported dialect: %s", dialect)
	}
}

// ensureTable creates the migrations table if it doesn't exist.
func ensureTable(ctx context.Context, db *sql.DB, dialect dbkit.Dialect, tableName string) error {
	createSQL, err := getCreateTableSQL(dialect, tableName)
	if err != nil {
		return fmt.Errorf("get create table SQL: %w", err)
	}

	if _, err := db.ExecContext(ctx, createSQL); err != nil {
		return fmt.Errorf("create migrations table: %w", err)
	}

	return nil
}
