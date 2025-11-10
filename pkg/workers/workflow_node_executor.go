package workers

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

var ErrRecordLocked = errors.New("record locked")

type WorkflowNodeExecutor struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
	logger    *logrus.Entry
}

func NewWorkflowNodeExecutor(registry *registry.Registry) *WorkflowNodeExecutor {
	return &WorkflowNodeExecutor{
		registry:  registry,
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
			executions, err := models.ListPendingNodeExecutions()
			if err != nil {
				w.logger.Errorf("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, execution := range executions {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.logger.Errorf("Error acquiring semaphore: %v", err)
					continue
				}

				go func(execution models.WorkflowNodeExecution) {
					defer w.semaphore.Release(1)

					err := w.LockAndProcessNodeExecution(execution)
					if err == nil {
						return
					}

					if err == ErrRecordLocked {
						return
					}

					w.logger.Errorf("Error processing node execution - node=%s, execution=%s: %v", execution.NodeID, execution.ID, err)
				}(execution)
			}
		}
	}
}

func (w *WorkflowNodeExecutor) LockAndProcessNodeExecution(execution models.WorkflowNodeExecution) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		e, err := models.LockWorkflowNodeExecution(tx, execution.ID)
		if err != nil {
			w.logger.Errorf("Execution %s already being processed - skipping", execution.ID)
			return ErrRecordLocked
		}

		return w.processNodeExecution(tx, e)
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
		WithInput(input).
		Build(firstNode.Configuration)

	if err != nil {
		err = execution.FailInTransaction(
			tx,
			models.WorkflowNodeExecutionResultReasonError,
			fmt.Sprintf("error building configuration for execution of node %s: %v", firstNode.ID, err),
		)

		messages.NewWorkflowExecutionFinishedMessage(execution.WorkflowID.String(), execution).
			PublishWithDelay(1 * time.Second)

		return nil
	}

	createdChildExecution, err := models.CreatePendingChildExecution(tx, execution, firstNode.ID, config)
	if err != nil {
		return fmt.Errorf("failed to create child execution: %w", err)
	}

	messages.NewWorkflowExecutionCreatedMessage(createdChildExecution.WorkflowID.String(), createdChildExecution).PublishWithDelay(1 * time.Second)

	err = execution.StartInTransaction(tx)

	if err == nil {
		messages.NewWorkflowExecutionStartedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)
	}

	return err
}

func (w *WorkflowNodeExecutor) executeComponentNode(tx *gorm.DB, execution *models.WorkflowNodeExecution, node *models.WorkflowNode) error {
	logger := logging.ForExecution(
		logging.ForNode(w.logger, *node),
		execution,
		nil,
	)

	err := execution.StartInTransaction(tx)
	if err != nil {
		logger.Errorf("failed to start execution: %v", err)
		return fmt.Errorf("failed to start execution: %w", err)
	}

	messages.NewWorkflowExecutionStartedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)

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

	workflow, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
	if err != nil {
		logger.Errorf("failed to find workflow: %v", err)
		return fmt.Errorf("failed to find workflow: %v", err)
	}

	ctx := components.ExecutionContext{
		ID:                    execution.ID.String(),
		WorkflowID:            execution.WorkflowID.String(),
		Configuration:         execution.Configuration.Data(),
		Data:                  input,
		MetadataContext:       contexts.NewExecutionMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
		RequestContext:        contexts.NewExecutionRequestContext(tx, execution),
		AuthContext:           contexts.NewAuthContext(tx, workflow.OrganizationID, nil, nil),
		IntegrationContext:    contexts.NewIntegrationContext(tx, w.registry),
	}

	if err := component.Execute(ctx); err != nil {
		logger.Errorf("failed to execute component: %v", err)
		err = execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, err.Error())
		messages.NewWorkflowExecutionFinishedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)
		return err
	}

	messages.NewWorkflowExecutionFinishedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)

	logger.Info("Component executed successfully")

	return tx.Save(execution).Error
}
