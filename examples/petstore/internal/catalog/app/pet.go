// Package app holds the catalog usecases.
package app

import (
	"context"
	"fmt"

	"github.com/fromforgesoftware/go-kit/application/repository"
	"github.com/fromforgesoftware/go-kit/application/usecase"
	"github.com/fromforgesoftware/go-kit/audit"
	apierrors "github.com/fromforgesoftware/go-kit/errors"
	"github.com/fromforgesoftware/go-kit/monitoring/logger"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/domain"
	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/platform/auth"
)

// PetRepository persists pets via kit generics, plus a status flip.
type PetRepository interface {
	repository.Creator[domain.Pet]
	repository.Getter[domain.Pet]
	repository.Lister[domain.Pet]
	repository.Deleter
	SetStatus(ctx context.Context, id string, status domain.PetStatus) error
}

// PetUsecase is the catalog management surface.
type PetUsecase interface {
	repository.Getter[domain.Pet]
	repository.Lister[domain.Pet]
	repository.Deleter
	Create(ctx context.Context, p domain.Pet) (domain.Pet, error)
	SetStatus(ctx context.Context, id string, status domain.PetStatus) error
}

type petUsecase struct {
	usecase.Getter[domain.Pet]
	usecase.Lister[domain.Pet]
	repository.Deleter

	pets  PetRepository
	audit audit.Sink
	log   logger.Logger
}

func NewPetUsecase(pets PetRepository, sink audit.Sink) PetUsecase {
	return &petUsecase{
		Getter:  usecase.NewGetter(pets, domain.ResourceTypePet),
		Lister:  usecase.NewLister(pets),
		Deleter: usecase.NewDeleter(pets),
		pets:    pets,
		audit:   sink,
		log:     logger.New(),
	}
}

func (u *petUsecase) Create(ctx context.Context, p domain.Pet) (domain.Pet, error) {
	if p.Name() == "" || p.Species() == "" {
		return nil, apierrors.InvalidArgument("name and species are required")
	}
	created, err := u.pets.Create(ctx, p)
	if err != nil {
		return nil, err
	}

	// Audit the state-changing op. The actor is the authenticated caller from
	// the JWT claims the auth layer put on the context. Auditing is best-effort:
	// a sink failure is logged, not surfaced to the caller.
	claims, _ := auth.ClaimsFromCtx(ctx)
	if err := u.audit.Emit(ctx, audit.Event{
		RealmID:      claims.Issuer,
		ActorID:      auth.OwnerFromCtx(ctx),
		ActorType:    "USER",
		ResourceType: string(domain.ResourceTypePet),
		ResourceID:   created.ID(),
		Action:       "pet.create",
		Summary:      fmt.Sprintf("created pet %q (%s)", created.Name(), created.Species()),
		Metadata:     map[string]any{"species": created.Species(), "status": string(created.Status())},
	}); err != nil {
		u.log.WarnContext(ctx, "audit emit failed", "action", "pet.create", "pet", created.ID(), "error", err)
	}
	return created, nil
}

func (u *petUsecase) SetStatus(ctx context.Context, id string, status domain.PetStatus) error {
	if !status.Valid() {
		return apierrors.InvalidArgument("invalid status")
	}
	return u.pets.SetStatus(ctx, id, status)
}
