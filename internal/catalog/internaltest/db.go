//go:build integration

// Package internaltest holds catalog integration-test helpers.
package internaltest

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/fromforgesoftware/go-kit/migrator"
	"github.com/fromforgesoftware/go-kit/persistence/gormdb"
	"github.com/fromforgesoftware/go-kit/persistence/gormdb/gormdbtest"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/go-kit/filter"
	"github.com/fromforgesoftware/go-kit/search"
	"github.com/fromforgesoftware/go-kit/search/query"

	"github.com/fromforgesoftware/forge-examples/internal/catalog/fields"
)

func GetDB(t *testing.T) *gormdb.DBClient {
	t.Helper()
	tdb := gormdbtest.GetDB(t, "catalog")
	if tdb == nil {
		t.Skip("test database unavailable (docker/gnomock); skipping integration test")
	}
	t.Setenv("DB_HOST", tdb.Host)
	t.Setenv("DB_PORT", fmt.Sprintf("%d", tdb.Port))
	t.Setenv("DB_USER", tdb.User)
	t.Setenv("DB_PASSWORD", tdb.Password)
	t.Setenv("DB_NAME", tdb.DBName)
	t.Setenv("DB_SSL", "disable")
	t.Setenv("DB_SCHEMA", "catalog")
	require.NoError(t, migrator.Up(context.Background(), os.DirFS(migratorDir()), migrator.WithServiceName("catalog")))
	return tdb.DBClient
}

func TruncateTables(t *testing.T, db *gormdb.DBClient) {
	t.Helper()
	require.NoError(t, db.Exec(`TRUNCATE TABLE catalog.pet RESTART IDENTITY CASCADE;`).Error)
}

func GetByID(id string) search.Option {
	return search.WithQueryOpts(query.FilterBy(filter.OpEq, fields.ID, id))
}

func FilterByStatus(status string) search.Option {
	return search.WithQueryOpts(query.FilterBy(filter.OpEq, fields.Status, status))
}

func migratorDir() string {
	_, f, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(f), "..", "..", "..", "cmd", "catalog-migrator")
}
