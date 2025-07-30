package models

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"gorm.io/datatypes"
)

type StageExecutor struct {
	ID         uuid.UUID `gorm:"primary_key;default:uuid_generate_v4()"`
	StageID    uuid.UUID
	ResourceID *uuid.UUID
	Type       string
	Spec       datatypes.JSON
}

func (e *StageExecutor) GetResource() (*Resource, error) {
	var resource Resource

	err := database.Conn().
		Where("id = ?", e.ResourceID).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func (e *StageExecutor) GetIntegrationResource() (*IntegrationResource, error) {
	var r IntegrationResource

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Select("resources.name as name, resources.type as type, integrations.name as integration_name, integrations.domain_type as domain_type").
		Where("resources.id = ?", e.ResourceID).
		First(&r).
		Error

	if err != nil {
		return nil, err
	}

	return &r, nil
}

func (e *StageExecutor) FindIntegration() (*Integration, error) {
	var integration Integration

	err := database.Conn().
		Table("resources").
		Joins("INNER JOIN integrations ON integrations.id = resources.integration_id").
		Where("resources.id = ?", e.ResourceID).
		Select("integrations.*").
		First(&integration).
		Error

	if err != nil {
		return nil, err
	}

	return &integration, nil
}
