// Package domain holds the adoptions aggregate: an Adoption order placed by an
// owner for a catalog pet, settled through a Gleipnir-vended payment token.
package domain

import (
	"github.com/fromforgesoftware/go-kit/resource"
)

// ResourceTypeAdoption is the JSON:API type for /v1/adoptions.
const ResourceTypeAdoption resource.Type = "adoptions"

// AdoptionStatus is the order lifecycle.
type AdoptionStatus string

const (
	AdoptionStatusPlaced    AdoptionStatus = "PLACED"
	AdoptionStatusCompleted AdoptionStatus = "COMPLETED"
	AdoptionStatusFailed    AdoptionStatus = "FAILED"
)

// Adoption is an owner's order to adopt a pet.
type Adoption interface {
	resource.Resource
	Owner() string
	PetID() string
	Status() AdoptionStatus
	FeeCents() int
}

type adoption struct {
	resource.Resource

	owner    string
	petID    string
	status   AdoptionStatus
	feeCents int
}

type AdoptionOption func(*adoption)

func WithAdoptionID(id string) AdoptionOption {
	return func(a *adoption) { a.Resource = resource.Update(a.Resource, resource.WithID(id)) }
}
func WithAdoptionStatus(s AdoptionStatus) AdoptionOption {
	return func(a *adoption) { a.status = s }
}
func WithAdoptionFeeCents(c int) AdoptionOption {
	return func(a *adoption) { a.feeCents = c }
}

// NewAdoption builds an adoption order; status defaults to PLACED.
func NewAdoption(owner, petID string, opts ...AdoptionOption) Adoption {
	a := &adoption{
		Resource: resource.New(resource.WithType(ResourceTypeAdoption)),
		owner:    owner,
		petID:    petID,
		status:   AdoptionStatusPlaced,
	}
	for _, opt := range opts {
		opt(a)
	}
	return a
}

func (a *adoption) Owner() string          { return a.owner }
func (a *adoption) PetID() string          { return a.petID }
func (a *adoption) Status() AdoptionStatus { return a.status }
func (a *adoption) FeeCents() int          { return a.feeCents }
