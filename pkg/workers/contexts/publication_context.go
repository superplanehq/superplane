package contexts

import (
	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"gorm.io/gorm"
)

type PublicationContext struct {
	tx             *gorm.DB
	organizationID uuid.UUID
	integrationID  uuid.UUID
	execution      *models.CanvasNodeExecution
}

func NewPublicationContext(tx *gorm.DB, organizationID uuid.UUID, node *models.CanvasNode, execution *models.CanvasNodeExecution) *PublicationContext {
	c := &PublicationContext{tx: tx, organizationID: organizationID, execution: execution}
	if node != nil && node.AppInstallationID != nil {
		c.integrationID = *node.AppInstallationID
	}
	return c
}

func (c *IntegrationContext) Publications() core.PublicationStore {
	return &PublicationContext{tx: c.tx, organizationID: c.integration.OrganizationID, integrationID: c.integration.ID}
}

func (c *PublicationContext) ResolveOrder(source, attempt uuid.UUID) (core.PublicationOrder, error) {
	return models.ResolvePublicationOrder(c.tx, c.organizationID, c.integrationID, c.execution, source, attempt)
}
func (c *PublicationContext) ExecutionOutput(id uuid.UUID) (core.PublicationExecution, error) {
	return models.FindPublicationExecutionOutput(c.tx, c.integrationID, c.execution, id)
}
func (c *PublicationContext) Get(key string) (*core.OrderedPublication, error) {
	return models.FindOrderedPublication(c.tx, c.integrationID, key)
}
func (c *PublicationContext) List(prefix string) ([]core.OrderedPublication, error) {
	return models.ListOrderedPublications(c.tx, c.integrationID, prefix)
}
func (c *PublicationContext) Update(key string, apply func(*core.OrderedPublication) error) error {
	return models.UpdateOrderedPublication(c.tx, c.integrationID, key, apply)
}
func (c *PublicationContext) AttemptFinished(id uuid.UUID) (bool, error) {
	return models.PublicationAttemptFinished(c.tx, c.organizationID, id)
}

func (c *PublicationContext) NodeSnapshot(canvasID uuid.UUID, nodeID string, attempt uuid.UUID) (*core.PublicationNode, error) {
	return models.FindPublicationNode(c.tx, c.organizationID, c.integrationID, canvasID, nodeID, attempt)
}
