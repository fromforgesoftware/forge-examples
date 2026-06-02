// Package app holds the adoptions usecases. Placing an adoption orchestrates
// three platform services: catalog (availability + status), Gleipnir (vend a
// payment-provider token), and Herald (notify) — each behind a port.
package app

import (
	"context"

	"github.com/fromforgesoftware/go-kit/application/repository"
	"github.com/fromforgesoftware/go-kit/application/usecase"
	apierrors "github.com/fromforgesoftware/go-kit/errors"
	"github.com/fromforgesoftware/go-kit/monitoring/logger"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
)

// PetInfo is the slice of a catalog pet adoptions needs.
type PetInfo struct {
	ID     string
	Status string
}

// PaymentToken is the secret Gleipnir vends for the payment connector.
type PaymentToken struct {
	AccessToken string
	APIKey      string
	APISecret   string
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

// Notifier announces a completed adoption (→ Herald).
type Notifier interface {
	AdoptionCompleted(ctx context.Context, a domain.Adoption) error
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
	notifier      Notifier
	paymentConnID string
	feeCents      int
	log           logger.Logger
}

func NewAdoptionUsecase(
	orders AdoptionRepository,
	catalog Catalog,
	vendor TokenVendor,
	notifier Notifier,
	paymentConnID string,
	feeCents int,
) AdoptionUsecase {
	return &adoptionUsecase{
		Getter:        usecase.NewGetter(orders, domain.ResourceTypeAdoption),
		Lister:        usecase.NewLister(orders),
		orders:        orders,
		catalog:       catalog,
		vendor:        vendor,
		notifier:      notifier,
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

	if _, err := u.vendor.Vend(ctx, owner, u.paymentConnID); err != nil {
		return nil, apierrors.InternalError("payment authorization failed: " + err.Error())
	}
	// A real charge would call the payment provider with the vended token here.

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

	if err := u.notifier.AdoptionCompleted(ctx, order); err != nil {
		u.log.WarnContext(ctx, "adoption notification failed", "adoption", order.ID(), "error", err)
	}
	return order, nil
}
