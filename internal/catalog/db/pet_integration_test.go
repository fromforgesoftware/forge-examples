//go:build integration

package db_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/go-kit/application/repository"

	"github.com/fromforgesoftware/forge-examples/internal/catalog/db"
	"github.com/fromforgesoftware/forge-examples/internal/catalog/domain"
	"github.com/fromforgesoftware/forge-examples/internal/catalog/internaltest"
)

func TestPetCreateGetSetStatusDelete(t *testing.T) {
	client := internaltest.GetDB(t)
	t.Cleanup(func() { internaltest.TruncateTables(t, client) })

	ctx := context.Background()
	repo, err := db.NewPetRepository(client)
	require.NoError(t, err)

	created, err := repo.Create(ctx, domain.NewPet("Rex", "dog"))
	require.NoError(t, err)
	require.NotEmpty(t, created.ID())
	assert.Equal(t, domain.PetStatusAvailable, created.Status())

	t.Run("get", func(t *testing.T) {
		got, err := repo.Get(ctx, internaltest.GetByID(created.ID()))
		require.NoError(t, err)
		assert.Equal(t, "Rex", got.Name())
		assert.Equal(t, "dog", got.Species())
	})

	t.Run("set status", func(t *testing.T) {
		require.NoError(t, repo.SetStatus(ctx, created.ID(), domain.PetStatusAdopted))
		got, err := repo.Get(ctx, internaltest.GetByID(created.ID()))
		require.NoError(t, err)
		assert.Equal(t, domain.PetStatusAdopted, got.Status())
	})

	t.Run("delete", func(t *testing.T) {
		require.NoError(t, repo.Delete(ctx, repository.DeleteTypeHard, internaltest.GetByID(created.ID())))
		_, err := repo.Get(ctx, internaltest.GetByID(created.ID()))
		require.Error(t, err)
	})
}

func TestPetListFilterByStatus(t *testing.T) {
	client := internaltest.GetDB(t)
	t.Cleanup(func() { internaltest.TruncateTables(t, client) })

	ctx := context.Background()
	repo, err := db.NewPetRepository(client)
	require.NoError(t, err)

	_, err = repo.Create(ctx, domain.NewPet("Rex", "dog"))
	require.NoError(t, err)
	_, err = repo.Create(ctx, domain.NewPet("Milo", "cat", domain.WithPetStatus(domain.PetStatusAdopted)))
	require.NoError(t, err)

	got, err := repo.List(ctx, internaltest.FilterByStatus(string(domain.PetStatusAvailable)))
	require.NoError(t, err)
	assert.Equal(t, 1, got.TotalCount())
	assert.Equal(t, "Rex", got.Results()[0].Name())
}
