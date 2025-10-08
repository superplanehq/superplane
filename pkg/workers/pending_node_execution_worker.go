package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

type PendingNodeExecutionWorker struct {
	registry *registry.Registry
}

func NewPendingNodeExecutionWorker(registry *registry.Registry) *PendingNodeExecutionWorker {
	return &PendingNodeExecutionWorker{
		registry: registry,
	}
}

func (w *PendingNodeExecutionWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if err := w.processExecutions(); err != nil {
				w.log("Error processing node executions: %v", err)
			}
		}
	}
}

func (w *PendingNodeExecutionWorker) processExecutions() error {
	executions, err := models.FindPendingNodeExecutions()
	if err != nil {
		return err
	}

	for _, execution := range executions {
		if err := w.executeNode(&execution); err != nil {
			w.log("Error executing node %s: %v", execution.NodeID, err)
			if err := execution.Fail(err.Error()); err != nil {
				w.log("Error marking execution as failed: %v", err)
			}
		}
	}

	return nil
}

func (w *PendingNodeExecutionWorker) executeNode(execution *models.WorkflowNodeExecution) error {
	w.log("Executing node: workflow=%s, node=%s, event=%s", execution.WorkflowID, execution.NodeID, execution.EventID)

	node, err := w.findNode(execution)
	if err != nil {
		return err
	}

	if node.Ref.Blueprint != nil {
		w.log("Node %s is a blueprint node (%s)", execution.NodeID, node.Ref.Blueprint.Name)
		return w.executeBlueprintNode(execution, node)
	}

	w.log("Node %s is a component node (%s)", execution.NodeID, node.Ref.Component.Name)
	return w.executeComponentNode(execution, node)
}

func (w *PendingNodeExecutionWorker) findNode(execution *models.WorkflowNodeExecution) (*models.Node, error) {
	event, err := models.FindWorkflowEvent(execution.EventID.String())
	if err != nil {
		return nil, fmt.Errorf("workflow event %s not found: %w", execution.EventID, err)
	}

	//
	// If this event is for a blueprint, find the node in the blueprint
	//
	if event.BlueprintName != nil {
		w.log("Looking for node %s in blueprint '%s'", execution.NodeID, *event.BlueprintName)
		blueprint, err := models.FindBlueprintByName(*event.BlueprintName)
		if err != nil {
			return nil, fmt.Errorf("blueprint %s not found: %w", *event.BlueprintName, err)
		}

		return blueprint.FindNode(execution.NodeID)
	}

	//
	// Otherwise, find it in the workflow itself.
	//
	w.log("Looking for node %s in workflow %s", execution.NodeID, execution.WorkflowID)
	workflow, err := models.FindWorkflow(execution.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found: %w", execution.WorkflowID, err)
	}

	return workflow.FindNode(execution.NodeID)
}

func (w *PendingNodeExecutionWorker) executeBlueprintNode(execution *models.WorkflowNodeExecution, node *models.Node) error {
	_, err := models.FindBlueprintByName(node.Ref.Blueprint.Name)
	if err != nil {
		return fmt.Errorf("blueprint %s not found: %w", node.Ref.Blueprint.Name, err)
	}

	event, err := models.FindWorkflowEvent(execution.EventID.String())
	if err != nil {
		return fmt.Errorf("workflow event %s not found: %w", execution.EventID, err)
	}

	//
	// For blueprint executions,
	// we create a child workflow_events record with the blueprint name.
	//
	w.log("Creating child event for blueprint %s", node.Ref.Blueprint.Name)

	now := time.Now()
	blueprintName := node.Ref.Blueprint.Name
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		childEvent := models.WorkflowEvent{
			ID:            uuid.New(),
			WorkflowID:    execution.WorkflowID,
			ParentEventID: &event.ID,
			BlueprintName: &blueprintName,
			Data:          event.Data,
			State:         models.WorkflowEventStateRouting,
			CreatedAt:     &now,
			UpdatedAt:     &now,
		}

		if err := tx.Create(&childEvent).Error; err != nil {
			return err
		}

		w.log("Created child event %s for blueprint %s", childEvent.ID, blueprintName)
		return execution.StartInTransaction(tx)
	})
}

func (w *PendingNodeExecutionWorker) executeComponentNode(execution *models.WorkflowNodeExecution, node *models.Node) error {
	err := execution.Start()
	if err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", node.Ref.Component.Name, err)
	}

	event, err := models.FindWorkflowEvent(execution.EventID.String())
	if err != nil {
		return err
	}

	ctx := components.ExecutionContext{
		Configuration:         execution.Configuration.Data(),
		Data:                  execution.Inputs.Data(),
		MetadataContext:       contexts.NewMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(execution, event),
	}

	//
	// Execute component - it handles its own lifecycle
	//
	err = component.Execute(ctx)
	if err != nil {
		return execution.Fail(err.Error())
	}

	// Save any metadata changes
	return database.Conn().Save(execution).Error
}

func (w *PendingNodeExecutionWorker) log(format string, v ...any) {
	log.Printf("[PendingNodeExecutionWorker] "+format, v...)
}
