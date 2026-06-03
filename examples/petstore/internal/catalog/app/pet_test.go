package app_test

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/go-kit/application/repository"
	"github.com/fromforgesoftware/go-kit/audit"
	"github.com/fromforgesoftware/go-kit/resource"
	"github.com/fromforgesoftware/go-kit/search"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/app"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/domain"
)

type fakePets struct{ created domain.Pet }

func (f *fakePets) Create(_ context.Context, p domain.Pet) (domain.Pet, error) {
	f.created = p
	return p, nil
}
func (f *fakePets) Get(context.Context, ...search.Option) (domain.Pet, error) { return f.created, nil }
func (f *fakePets) List(context.Context, ...search.Option) (resource.ListResponse[domain.Pet], error) {
	return resource.NewListResponse([]domain.Pet{f.created}, 1), nil
}
func (f *fakePets) Delete(context.Context, repository.DeleteType, ...search.Option) error { return nil }
func (f *fakePets) SetStatus(context.Context, string, domain.PetStatus) error             { return nil }

type fakeAudit struct{ events []audit.Event }

func (f *fakeAudit) Emit(_ context.Context, e audit.Event) error {
	f.events = append(f.events, e)
	return nil
}

func TestCreate_EmitsAuditEvent(t *testing.T) {
	pets := &fakePets{}
	sink := &fakeAudit{}
	u := app.NewPetUsecase(pets, sink)

	created, err := u.Create(context.Background(), domain.NewPet("Rex", "dog"))
	require.NoError(t, err)
	require.Len(t, sink.events, 1, "must emit one audit event")
	e := sink.events[0]
	assert.Equal(t, "pet.create", e.Action)
	assert.Equal(t, "pets", e.ResourceType)
	assert.Equal(t, created.ID(), e.ResourceID)
}

func TestCreate_RejectsMissingFields(t *testing.T) {
	u := app.NewPetUsecase(&fakePets{}, &fakeAudit{})
	_, err := u.Create(context.Background(), domain.NewPet("", "dog"))
	assert.Error(t, err)
}
