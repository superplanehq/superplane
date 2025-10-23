package contexts

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/integrations"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

type IntegrationContext struct {
	registry *registry.Registry
}

func NewIntegrationContext(registry *registry.Registry) triggers.IntegrationContext {
	return &IntegrationContext{
		registry: registry,
	}
}

func (c *IntegrationContext) GetIntegration(ID string) (integrations.ResourceManager, error) {
	integrationID, err := uuid.Parse(ID)
	if err != nil {
		return nil, err
	}

	integration, err := models.FindIntegrationByID(integrationID)
	if err != nil {
		return nil, err
	}

	return c.registry.NewResourceManager(context.Background(), integration)
}
