package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/telemetry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var ErrRecordLocked = errors.New("record locked")

type WorkflowNodeExecutor struct {
	encryptor crypto.Encryptor
	registry  *registry.Registry
	baseURL   string
	semaphore *semaphore.Weighted
	logger    *logrus.Entry
}

func NewWorkflowNodeExecutor(encryptor crypto.Encryptor, registry *registry.Registry, baseURL string) *WorkflowNodeExecutor {
	return &WorkflowNodeExecutor{
		encryptor: encryptor,
		registry:  registry,
		baseURL:   baseURL,
		semaphore: semaphore.NewWeighted(25),
		logger:    logrus.WithFields(logrus.Fields{"worker": "WorkflowNodeExecutor"}),
	}
}

func (w *WorkflowNodeExecutor) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			tickStart := time.Now()

			executions, err := models.ListPendingNodeExecutions()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			telemetry.RecordExecutorWorkerNodesCount(context.Background(), len(executions))

			for _, execution := range executions {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				messages.NewWorkflowExecutionMessage(execution.WorkflowID.String(), execution.ID.String(), execution.NodeID).Publish()

				go func(execution models.WorkflowNodeExecution) {
					defer w.semaphore.Release(1)

					err := w.LockAndProcessNodeExecution(execution.ID)
					if err == nil {
						messages.NewWorkflowExecutionMessage(execution.WorkflowID.String(), execution.ID.String(), execution.NodeID).Publish()
						return
					}

					if err == ErrRecordLocked {
						return
					}

					w.logger.Errorf("Error processing node execution - node=%s, execution=%s: %v", execution.NodeID, execution.ID, err)
				}(execution)
			}

			telemetry.RecordExecutorWorkerTickDuration(context.Background(), time.Since(tickStart))
		}
	}
}

func (w *WorkflowNodeExecutor) LockAndProcessNodeExecution(id uuid.UUID) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		var execution models.WorkflowNodeExecution

		//
		// Try to lock the execution record for update.
		// If we can't, it means another worker is already processing it.
		//
		// We also ensure that the execution is still in pending state,
		// to avoid processing already started or finished executions.
		//
		// Why we need to check the state again:
		//
		// Even though we fetch pending executions in the main loop,
		// there is a race condition where multiple workers might pick the same execution
		// before any of them has a chance to lock it.
		//
		// By checking the state again here, we ensure that only one worker
		// can start processing a given execution.
		//
		// Note: We use SKIP LOCKED to avoid waiting on locked records.
		//

		err := tx.
			Clauses(clause.Locking{Strength: "UPDATE", Options: "SKIP LOCKED"}).
			Where("id = ?", id).
			Where("state = ?", models.WorkflowNodeExecutionStatePending).
			First(&execution).
			Error

		if err != nil {
			w.logger.Errorf("Execution %s already being processed - skipping", id.String())
			return ErrRecordLocked
		}

		return w.processNodeExecution(tx, &execution)
	})
}

func (w *WorkflowNodeExecutor) processNodeExecution(tx *gorm.DB, execution *models.WorkflowNodeExecution) error {
	node, err := models.FindWorkflowNode(tx, execution.WorkflowID, execution.NodeID)
	if err != nil {
		return err
	}

	if node.Type == models.NodeTypeBlueprint {
		return w.executeBlueprintNode(tx, execution, node)
	}

	return w.executeComponentNode(tx, execution, node)
}

func (w *WorkflowNodeExecutor) executeBlueprintNode(tx *gorm.DB, execution *models.WorkflowNodeExecution, node *models.WorkflowNode) error {
	ref := node.Ref.Data()
	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, ref.Blueprint.ID)
	if err != nil {
		return execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, "failed to find blueprint")
	}

	firstNode := blueprint.FindRootNode()
	if firstNode == nil {
		return fmt.Errorf("blueprint %s has no start node", blueprint.ID)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		return fmt.Errorf("error finding input: %v", err)
	}

	inputEvent, err := models.FindWorkflowEventInTransaction(tx, execution.EventID)
	if err != nil {
		return fmt.Errorf("error finding input event: %v", err)
	}

	//
	// Build the configuration for the first node.
	// If we have an error here, we should fail the execution,
	// since this means the first node has improper configuration,
	// and the user should be aware of this.
	//
	config, err := contexts.NewNodeConfigurationBuilder(tx, execution.WorkflowID).
		WithRootEvent(&execution.RootEventID).
		WithPreviousExecution(&execution.ID).
		ForBlueprintNode(node).
		WithInput(map[string]any{inputEvent.NodeID: input}).
		Build(firstNode.Configuration)

	if err != nil {
		err = execution.FailInTransaction(
			tx,
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("error building configuration for execution of node %s: %v", firstNode.ID, err),
		)

		return nil
	}

	_, err = models.CreatePendingChildExecution(tx, execution, firstNode.ID, config)
	if err != nil {
		return fmt.Errorf("failed to create child execution: %w", err)
	}

	err = execution.StartInTransaction(tx)

	return err
}

func (w *WorkflowNodeExecutor) executeComponentNode(tx *gorm.DB, execution *models.WorkflowNodeExecution, node *models.WorkflowNode) error {
	logger := logging.WithExecution(
		logging.WithNode(w.logger, *node),
		execution,
		nil,
	)

	err := execution.StartInTransaction(tx)
	if err != nil {
		logger.Errorf("failed to start execution: %v", err)
		return fmt.Errorf("failed to start execution: %w", err)
	}

	ref := node.Ref.Data()
	component, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		logger.Errorf("component %s not found: %v", ref.Component.Name, err)
		return fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		logger.Errorf("failed to get execution inputs: %v", err)
		return fmt.Errorf("failed to get execution inputs: %w", err)
	}

	workflow, err := models.FindWorkflowWithoutOrgScopeInTransaction(tx, node.WorkflowID)
	if err != nil {
		logger.Errorf("failed to find workflow: %v", err)
		return fmt.Errorf("failed to find workflow: %v", err)
	}

	ctx := core.ExecutionContext{
		ID:             execution.ID,
		WorkflowID:     execution.WorkflowID.String(),
		OrganizationID: workflow.OrganizationID.String(),
		NodeID:         execution.NodeID,
		BaseURL:        w.baseURL,
		Configuration:  execution.Configuration.Data(),
		Data:           input,
		HTTP:           contexts.NewHTTPContext(w.registry.GetHTTPClient()),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		NodeMetadata:   contexts.NewNodeMetadataContext(tx, node),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Auth:           contexts.NewAuthContext(tx, workflow.OrganizationID, nil, nil),
		Integration:    contexts.NewIntegrationContext(tx, w.registry),
		Notifications:  contexts.NewNotificationContext(tx, workflow.OrganizationID, execution.WorkflowID),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				logger.Errorf("app installation %s not found", *node.AppInstallationID)
				return execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, "app installation not found")
			}

			logger.Errorf("failed to find app installation: %v", err)
			return fmt.Errorf("failed to find app installation: %v", err)
		}

		logger = logging.WithAppInstallation(logger, *appInstallation)
		ctx.AppInstallation = contexts.NewAppInstallationContext(tx, node, appInstallation, w.encryptor, w.registry)
	}

	ctx.Logger = logger
	if err := component.Execute(ctx); err != nil {
		logger.Errorf("failed to execute component: %v", err)
		err = execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, err.Error())
		return err
	}

	logger.Info("Component executed successfully")

	return tx.Save(execution).Error
}
