// Package app holds the adoptions usecases. Placing an adoption orchestrates
// the platform services behind ports: catalog (availability + status),
// Gleipnir (vend a payment-provider token), a payment charge that consumes
// that token, Talos (audit), and Gjallarhorn (notify).
package app

import (
	"context"
	"fmt"

	"github.com/fromforgesoftware/go-kit/application/repository"
	"github.com/fromforgesoftware/go-kit/application/usecase"
	"github.com/fromforgesoftware/go-kit/audit"
	apierrors "github.com/fromforgesoftware/go-kit/errors"
	"github.com/fromforgesoftware/go-kit/monitoring/logger"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/domain"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

// PetInfo is the slice of a catalog pet adoptions needs.
type PetInfo struct {
	ID      string
	Name    string
	Species string
	Status  string
}

// PaymentToken is the secret Gleipnir vends for the payment connector.
type PaymentToken struct {
	AccessToken string
	APIKey      string
	APISecret   string
}

// AdoptionNotification is the enriched payload sent to Gjallarhorn so the
// recipient gets a meaningful adoption-confirmation (pet, owner, fee).
type AdoptionNotification struct {
	AdoptionID string
	Owner      string
	PetID      string
	PetName    string
	PetSpecies string
	FeeCents   int
}

// Catalog is the S2S view of the catalog service.
type Catalog interface {
	GetPet(ctx context.Context, petID string) (PetInfo, error)
	SetPetStatus(ctx context.Context, petID, status string) error
}

// TokenVendor vends a connector secret from Gleipnir.
type TokenVendor interface {
	Vend(ctx context.Context, owner, connectionID string) (PaymentToken, error)
}

// Charger settles the adoption fee using a Gleipnir-vended payment token,
// returning a provider charge id. The petstore ships a mock implementation
// (see platform/mockpayment) standing in for a real PSP call.
type Charger interface {
	Charge(ctx context.Context, token PaymentToken, amountCents int, reference string) (chargeID string, err error)
}

// Notifier announces a completed adoption (→ Gjallarhorn).
type Notifier interface {
	AdoptionCompleted(ctx context.Context, n AdoptionNotification) error
}

// AdoptionRepository persists orders.
type AdoptionRepository interface {
	repository.Creator[domain.Adoption]
	repository.Getter[domain.Adoption]
	repository.Lister[domain.Adoption]
}

// AdoptionUsecase is the adoptions surface.
type AdoptionUsecase interface {
	repository.Getter[domain.Adoption]
	repository.Lister[domain.Adoption]
	Place(ctx context.Context, owner, petID string) (domain.Adoption, error)
}

type adoptionUsecase struct {
	usecase.Getter[domain.Adoption]
	usecase.Lister[domain.Adoption]

	orders        AdoptionRepository
	catalog       Catalog
	vendor        TokenVendor
	charger       Charger
	notifier      Notifier
	audit         audit.Sink
	paymentConnID string
	feeCents      int
	log           logger.Logger
}

func NewAdoptionUsecase(
	orders AdoptionRepository,
	catalog Catalog,
	vendor TokenVendor,
	charger Charger,
	notifier Notifier,
	sink audit.Sink,
	paymentConnID string,
	feeCents int,
) AdoptionUsecase {
	return &adoptionUsecase{
		Getter:        usecase.NewGetter(orders, domain.ResourceTypeAdoption),
		Lister:        usecase.NewLister(orders),
		orders:        orders,
		catalog:       catalog,
		vendor:        vendor,
		charger:       charger,
		notifier:      notifier,
		audit:         sink,
		paymentConnID: paymentConnID,
		feeCents:      feeCents,
		log:           logger.New(),
	}
}

// Place adopts a pet: confirm it's available, vend a payment token from Gleipnir
// and charge the fee, mark the pet adopted in the catalog, persist the order,
// and notify. owner is the opaque key from the caller's token.
func (u *adoptionUsecase) Place(ctx context.Context, owner, petID string) (domain.Adoption, error) {
	if owner == "" || petID == "" {
		return nil, apierrors.InvalidArgument("owner and petId are required")
	}

	pet, err := u.catalog.GetPet(ctx, petID)
	if err != nil {
		return nil, err
	}
	if pet.Status != "AVAILABLE" {
		return nil, apierrors.Conflict("pet is not available for adoption")
	}

	token, err := u.vendor.Vend(ctx, owner, u.paymentConnID)
	if err != nil {
		return nil, apierrors.InternalError("payment authorization failed: " + err.Error())
	}

	// Settle the fee using the Gleipnir-vended token. u.charger is a MOCK
	// stand-in for a real PSP integration (see platform/mockpayment): it proves
	// the vended secret flows downstream into the charge.
	chargeID, err := u.charger.Charge(ctx, token, u.feeCents, "adoption:"+petID)
	if err != nil {
		return nil, apierrors.InternalError("payment charge failed: " + err.Error())
	}

	if err := u.catalog.SetPetStatus(ctx, petID, "ADOPTED"); err != nil {
		return nil, err
	}

	order, err := u.orders.Create(ctx, domain.NewAdoption(owner, petID,
		domain.WithAdoptionStatus(domain.AdoptionStatusCompleted),
		domain.WithAdoptionFeeCents(u.feeCents),
	))
	if err != nil {
		return nil, err
	}

	// Audit the placed adoption. Actor is the authenticated caller from the JWT
	// claims; best-effort so a sink failure never fails the order.
	claims, _ := auth.ClaimsFromCtx(ctx)
	if err := u.audit.Emit(ctx, audit.Event{
		RealmID:      claims.Issuer,
		ActorID:      owner,
		ActorType:    "USER",
		ResourceType: string(domain.ResourceTypeAdoption),
		ResourceID:   order.ID(),
		Action:       "adoption.placed",
		Summary:      fmt.Sprintf("adopted pet %q (%s) for %s", pet.Name, pet.Species, owner),
		Metadata: map[string]any{
			"petId":    petID,
			"chargeId": chargeID,
			"feeCents": u.feeCents,
		},
	}); err != nil {
		u.log.WarnContext(ctx, "audit emit failed", "action", "adoption.placed", "adoption", order.ID(), "error", err)
	}

	if err := u.notifier.AdoptionCompleted(ctx, AdoptionNotification{
		AdoptionID: order.ID(),
		Owner:      owner,
		PetID:      petID,
		PetName:    pet.Name,
		PetSpecies: pet.Species,
		FeeCents:   u.feeCents,
	}); err != nil {
		u.log.WarnContext(ctx, "adoption notification failed", "adoption", order.ID(), "error", err)
	}
	return order, nil
}
