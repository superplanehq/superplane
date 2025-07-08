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
	Data          []byte
}

func (i *Integration) FindResource(resourceType string) (*IntegrationResource, error) {
	var resource IntegrationResource

	err := database.Conn().
		Where("integration_id = ?", i.ID).
		Where("type = ?", resourceType).
		First(&resource).
		Error

	if err != nil {
		return nil, err
	}

	return &resource, nil
}
