package workers

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"golang.org/x/sync/semaphore"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type BlueprintNodeExecutor struct {
	registry  *registry.Registry
	semaphore *semaphore.Weighted
}

func NewBlueprintNodeExecutor(registry *registry.Registry) *BlueprintNodeExecutor {
	return &BlueprintNodeExecutor{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *BlueprintNodeExecutor) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			executions, err := models.ListPendingChildExecutions()
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

					if err := w.LockAndProcessExecution(execution); err != nil {
						w.log("Error processing child execution %s: %v", execution.ID, err)
					}
				}(execution)
			}
		}
	}
}

func (w *BlueprintNodeExecutor) LockAndProcessExecution(execution models.WorkflowNodeExecution) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		e, err := models.LockWorkflowNodeExecution(tx, execution.ID)
		if err != nil {
			w.log("Execution already being processed - skipping")
			return nil
		}

		return w.processExecution(tx, e)
	})
}

// TODO: handle nested blueprints here?
func (w *BlueprintNodeExecutor) processExecution(tx *gorm.DB, execution *models.WorkflowNodeExecution) error {
	parent, err := models.FindNodeExecutionInTransaction(tx, execution.WorkflowID, *execution.ParentExecutionID)
	if err != nil {
		return fmt.Errorf("failed to find parent execution: %w", err)
	}

	parentNode, err := models.FindWorkflowNode(tx, parent.WorkflowID, parent.NodeID)
	if err != nil {
		return fmt.Errorf("failed to find workflow node: %w", err)
	}

	parentNodeRef := parentNode.Ref.Data()
	blueprint, err := models.FindUnscopedBlueprintInTransaction(tx, parentNodeRef.Blueprint.ID)
	if err != nil {
		return fmt.Errorf("failed to find blueprint: %w", err)
	}

	node, err := blueprint.FindNode(childNodeID(execution))
	if err != nil {
		return fmt.Errorf("blueprint node %s not found", execution.NodeID)
	}

	err = execution.StartInTransaction(tx)
	if err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", node.Ref.Component.Name, err)
	}

	input, err := execution.GetInput(tx)
	if err != nil {
		return fmt.Errorf("failed to get execution inputs: %w", err)
	}

	ctx := components.ExecutionContext{
		Configuration:         execution.Configuration.Data(),
		Data:                  input,
		MetadataContext:       contexts.NewMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(tx, execution),
		RequestContext:        contexts.NewExecutionRequestContext(execution),
	}

	if err := component.Execute(ctx); err != nil {
		return execution.FailInTransaction(tx, models.WorkflowNodeExecutionResultReasonError, err.Error())
	}

	return tx.Save(execution).Error
}

func childNodeID(execution *models.WorkflowNodeExecution) string {
	if execution.ParentExecutionID == nil {
		return ""
	}

	parts := strings.Split(execution.NodeID, ":")
	return parts[len(parts)-1]
}

func (w *BlueprintNodeExecutor) log(format string, v ...any) {
	log.Printf("[BlueprintNodeExecutor] "+format, v...)
}
