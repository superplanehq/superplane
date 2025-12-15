package contexts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

type IntegrationContext struct {
	tx       *gorm.DB
	registry *registry.Registry
}

func NewIntegrationContext(tx *gorm.DB, registry *registry.Registry) *IntegrationContext {
	return &IntegrationContext{
		tx:       tx,
		registry: registry,
	}
}

func (c *IntegrationContext) GetIntegration(ID string) (integrations.ResourceManager, error) {
	integrationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, err
	}

	integration, err := models.FindIntegrationByIDInTransaction(c.tx, integrationID)
	if err != nil {
		return nil, err
	}

	return c.registry.NewResourceManagerInTransaction(context.Background(), c.tx, integration)
}
