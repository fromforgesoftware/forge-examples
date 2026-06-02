//go:build integration

package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/db"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/internaltest"
)

func TestAdoptionCreateGetListByOwner(t *testing.T) {
	client := internaltest.GetDB(t)
	t.Cleanup(func() { internaltest.TruncateTables(t, client) })

	ctx := context.Background()
	repo, err := db.NewAdoptionRepository(client)
	require.NoError(t, err)

	created, err := repo.Create(ctx, domain.NewAdoption("owner-a", "11111111-1111-1111-1111-111111111111",
		domain.WithAdoptionStatus(domain.AdoptionStatusCompleted), domain.WithAdoptionFeeCents(5000)))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID())

	_, err = repo.Create(ctx, domain.NewAdoption("owner-b", "22222222-2222-2222-2222-222222222222"))
	require.NoError(t, err)

	t.Run("get", func(t *testing.T) {
		got, err := repo.Get(ctx, internaltest.GetByID(created.ID()))
		require.NoError(t, err)
		assert.Equal(t, "owner-a", got.Owner())
		assert.Equal(t, domain.AdoptionStatusCompleted, got.Status())
		assert.Equal(t, 5000, got.FeeCents())
	})

	t.Run("list scoped by owner", func(t *testing.T) {
		got, err := repo.List(ctx, internaltest.FilterByOwner("owner-a"))
		require.NoError(t, err)
		assert.Equal(t, 1, got.TotalCount())
	})
}
