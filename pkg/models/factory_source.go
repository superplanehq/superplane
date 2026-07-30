package models

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrFactorySourceNotFound = errors.New("factory source not found")

type FactorySource struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	FactoryID      uuid.UUID
	Name           string
	IntegrationID  uuid.UUID
	Configuration  datatypes.JSONType[map[string]any]
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

func (FactorySource) TableName() string {
	return "factory_sources"
}

func CreateFactorySource(
	tx *gorm.DB,
	organizationID, factoryID, integrationID uuid.UUID,
	name string,
	configuration map[string]any,
) (*FactorySource, error) {
	if configuration == nil {
		configuration = map[string]any{}
	}

	now := time.Now()
	source := &FactorySource{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		FactoryID:      factoryID,
		Name:           name,
		IntegrationID:  integrationID,
		Configuration:  datatypes.NewJSONType(configuration),
		CreatedAt:      now,
		UpdatedAt:      now,
	}

	if err := tx.Clauses(clause.Returning{}).Create(source).Error; err != nil {
		return nil, err
	}

	return source, nil
}

func FindFactorySource(tx *gorm.DB, organizationID, factoryID, sourceID uuid.UUID) (*FactorySource, error) {
	var source FactorySource
	err := tx.
		Where("organization_id = ? AND factory_id = ? AND id = ?", organizationID, factoryID, sourceID).
		First(&source).
		Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ErrFactorySourceNotFound
		}
		return nil, err
	}

	return &source, nil
}

func ListFactorySources(tx *gorm.DB, organizationID, factoryID uuid.UUID) ([]FactorySource, error) {
	var sources []FactorySource
	err := tx.
		Where("organization_id = ? AND factory_id = ?", organizationID, factoryID).
		Order("name ASC").
		Order("id ASC").
		Find(&sources).
		Error
	if err != nil {
		return nil, err
	}

	return sources, nil
}
