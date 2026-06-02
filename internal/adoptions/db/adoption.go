// Package db holds the adoptions Postgres repository.
package db

import (
	"context"
	"errors"
	"time"

	apierrors "github.com/fromforgesoftware/go-kit/errors"
	"github.com/fromforgesoftware/go-kit/persistence/gormdb"
	"github.com/fromforgesoftware/go-kit/persistence/postgres"
	"github.com/fromforgesoftware/go-kit/resource"
	"github.com/fromforgesoftware/go-kit/search"
	"github.com/fromforgesoftware/go-kit/slicesx"
	"gorm.io/gorm"

	"github.com/fromforgesoftware/forge-examples/internal/adoptions/domain"
	"github.com/fromforgesoftware/forge-examples/internal/adoptions/fields"
)

var adoptionFieldMapping = map[string]string{
	fields.ID:        "id",
	fields.CreatedAt: "created_at",
	fields.UpdatedAt: "updated_at",
	fields.Owner:     "owner",
	fields.PetID:     "pet_id",
	fields.Status:    "status",
}

type adoptionEntity struct {
	EID        string    `gorm:"column:id;type:uuid;default:uuid_generate_v4();primaryKey"`
	ECreatedAt time.Time `gorm:"column:created_at;type:timestamptz;default:now()"`
	EUpdatedAt time.Time `gorm:"column:updated_at;type:timestamptz;default:now()"`
	EOwner     string    `gorm:"column:owner"`
	EPetID     string    `gorm:"column:pet_id;type:uuid"`
	EStatus    string    `gorm:"column:status"`
	EFeeCents  int       `gorm:"column:fee_cents"`
}

func (e *adoptionEntity) TableName() string     { return "adoptions.adoption" }
func (e *adoptionEntity) Type() resource.Type   { return domain.ResourceTypeAdoption }
func (e *adoptionEntity) ID() string            { return e.EID }
func (e *adoptionEntity) LID() string           { return "" }
func (e *adoptionEntity) CreatedAt() time.Time  { return e.ECreatedAt }
func (e *adoptionEntity) UpdatedAt() time.Time  { return e.EUpdatedAt }
func (e *adoptionEntity) DeletedAt() *time.Time { return nil }

func (e *adoptionEntity) Owner() string                 { return e.EOwner }
func (e *adoptionEntity) PetID() string                 { return e.EPetID }
func (e *adoptionEntity) Status() domain.AdoptionStatus { return domain.AdoptionStatus(e.EStatus) }
func (e *adoptionEntity) FeeCents() int                 { return e.EFeeCents }

func adoptionToEntity(a domain.Adoption) *adoptionEntity {
	return &adoptionEntity{
		EID:       a.ID(),
		EOwner:    a.Owner(),
		EPetID:    a.PetID(),
		EStatus:   string(a.Status()),
		EFeeCents: a.FeeCents(),
	}
}

type adoptionRepo struct {
	*postgres.Repo
}

func NewAdoptionRepository(db *gormdb.DBClient) (*adoptionRepo, error) {
	r, err := postgres.NewRepo(db, adoptionFieldMapping)
	if err != nil {
		return nil, err
	}
	return &adoptionRepo{Repo: r}, nil
}

func (r *adoptionRepo) Create(ctx context.Context, a domain.Adoption) (domain.Adoption, error) {
	e := adoptionToEntity(a)
	tx := r.DB.WithContext(ctx)
	if e.EID == "" {
		tx = tx.Omit("id")
	}
	if e.ECreatedAt.IsZero() {
		tx = tx.Omit("created_at", "updated_at")
	}
	if err := tx.Create(e).Error; err != nil {
		return nil, postgres.NewErrUnknown(err)
	}
	return e, nil
}

func (r *adoptionRepo) Get(ctx context.Context, opts ...search.Option) (domain.Adoption, error) {
	var e adoptionEntity
	if err := r.QueryApply(ctx, search.New(opts...).Query()).First(&e).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, apierrors.NotFound("adoption", "")
		}
		return nil, postgres.NewErrUnknown(err)
	}
	return &e, nil
}

func (r *adoptionRepo) List(ctx context.Context, opts ...search.Option) (resource.ListResponse[domain.Adoption], error) {
	q := search.New(opts...).Query()
	var found []*adoptionEntity
	if err := r.QueryApply(ctx, q).Find(&found).Error; err != nil {
		return nil, postgres.NewErrUnknown(err)
	}
	var total int64
	if err := r.CountApply(ctx, new(adoptionEntity), q).Count(&total).Error; err != nil {
		return nil, postgres.NewErrUnknown(err)
	}
	out := slicesx.Map(found, func(e *adoptionEntity) domain.Adoption { return e })
	return resource.NewListResponse(out, int(total)), nil
}
