/*
Copyright © 2026 Acronis International GmbH.

Released under MIT license.
*/

package v2_test

import (
	"embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	v2 "github.com/acronis/go-dbkit/migrate/v2"
)

//go:embed testdata/*.sql
var testdataFS embed.FS

func TestLoadAllEmbedFSMigrations(t *testing.T) {
	migrations, err := v2.LoadAllEmbedFSMigrations(testdataFS, "testdata")
	require.NoError(t, err, "Failed to load migrations")

	require.Len(t, migrations, 2, "Expected 2 migrations")

	// Verify IDs are parsed correctly
	expectedIDs := map[string]bool{
		"0001_create_users": false,
		"0002_create_posts": false,
	}

	for _, mig := range migrations {
		id := mig.ID()
		assert.Contains(t, expectedIDs, id, "Unexpected migration ID: %s", id)
		expectedIDs[id] = true
	}

	for id, found := range expectedIDs {
		assert.True(t, found, "Expected migration not found: %s", id)
	}
}

func TestLoadEmbedFSMigrations_Selective(t *testing.T) {
	// Load only one migration
	migrations, err := v2.LoadEmbedFSMigrations(testdataFS, "testdata", []string{"0001_create_users"})
	require.NoError(t, err, "Failed to load migrations")

	require.Len(t, migrations, 1, "Expected 1 migration")

	assert.Equal(t, "0001_create_users", migrations[0].ID(), "Expected migration ID '0001_create_users'")

	// Verify SQL content is loaded
	upSQL := migrations[0].UpSQL()
	assert.NotEmpty(t, upSQL, "Expected up SQL to be loaded")

	downSQL := migrations[0].DownSQL()
	assert.NotEmpty(t, downSQL, "Expected down SQL to be loaded")
}

func TestLoadEmbedFSMigrations_MissingFile(t *testing.T) {
	// Try to load a migration that doesn't exist
	_, err := v2.LoadEmbedFSMigrations(testdataFS, "testdata", []string{"9999_nonexistent"})
	assert.Error(t, err, "Expected error when loading nonexistent migration")
}
