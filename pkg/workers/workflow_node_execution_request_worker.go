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
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type NodeExecutionRequestWorker struct {
	semaphore *semaphore.Weighted
	registry  *registry.Registry
}

func NewNodeExecutionRequestWorker(registry *registry.Registry) *NodeExecutionRequestWorker {
	return &NodeExecutionRequestWorker{
		registry:  registry,
		semaphore: semaphore.NewWeighted(25),
	}
}

func (w *NodeExecutionRequestWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			requests, err := models.ListNodeExecutionRequests()
			if err != nil {
				w.log("Error finding workflow nodes ready to be processed: %v", err)
			}

			for _, request := range requests {
				if err := w.semaphore.Acquire(context.Background(), 1); err != nil {
					w.log("Error acquiring semaphore: %v", err)
					continue
				}

				go func(request models.NodeExecutionRequest) {
					defer w.semaphore.Release(1)

					if err := w.LockAndProcessRequest(request); err != nil {
						w.log("Error processing request %s: %v", request.ID, err)
					}
				}(request)
			}
		}
	}
}

func (w *NodeExecutionRequestWorker) LockAndProcessRequest(request models.NodeExecutionRequest) error {
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		r, err := models.LockNodeExecutionRequest(tx, request.ID)
		if err != nil {
			w.log("Execution already being processed - skipping")
			return nil
		}

		return w.processRequest(tx, r)
	})
}

func (w *NodeExecutionRequestWorker) processRequest(tx *gorm.DB, request *models.NodeExecutionRequest) error {
	switch request.Type {
	case models.NodeExecutionRequestTypeInvokeAction:
		return w.invokeAction(tx, request)
	}

	return fmt.Errorf("unsupported node execution request type %s", request.Type)
}

func (w *NodeExecutionRequestWorker) invokeAction(tx *gorm.DB, request *models.NodeExecutionRequest) error {
	execution, err := models.FindNodeExecutionInTransaction(tx, request.WorkflowID, request.ExecutionID)
	if err != nil {
		return fmt.Errorf("execution %s not found: %w", request.ExecutionID, err)
	}

	workflow, err := models.FindUnscopedWorkflowInTransaction(tx, execution.WorkflowID)
	if err != nil {
		return fmt.Errorf("workflow %s not found: %w", execution.WorkflowID, err)
	}

	node, err := workflow.FindNode(execution.NodeID)
	if err != nil {
		return fmt.Errorf("node not found: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Data().Component.Name)
	if err != nil {
		return fmt.Errorf("component not found: %w", err)
	}

	spec := request.Spec.Data()
	if spec.InvokeAction == nil {
		return fmt.Errorf("spec is not specified")
	}

	actionName := spec.InvokeAction.ActionName
	actionDef := findAction(component, actionName)
	if actionDef == nil {
		return fmt.Errorf("action '%s' not found for component '%s'", actionName, node.Ref.Data().Component.Name)
	}

	actionCtx := components.ActionContext{
		Name:                  actionName,
		ActionParameters:      spec.InvokeAction.Parameters,
		MetadataContext:       contexts.NewMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(database.Conn(), execution),
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return fmt.Errorf("action execution failed: %w", err)
	}

	err = tx.Save(&execution).Error
	if err != nil {
		return fmt.Errorf("error saving execution after action handler: %v", err)
	}

	return request.Complete(tx)
}

func (w *NodeExecutionRequestWorker) log(format string, v ...any) {
	log.Printf("[NodeExecutionRequestWorker] "+format, v...)
}

func findAction(component components.Component, actionName string) *components.Action {
	for _, action := range component.Actions() {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
