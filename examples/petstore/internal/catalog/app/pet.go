// Package app holds the catalog usecases.
package app

import (
	"context"

	"github.com/fromforgesoftware/go-kit/application/repository"
	"github.com/fromforgesoftware/go-kit/application/usecase"
	apierrors "github.com/fromforgesoftware/go-kit/errors"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/domain"
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

	pets PetRepository
}

func NewPetUsecase(pets PetRepository) PetUsecase {
	return &petUsecase{
		Getter:  usecase.NewGetter(pets, domain.ResourceTypePet),
		Lister:  usecase.NewLister(pets),
		Deleter: usecase.NewDeleter(pets),
		pets:    pets,
	}
}

func (u *petUsecase) Create(ctx context.Context, p domain.Pet) (domain.Pet, error) {
	if p.Name() == "" || p.Species() == "" {
		return nil, apierrors.InvalidArgument("name and species are required")
	}
	return u.pets.Create(ctx, p)
}

func (u *petUsecase) SetStatus(ctx context.Context, id string, status domain.PetStatus) error {
	if !status.Valid() {
		return apierrors.InvalidArgument("invalid status")
	}
	return u.pets.SetStatus(ctx, id, status)
}
