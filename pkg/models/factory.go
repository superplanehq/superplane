package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const factoryNameUniqueConstraint = "factories_organization_id_name_key"

var ErrFactoryNameAlreadyExists = errors.New("factory name already exists")
var ErrFactoryNotFound = errors.New("factory not found")

type Factory struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Description    string
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (Factory) TableName() string {
	return "factories"
}

func MapFactoryNameUniqueConstraintError(err error) error {
	if err == nil {
		return nil
	}

	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.ConstraintName == factoryNameUniqueConstraint {
		return ErrFactoryNameAlreadyExists
	}

	return err
}

func CreateFactory(tx *gorm.DB, organizationID uuid.UUID, name, description string) (*Factory, error) {
	now := time.Now()
	factory := &Factory{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           name,
		Description:    description,
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(factory).Error; err != nil {
		return nil, MapFactoryNameUniqueConstraintError(err)
	}

	return factory, nil
}

func FindFactory(tx *gorm.DB, organizationID, factoryID uuid.UUID) (*Factory, error) {
	var factory Factory
	err := tx.
		Where("organization_id = ? AND id = ?", organizationID, factoryID).
		First(&factory).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactoryNotFound
		}
		return nil, err
	}

	return &factory, nil
}

func ListFactories(tx *gorm.DB, organizationID uuid.UUID) ([]Factory, error) {
	var factories []Factory
	err := tx.
		Where("organization_id = ?", organizationID).
		Order("name ASC").
		Order("id ASC").
		Find(&factories).
		Error
	if err != nil {
		return nil, err
	}

	return factories, nil
}
