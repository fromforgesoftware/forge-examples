package app_test

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/fromforgesoftware/go-kit/audit"
	"github.com/fromforgesoftware/go-kit/resource"
	"github.com/fromforgesoftware/go-kit/search"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/app"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/domain"
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

type fakeCharger struct {
	err       error
	gotToken  app.PaymentToken
	gotAmount int
	calls     int
	chargeID  string
}

func (f *fakeCharger) Charge(_ context.Context, token app.PaymentToken, amountCents int, _ string) (string, error) {
	f.calls++
	f.gotToken = token
	f.gotAmount = amountCents
	if f.chargeID == "" {
		f.chargeID = "charge-1"
	}
	return f.chargeID, f.err
}

type fakeNotifier struct {
	called bool
	last   app.AdoptionNotification
}

func (f *fakeNotifier) AdoptionCompleted(_ context.Context, n app.AdoptionNotification) error {
	f.called = true
	f.last = n
	return nil
}

type fakeAudit struct{ events []audit.Event }

func (f *fakeAudit) Emit(_ context.Context, e audit.Event) error {
	f.events = append(f.events, e)
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

type usecaseDeps struct {
	cat      *fakeCatalog
	vendor   *fakeVendor
	charger  *fakeCharger
	notifier *fakeNotifier
	audit    *fakeAudit
	orders   *fakeOrders
}

func newDeps() *usecaseDeps {
	return &usecaseDeps{
		cat:      &fakeCatalog{pet: app.PetInfo{ID: "pet-1", Name: "Rex", Species: "dog", Status: "AVAILABLE"}},
		vendor:   &fakeVendor{},
		charger:  &fakeCharger{},
		notifier: &fakeNotifier{},
		audit:    &fakeAudit{},
		orders:   &fakeOrders{},
	}
}

func (d *usecaseDeps) usecase() app.AdoptionUsecase {
	return app.NewAdoptionUsecase(d.orders, d.cat, d.vendor, d.charger, d.notifier, d.audit, "pay-conn", 5000)
}

func TestPlace_Success(t *testing.T) {
	d := newDeps()
	u := d.usecase()

	order, err := u.Place(context.Background(), "owner-1", "pet-1")
	require.NoError(t, err)
	assert.Equal(t, domain.AdoptionStatusCompleted, order.Status())
	assert.Equal(t, 5000, order.FeeCents())
	assert.Equal(t, 1, d.vendor.calls, "must vend a payment token")
	assert.True(t, d.cat.setStatusCall)
	assert.Equal(t, "ADOPTED", d.cat.statusSet)
	assert.True(t, d.notifier.called)
}

func TestPlace_ChargesWithVendedToken(t *testing.T) {
	d := newDeps()
	_, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.NoError(t, err)
	assert.Equal(t, 1, d.charger.calls, "must charge once")
	assert.Equal(t, "pay-tok", d.charger.gotToken.AccessToken, "charge must use the vended token")
	assert.Equal(t, 5000, d.charger.gotAmount)
}

func TestPlace_ChargeFailureAborts(t *testing.T) {
	d := newDeps()
	d.charger.err = errors.New("psp declined")
	_, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.Error(t, err)
	assert.False(t, d.cat.setStatusCall, "pet must not be marked adopted if the charge fails")
}

func TestPlace_EmitsAuditEvent(t *testing.T) {
	d := newDeps()
	order, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.NoError(t, err)
	require.Len(t, d.audit.events, 1, "must emit one audit event")
	e := d.audit.events[0]
	assert.Equal(t, "adoption.placed", e.Action)
	assert.Equal(t, "adoptions", e.ResourceType)
	assert.Equal(t, order.ID(), e.ResourceID)
	assert.Equal(t, "owner-1", e.ActorID)
}

func TestPlace_NotifiesWithEnrichedPayload(t *testing.T) {
	d := newDeps()
	order, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.NoError(t, err)
	n := d.notifier.last
	assert.Equal(t, order.ID(), n.AdoptionID)
	assert.Equal(t, "owner-1", n.Owner)
	assert.Equal(t, "Rex", n.PetName)
	assert.Equal(t, "dog", n.PetSpecies)
	assert.Equal(t, 5000, n.FeeCents)
}

func TestPlace_PetNotAvailable(t *testing.T) {
	d := newDeps()
	d.cat.pet = app.PetInfo{ID: "pet-1", Status: "ADOPTED"}

	_, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.Error(t, err)
	assert.Zero(t, d.vendor.calls, "must not vend when the pet is unavailable")
	assert.Zero(t, d.charger.calls)
	assert.False(t, d.cat.setStatusCall)
}

func TestPlace_VendFailureAborts(t *testing.T) {
	d := newDeps()
	d.vendor.err = errors.New("kms down")

	_, err := d.usecase().Place(context.Background(), "owner-1", "pet-1")
	require.Error(t, err)
	assert.Zero(t, d.charger.calls, "must not charge if vending fails")
	assert.False(t, d.cat.setStatusCall, "pet must not be marked adopted if payment fails")
}

func TestPlace_RequiresOwnerAndPet(t *testing.T) {
	u := newDeps().usecase()
	_, err := u.Place(context.Background(), "", "pet-1")
	assert.Error(t, err)
	_, err = u.Place(context.Background(), "owner-1", "")
	assert.Error(t, err)
}
