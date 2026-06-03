// Package domain holds the catalog aggregate: Pet, a globally-listed animal
// whose availability the adoptions service flips as orders are placed.
package domain

import (
	"github.com/fromforgesoftware/go-kit/resource"
)

// ResourceTypePet is the JSON:API type for /v1/pets.
const ResourceTypePet resource.Type = "pets"

// PetStatus is a pet's adoption lifecycle.
type PetStatus string

const (
	PetStatusAvailable PetStatus = "AVAILABLE"
	PetStatusPending   PetStatus = "PENDING"
	PetStatusAdopted   PetStatus = "ADOPTED"
)

func (s PetStatus) Valid() bool {
	switch s {
	case PetStatusAvailable, PetStatusPending, PetStatusAdopted:
		return true
	}
	return false
}

// Pet is an animal in the catalog.
type Pet interface {
	resource.Resource
	Name() string
	Species() string
	Status() PetStatus
}

type pet struct {
	resource.Resource

	name    string
	species string
	status  PetStatus
}

type PetOption func(*pet)

func WithPetID(id string) PetOption {
	return func(p *pet) { p.Resource = resource.Update(p.Resource, resource.WithID(id)) }
}
func WithPetStatus(s PetStatus) PetOption {
	return func(p *pet) { p.status = s }
}

// NewPet builds a pet aggregate; status defaults to AVAILABLE.
func NewPet(name, species string, opts ...PetOption) Pet {
	p := &pet{
		Resource: resource.New(resource.WithType(ResourceTypePet)),
		name:     name,
		species:  species,
		status:   PetStatusAvailable,
	}
	for _, opt := range opts {
		opt(p)
	}
	return p
}

func (p *pet) Name() string      { return p.name }
func (p *pet) Species() string   { return p.species }
func (p *pet) Status() PetStatus { return p.status }
