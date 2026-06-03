// Package api holds the adoptions JSON:API DTOs.
package api

import (
	"time"

	"github.com/fromforgesoftware/go-kit/resource"

	"github.com/fromforgesoftware/forge-examples/examples/petstore/internal/adoptions/domain"
)

const ResourceTypeAdoption resource.Type = "adoptions"

type AdoptionDTO struct {
	resource.RestDTO

	ROwner     string    `jsonapi:"attr,owner,omitempty"`
	RPetID     string    `jsonapi:"attr,petId,omitempty"`
	RStatus    string    `jsonapi:"attr,status,omitempty"`
	RFeeCents  int       `jsonapi:"attr,feeCents,omitempty"`
	RCreatedAt time.Time `jsonapi:"attr,createdAt,omitempty"`
	RUpdatedAt time.Time `jsonapi:"attr,updatedAt,omitempty"`
}

func AdoptionToDTO(a domain.Adoption) *AdoptionDTO {
	if a == nil {
		return nil
	}
	dto := &AdoptionDTO{
		RestDTO:    resource.ToRestDTO(a),
		ROwner:     a.Owner(),
		RPetID:     a.PetID(),
		RStatus:    string(a.Status()),
		RFeeCents:  a.FeeCents(),
		RCreatedAt: a.CreatedAt(),
		RUpdatedAt: a.UpdatedAt(),
	}
	dto.RType = ResourceTypeAdoption
	return dto
}
