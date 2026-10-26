package workers

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

// recoverHostedCreditNode revalidates the credit error under the caller's node
// lock. Run SuperPlane Setup validates configuration only; other component
// errors must still be repaired through the publish lifecycle.
func recoverHostedCreditNode(tx *gorm.DB, registry *registry.Registry, canvas *models.Canvas, node *models.CanvasNode) error {
	if !node.IsHostedCreditBlocked() {
		return nil
	}
	if registry == nil {
		return fmt.Errorf("registry is required to recover a hosted credit error")
	}

	action, err := registry.GetAction(models.SuperPlaneRunnerComponent)
	if err != nil {
		return err
	}
	config := node.Configuration.Data()
	if err := configuration.ValidateConfiguration(action.Configuration(), config); err != nil {
		return nil
	}
	validator, ok := action.(core.NodeConfigurationValidator)
	if !ok {
		return fmt.Errorf("hosted runner does not provide configuration validation")
	}
	if err := validator.ValidateNodeConfiguration(config); err != nil {
		return nil
	}
	if err := models.SuperPlaneRunnerReadinessError(tx, canvas.OrganizationID, canvas.FactoryID); err != nil {
		if models.IsSuperPlaneRunnerReadinessError(err) {
			return nil
		}
		return err
	}

	return node.MarkReady(tx)
}
