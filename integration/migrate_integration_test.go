//go:build integration

package integration

import (
	"database/sql"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	_ "github.com/lib/pq"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMigrationsApplyAndTablesExist(t *testing.T) {
	dsn := os.Getenv("POSTGRES_TEST_URL")
	if dsn == "" {
		t.Skip("set POSTGRES_TEST_URL for integration tests, e.g. postgres://user:pass@host:5432/db?sslmode=disable")
	}
	db, err := sql.Open("postgres", dsn)
	require.NoError(t, err)
	t.Cleanup(func() { _ = db.Close() })

	driver, err := postgres.WithInstance(db, &postgres.Config{})
	require.NoError(t, err)

	_, thisFile, _, ok := runtime.Caller(0)
	require.True(t, ok)
	repoRoot := filepath.Join(filepath.Dir(thisFile), "..")
	migDir := filepath.Join(repoRoot, "migrations")
	path := filepath.ToSlash(migDir)
	m, err := migrate.NewWithDatabaseInstance("file://"+path, "postgres", driver)
	require.NoError(t, err)
	t.Cleanup(func() { _, _ = m.Close() })

	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		require.NoError(t, err)
	}

	var n int
	err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'core_storage'`).Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 1, n)

	err = db.QueryRow(`SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_name = 'media_processing_jobs'`).Scan(&n)
	require.NoError(t, err)
	assert.Equal(t, 1, n)
}
