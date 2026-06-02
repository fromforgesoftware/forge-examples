package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/go-kit/resource"
	"github.com/fromforgesoftware/go-kit/search"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/app"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
)

type fakeCatalog struct {
	pet           app.PetInfo
	getErr        error
	statusSet     string
	setStatusErr  error
	setStatusCall bool
}

func (f *fakeCatalog) GetPet(context.Context, string) (app.PetInfo, error) { return f.pet, f.getErr }
func (f *fakeCatalog) SetPetStatus(_ context.Context, _, status string) error {
	f.setStatusCall = true
	f.statusSet = status
	return f.setStatusErr
}

type fakeVendor struct {
	err   error
	calls int
}

func (f *fakeVendor) Vend(context.Context, string, string) (app.PaymentToken, error) {
	f.calls++
	return app.PaymentToken{AccessToken: "pay-tok"}, f.err
}

type fakeNotifier struct{ called bool }

func (f *fakeNotifier) AdoptionCompleted(context.Context, domain.Adoption) error {
	f.called = true
	return nil
}

type fakeOrders struct{ created domain.Adoption }

func (f *fakeOrders) Create(_ context.Context, a domain.Adoption) (domain.Adoption, error) {
	f.created = a
	return a, nil
}
func (f *fakeOrders) Get(context.Context, ...search.Option) (domain.Adoption, error) {
	return f.created, nil
}
func (f *fakeOrders) List(context.Context, ...search.Option) (resource.ListResponse[domain.Adoption], error) {
	return resource.NewListResponse([]domain.Adoption{f.created}, 1), nil
}

func newUsecase(cat *fakeCatalog, vendor *fakeVendor, notifier *fakeNotifier, orders *fakeOrders) app.AdoptionUsecase {
	return app.NewAdoptionUsecase(orders, cat, vendor, notifier, "pay-conn", 5000)
}

func TestPlace_Success(t *testing.T) {
	cat := &fakeCatalog{pet: app.PetInfo{ID: "pet-1", Status: "AVAILABLE"}}
	vendor := &fakeVendor{}
	notifier := &fakeNotifier{}
	u := newUsecase(cat, vendor, notifier, &fakeOrders{})

	order, err := u.Place(context.Background(), "owner-1", "pet-1")
	require.NoError(t, err)
	assert.Equal(t, domain.AdoptionStatusCompleted, order.Status())
	assert.Equal(t, 5000, order.FeeCents())
	assert.Equal(t, 1, vendor.calls, "must vend a payment token")
	assert.True(t, cat.setStatusCall)
	assert.Equal(t, "ADOPTED", cat.statusSet)
	assert.True(t, notifier.called)
}

func TestPlace_PetNotAvailable(t *testing.T) {
	cat := &fakeCatalog{pet: app.PetInfo{ID: "pet-1", Status: "ADOPTED"}}
	vendor := &fakeVendor{}
	u := newUsecase(cat, vendor, &fakeNotifier{}, &fakeOrders{})

	_, err := u.Place(context.Background(), "owner-1", "pet-1")
	require.Error(t, err)
	assert.Zero(t, vendor.calls, "must not vend when the pet is unavailable")
	assert.False(t, cat.setStatusCall)
}

func TestPlace_VendFailureAborts(t *testing.T) {
	cat := &fakeCatalog{pet: app.PetInfo{ID: "pet-1", Status: "AVAILABLE"}}
	vendor := &fakeVendor{err: errors.New("kms down")}
	u := newUsecase(cat, vendor, &fakeNotifier{}, &fakeOrders{})

	_, err := u.Place(context.Background(), "owner-1", "pet-1")
	require.Error(t, err)
	assert.False(t, cat.setStatusCall, "pet must not be marked adopted if payment fails")
}

func TestPlace_RequiresOwnerAndPet(t *testing.T) {
	u := newUsecase(&fakeCatalog{}, &fakeVendor{}, &fakeNotifier{}, &fakeOrders{})
	_, err := u.Place(context.Background(), "", "pet-1")
	assert.Error(t, err)
	_, err = u.Place(context.Background(), "owner-1", "")
	assert.Error(t, err)
}
