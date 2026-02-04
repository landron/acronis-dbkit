/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2_test

import (
	"embed"
	"testing"

	v2 "github.com/acronis/go-dbkit/migrate/v2"
)

//go:embed testdata/*.sql
var testdataFS embed.FS

func TestLoadAllEmbedFSMigrations(t *testing.T) {
	migrations, err := v2.LoadAllEmbedFSMigrations(testdataFS, "testdata")
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrations) != 2 {
		t.Fatalf("Expected 2 migrations, got %d", len(migrations))
	}

	// Verify IDs are parsed correctly
	expectedIDs := map[string]bool{
		"0001_create_users": false,
		"0002_create_posts": false,
	}

	for _, mig := range migrations {
		id := mig.ID()
		if _, ok := expectedIDs[id]; !ok {
			t.Errorf("Unexpected migration ID: %s", id)
		}
		expectedIDs[id] = true
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("Expected migration not found: %s", id)
		}
	}
}

func TestLoadEmbedFSMigrations_Selective(t *testing.T) {
	// Load only one migration
	migrations, err := v2.LoadEmbedFSMigrations(testdataFS, "testdata", []string{"0001_create_users"})
	if err != nil {
		t.Fatalf("Failed to load migrations: %v", err)
	}

	if len(migrations) != 1 {
		t.Fatalf("Expected 1 migration, got %d", len(migrations))
	}

	if migrations[0].ID() != "0001_create_users" {
		t.Errorf("Expected migration ID '0001_create_users', got '%s'", migrations[0].ID())
	}

	// Verify SQL content is loaded
	upSQL := migrations[0].UpSQL()
	if len(upSQL) == 0 {
		t.Error("Expected up SQL to be loaded")
	}

	downSQL := migrations[0].DownSQL()
	if len(downSQL) == 0 {
		t.Error("Expected down SQL to be loaded")
	}
}

func TestLoadEmbedFSMigrations_MissingFile(t *testing.T) {
	// Try to load a migration that doesn't exist
	_, err := v2.LoadEmbedFSMigrations(testdataFS, "testdata", []string{"9999_nonexistent"})
	if err == nil {
		t.Error("Expected error when loading nonexistent migration")
	}
}
