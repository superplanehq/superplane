package models

import (
	"time"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
)

type IntegrationResource struct {
	ID            uuid.UUID
	Name          string
	IntegrationID uuid.UUID
	Type          string
	CreatedAt     *time.Time
}

func FindIntegrationResourceByID(id uuid.UUID) (*IntegrationResource, error) {
	var resource IntegrationResource

	err := database.Conn().
		Where("id = ?", id).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}

func FindIntegrationResource(integrationID uuid.UUID, resourceType, name string) (*IntegrationResource, error) {
	var resource IntegrationResource

	err := database.Conn().
		Where("integration_id = ?", integrationID).
		Where("type = ?", resourceType).
		Where("name = ?", name).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
