/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2

import (
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
)

const (
	upSuffix   = ".up.sql"
	downSuffix = ".down.sql"
)

// LoadAllEmbedFSMigrations loads all migrations from an embedded filesystem.
// It expects files in the format: <id>.up.sql and <id>.down.sql
func LoadAllEmbedFSMigrations(fsys embed.FS, dirName string) ([]Migration, error) {
	entries, err := fs.ReadDir(fsys, dirName)
	if err != nil {
		return nil, fmt.Errorf("read directory %s: %w", dirName, err)
	}

	// Collect all migration IDs
	migrationIDs := make(map[string]bool)
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		name := entry.Name()
		if strings.HasSuffix(name, upSuffix) {
			id := strings.TrimSuffix(name, upSuffix)
			migrationIDs[id] = true
		} else if strings.HasSuffix(name, downSuffix) {
			id := strings.TrimSuffix(name, downSuffix)
			migrationIDs[id] = true
		}
	}

	// Convert map to slice
	ids := make([]string, 0, len(migrationIDs))
	for id := range migrationIDs {
		ids = append(ids, id)
	}

	return LoadEmbedFSMigrations(fsys, dirName, ids)
}

// LoadEmbedFSMigrations loads specific migrations by ID from an embedded filesystem.
func LoadEmbedFSMigrations(fsys embed.FS, dirName string, ids []string) ([]Migration, error) {
	migrations := make([]Migration, 0, len(ids))

	for _, id := range ids {
		upFile := filepath.Join(dirName, id+upSuffix)
		downFile := filepath.Join(dirName, id+downSuffix)

		// Read up migration
		upContent, err := fs.ReadFile(fsys, upFile)
		if err != nil {
			return nil, fmt.Errorf("read up migration %s: %w", id, err)
		}

		// Read down migration
		downContent, err := fs.ReadFile(fsys, downFile)
		if err != nil {
			return nil, fmt.Errorf("read down migration %s: %w", id, err)
		}

		upSQL := parseSQL(string(upContent))
		downSQL := parseSQL(string(downContent))

		migrations = append(migrations, &fileMigration{
			id:      id,
			upSQL:   upSQL,
			downSQL: downSQL,
		})
	}

	return migrations, nil
}

// fileMigration represents a migration loaded from SQL files.
type fileMigration struct {
	id      string
	upSQL   []string
	downSQL []string
}

func (m *fileMigration) ID() string {
	return m.id
}

func (m *fileMigration) UpSQL() []string {
	return m.upSQL
}

func (m *fileMigration) DownSQL() []string {
	return m.downSQL
}

func (m *fileMigration) UpFn() func(tx *sql.Tx) error {
	return nil
}

func (m *fileMigration) DownFn() func(tx *sql.Tx) error {
	return nil
}

// parseSQL splits SQL content into individual statements.
// This is a simple implementation that splits on semicolons.
// A more sophisticated parser could handle edge cases like semicolons in strings.
func parseSQL(content string) []string {
	// Remove comments and split by semicolon
	var statements []string
	lines := strings.Split(content, "\n")
	var currentStmt strings.Builder

	for _, line := range lines {
		// Skip SQL comments (simple implementation)
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "--") || strings.HasPrefix(trimmed, "#") {
			continue
		}

		// Handle multi-line statements
		currentStmt.WriteString(line)
		currentStmt.WriteString("\n")

		// Check if statement is complete (ends with semicolon)
		if strings.HasSuffix(trimmed, ";") {
			stmt := strings.TrimSpace(currentStmt.String())
			if stmt != "" && stmt != ";" {
				statements = append(statements, stmt)
			}
			currentStmt.Reset()
		}
	}

	// Add any remaining statement
	if currentStmt.Len() > 0 {
		stmt := strings.TrimSpace(currentStmt.String())
		if stmt != "" && stmt != ";" {
			statements = append(statements, stmt)
		}
	}

	return statements
}
