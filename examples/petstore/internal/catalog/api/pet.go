// Package api holds the catalog JSON:API DTOs.
package api

import (
	"time"

	"github.com/fromforgesoftware/go-kit/resource"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/catalog/domain"
)

const ResourceTypePet resource.Type = "pets"

type PetDTO struct {
	resource.RestDTO

	RName      string    `jsonapi:"attr,name,omitempty"`
	RSpecies   string    `jsonapi:"attr,species,omitempty"`
	RStatus    string    `jsonapi:"attr,status,omitempty"`
	RCreatedAt time.Time `jsonapi:"attr,createdAt,omitempty"`
	RUpdatedAt time.Time `jsonapi:"attr,updatedAt,omitempty"`
}

func PetToDTO(p domain.Pet) *PetDTO {
	if p == nil {
		return nil
	}
	dto := &PetDTO{
		RestDTO:    resource.ToRestDTO(p),
		RName:      p.Name(),
		RSpecies:   p.Species(),
		RStatus:    string(p.Status()),
		RCreatedAt: p.CreatedAt(),
		RUpdatedAt: p.UpdatedAt(),
	}
	dto.RType = ResourceTypePet
	return dto
}

func PetFromDTO(dto *PetDTO) domain.Pet {
	if dto == nil {
		return nil
	}
	opts := []domain.PetOption{}
	if dto.RStatus != "" {
		opts = append(opts, domain.WithPetStatus(domain.PetStatus(dto.RStatus)))
	}
	return domain.NewPet(dto.RName, dto.RSpecies, opts...)
}
