package workers

import (
	"context"
	"errors"
	"reflect"
	"slices"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/billing"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

var errExecutionPreparationChanged = errors.New("execution changed during preparation; retry with current inputs")

type actionExecutionPreparation struct {
	execution *models.CanvasNodeExecution
	component string
	canvas    *models.Canvas
	dispatch  core.PreparedExecution
	err       error
}

func (p *actionExecutionPreparation) matches(execution *models.CanvasNodeExecution, node *models.CanvasNode, canvas *models.Canvas) bool {
	return p.component == executionComponentName(execution, node) &&
		p.execution.WorkflowID == execution.WorkflowID && p.execution.NodeID == execution.NodeID &&
		p.execution.RunID == execution.RunID && p.execution.EventID == execution.EventID &&
		p.execution.RootEventID == execution.RootEventID &&
		p.canvas.OrganizationID == canvas.OrganizationID &&
		reflect.DeepEqual(p.canvas.FactoryID, canvas.FactoryID) &&
		reflect.DeepEqual(p.execution.Configuration.Data(), execution.Configuration.Data())
}

// prepareActionExecution takes a short snapshot under the execution lock, then
// releases the transaction before resolving credentials, files, and resources.
// The final transaction locks and checks the execution again before dispatch.
func (w *NodeExecutor) prepareActionExecution(id uuid.UUID) (*actionExecutionPreparation, error) {
	db := database.DB(context.Background())
	var preparation *actionExecutionPreparation
	var preparer core.ExecutionPreparer
	var node *models.CanvasNode
	err := db.Transaction(func(tx *gorm.DB) error {
		execution, err := models.LockPendingNodeExecutionInActiveCanvas(tx, id)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrRecordLocked
			}
			return err
		}
		node, err = models.FindCanvasNode(tx, execution.WorkflowID, execution.NodeID)
		if err != nil {
			return err
		}
		component := executionComponentName(execution, node)
		action, err := w.registry.GetAction(component)
		if err != nil {
			return err
		}
		preparer = executionPreparer(action)
		if preparer == nil {
			return nil
		}
		canvas, err := models.FindCanvasWithoutOrgScopeInTransaction(tx, execution.WorkflowID)
		if err != nil {
			return err
		}
		preparation = &actionExecutionPreparation{execution: execution, component: component, canvas: canvas}
		if err := assertExecutionActionAccess(tx, canvas, execution, component); err != nil {
			if !isExecutionAccessDenied(err) {
				return err
			}
			// The final transaction records confirmed access denials. If access
			// changes meanwhile, leave the execution pending for fresh preparation.
			preparer = nil
		}
		return nil
	})
	if err != nil || preparer == nil {
		return preparation, err
	}

	execution, canvas := preparation.execution, preparation.canvas
	logger := logging.WithCanvasWorkspace(logging.WithExecution(logging.WithNode(w.logger, *node), execution), canvas)
	preparation.dispatch, preparation.err = preparer.Prepare(core.ExecutionContext{
		ID: execution.ID, RunID: execution.RunID, WorkflowID: execution.WorkflowID.String(),
		OrganizationID: canvas.OrganizationID.String(), CanvasName: canvas.Name,
		NodeID: execution.NodeID, NodeName: node.Name, BaseURL: w.baseURL,
		Configuration: execution.Configuration.Data(), Logger: logger,
		HTTP:    w.registry.HTTPContext(),
		Secrets: contexts.NewSecretsContext(db, w.registry, canvas.OrganizationID, w.encryptor),
	})
	return preparation, nil
}

func executionPreparer(action core.Action) core.ExecutionPreparer {
	if provider, ok := action.(interface{ ExecutionPreparer() core.ExecutionPreparer }); ok {
		return provider.ExecutionPreparer()
	}
	preparer, _ := action.(core.ExecutionPreparer)
	return preparer
}

func executionComponentName(execution *models.CanvasNodeExecution, node *models.CanvasNode) string {
	if name := execution.FrozenComponentName(); name != "" {
		return name
	}
	if ref := node.Ref.Data(); ref.Component != nil {
		return ref.Component.Name
	}
	return ""
}

func assertExecutionActionAccess(tx *gorm.DB, canvas *models.Canvas, execution *models.CanvasNodeExecution, component string) error {
	if err := billing.AssertOrganizationPaymentMethod(tx, canvas.OrganizationID); err != nil {
		return err
	}
	if models.IsAgentRunnerComponent(component) {
		if err := billing.AssertOrganizationPlanAllowsRuns(tx, canvas.OrganizationID); err != nil {
			return err
		}
	}
	if slices.Contains([]string{"runnerClaudeCode", "runnerCodex", "runnerOpenRouter"}, component) {
		configuration := execution.Configuration.Data()
		credentials, _ := configuration["credentials"].(map[string]any)
		source, _ := credentials["source"].(string)
		if source == "integration" || source == "secret" {
			provider, _ := configuration["llmProvider"].(string)
			return models.RequireOrganizationBYOKAccess(tx, canvas.OrganizationID, provider)
		}
	}
	return nil
}

func isExecutionAccessDenied(err error) bool {
	return errors.Is(err, billing.ErrPaymentMethodRequired) ||
		errors.Is(err, billing.ErrHostedSubscriptionRequired) ||
		errors.Is(err, models.ErrBYOKAccessDenied)
}
