package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type WorkflowNodeExecutor struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
}

func NewWorkflowNodeExecutor(registry *registry.Registry) *WorkflowNodeExecutor {
	return &WorkflowNodeExecutor{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
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
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, execution := range executions {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(execution models.WorkflowNodeExecution) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessNodeExecution(execution); err != nil {
						w.log("Error processing execution %s - workflow=%s, node=%s: %v", execution.ID, execution.WorkflowID, execution.NodeID, err)
					}
				}(execution)
			}
		}
	}
}

func (w *WorkflowNodeExecutor) LockAndProcessNodeExecution(execution models.WorkflowNodeExecution) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		e, err := models.LockWorkflowNodeExecution(tx, execution.ID)
		if err != nil {
			w.log("Node already being processed - skipping")
			return nil
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

	config, err := contexts.NewNodeConfigurationBuilder(tx, execution.WorkflowID).
		WithRootEvent(&execution.RootEventID).
		WithPreviousExecution(&execution.ID).
		ForBlueprintNode(node).
		WithInput(input).
		Build(firstNode.Configuration)

	if err != nil {
		return err
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
	err := execution.StartInTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	messages.NewWorkflowExecutionStartedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)

	ref := node.Ref.Data()
	component, err := w.registry.GetComponent(ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", ref.Component.Name, err)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		return fmt.Errorf("failed to get execution inputs: %w", err)
	}

	workflow, err := models.FindUnscopedWorkflowInTransaction(tx, node.WorkflowID)
	if err != nil {
		return fmt.Errorf("failed to find workflow: %v", err)
	}

	ctx := components.ExecutionContext{
		Configuration:         execution.Configuration.Data(),
		Data:                  input,
		MetadataContext:       contexts.NewExecutionMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
		RequestContext:        contexts.NewExecutionRequestContext(tx, execution),
		AuthContext:           contexts.NewAuthContext(workflow.OrganizationID, nil),
	}

	if err := component.Execute(ctx); err != nil {
		err = execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, err.Error())
		messages.NewWorkflowExecutionFinishedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)
		return err
	}

	messages.NewWorkflowExecutionFinishedMessage(execution.WorkflowID.String(), execution).PublishWithDelay(1 * time.Second)

	w.log("Execute() returned for execution=%s, node=%s", execution.ID, node.NodeID)
	return tx.Save(execution).Error
}

func (w *WorkflowNodeExecutor) log(format string, v ...any) {
	log.Printf("[WorkflowNodeExecutor] "+format, v...)
}
