/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2

import (
	"testing"

	"github.com/acronis/go-dbkit"
)

func TestGetCreateTableSQL_AllDialects(t *testing.T) {
	tableName := "migrations"

	tests := []struct {
		dialect      dbkit.Dialect
		wantContains []string
	}{
		{
			dialect:      dbkit.DialectMySQL,
			wantContains: []string{"CREATE TABLE IF NOT EXISTS", "DATETIME", "BOOLEAN"},
		},
		{
			dialect:      dbkit.DialectPostgres,
			wantContains: []string{"CREATE TABLE IF NOT EXISTS", "TIMESTAMP", "BOOLEAN"},
		},
		{
			dialect:      dbkit.DialectPgx,
			wantContains: []string{"CREATE TABLE IF NOT EXISTS", "TIMESTAMP", "BOOLEAN"},
		},
		{
			dialect:      dbkit.DialectSQLite,
			wantContains: []string{"CREATE TABLE IF NOT EXISTS", "TEXT", "BOOLEAN"},
		},
		{
			dialect:      dbkit.DialectMSSQL,
			wantContains: []string{"IF NOT EXISTS", "DATETIME2", "BIT"},
		},
	}

	for _, tt := range tests {
		t.Run(string(tt.dialect), func(t *testing.T) {
			sql, err := getCreateTableSQL(tt.dialect, tableName)
			if err != nil {
				t.Fatalf("getCreateTableSQL failed: %v", err)
			}

			for _, want := range tt.wantContains {
				if !contains(sql, want) {
					t.Errorf("SQL missing expected string %q:\n%s", want, sql)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
