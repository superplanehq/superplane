package workers

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/google/uuid"
	"gorm.io/datatypes"
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
		// Check if there's already an execution running for this node
		// Only process if no other execution is in started/waiting state
		_, err := models.FindLastNodeExecutionForNode(
			execution.WorkflowID,
			execution.NodeID,
			[]string{
				models.WorkflowNodeExecutionStateWaiting,
				models.WorkflowNodeExecutionStateStarted,
			},
		)

		if err == nil {
			// An execution is already running for this node, skip
			w.log("Node %s already has a running execution, skipping", execution.NodeID)
			continue
		}

		if err := w.executeNode(&execution); err != nil {
			w.log("Error executing node %s: %v", execution.NodeID, err)
			if err := execution.Fail(models.WorkflowNodeExecutionResultReasonError, err.Error()); err != nil {
				w.log("Error marking execution as failed: %v", err)
			}
		}
	}

	return nil
}

func (w *PendingNodeExecutionWorker) executeNode(execution *models.WorkflowNodeExecution) error {
	w.log("Executing node: workflow=%s, node=%s, execution=%s", execution.WorkflowID, execution.NodeID, execution.ID)

	node, err := w.findNode(execution)
	if err != nil {
		return err
	}

	if node.Ref.Blueprint != nil {
		w.log("Node %s is a blueprint node (%s)", execution.NodeID, node.Ref.Blueprint.ID)
		return w.executeBlueprintNode(execution, node)
	}

	w.log("Node %s is a component node (%s)", execution.NodeID, node.Ref.Component.Name)
	return w.executeComponentNode(execution, node)
}

func (w *PendingNodeExecutionWorker) findNode(execution *models.WorkflowNodeExecution) (*models.Node, error) {
	// If this execution is inside a blueprint, find the node in the blueprint
	if execution.BlueprintID != nil {
		w.log("Looking for node %s in blueprint '%s'", execution.NodeID, *execution.BlueprintID)
		blueprint, err := models.FindBlueprintByID(execution.BlueprintID.String())
		if err != nil {
			return nil, fmt.Errorf("blueprint %s not found: %w", *execution.BlueprintID, err)
		}

		return blueprint.FindNode(execution.NodeID)
	}

	// Otherwise, find it in the workflow itself
	w.log("Looking for node %s in workflow %s", execution.NodeID, execution.WorkflowID)
	workflow, err := models.FindWorkflow(execution.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow %s not found: %w", execution.WorkflowID, err)
	}

	return workflow.FindNode(execution.NodeID)
}

func (w *PendingNodeExecutionWorker) executeBlueprintNode(execution *models.WorkflowNodeExecution, node *models.Node) error {
	blueprint, err := models.FindBlueprintByID(node.Ref.Blueprint.ID)
	if err != nil {
		return fmt.Errorf("blueprint %s not found: %w", node.Ref.Blueprint.ID, err)
	}

	w.log("Executing blueprint node %s (blueprint: %s)", execution.NodeID, node.Ref.Blueprint.ID)

	// Find first node in blueprint (node with no incoming edges)
	firstNode := w.findFirstNodeInBlueprint(blueprint)
	if firstNode == nil {
		return fmt.Errorf("blueprint %s has no start node", blueprint.ID)
	}

	blueprintID, err := uuid.Parse(node.Ref.Blueprint.ID)
	if err != nil {
		return fmt.Errorf("invalid blueprint ID: %w", err)
	}

	now := time.Now()
	return database.Conn().Transaction(func(tx *gorm.DB) error {
		// Create first execution inside blueprint
		// This execution inherits from the blueprint node execution
		firstExec := models.WorkflowNodeExecution{
			ID:                   uuid.New(),
			WorkflowID:           execution.WorkflowID,
			NodeID:               firstNode.ID,
			RootEventID:          execution.RootEventID,
			PreviousExecutionID:  &execution.ID,
			PreviousOutputBranch: nil, // Special: entering blueprint, will use blueprint node's inputs
			PreviousOutputIndex:  nil,
			ParentExecutionID:    &execution.ID,
			BlueprintID:          &blueprintID,
			State:                models.WorkflowNodeExecutionStatePending,
			Configuration:        datatypes.NewJSONType(firstNode.Configuration),
			CreatedAt:            &now,
			UpdatedAt:            &now,
		}

		if err := tx.Create(&firstExec).Error; err != nil {
			return fmt.Errorf("failed to create first blueprint execution: %w", err)
		}

		w.log("Created first execution %s in blueprint for node %s", firstExec.ID, firstNode.ID)

		// Move blueprint node to started state (waiting for children)
		return execution.StartInTransaction(tx)
	})
}

func (w *PendingNodeExecutionWorker) findFirstNodeInBlueprint(blueprint *models.Blueprint) *models.Node {
	hasIncoming := make(map[string]bool)
	for _, edge := range blueprint.Edges {
		if edge.TargetType == "node" {
			hasIncoming[edge.TargetID] = true
		}
	}

	for _, node := range blueprint.Nodes {
		if !hasIncoming[node.ID] {
			return &node
		}
	}

	return nil
}

func (w *PendingNodeExecutionWorker) executeComponentNode(execution *models.WorkflowNodeExecution, node *models.Node) error {
	if err := execution.Start(); err != nil {
		return fmt.Errorf("failed to start execution: %w", err)
	}

	component, err := w.registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return fmt.Errorf("component %s not found: %w", node.Ref.Component.Name, err)
	}

	inputs, err := execution.GetInputs()
	if err != nil {
		return fmt.Errorf("failed to get execution inputs: %w", err)
	}

	ctx := components.ExecutionContext{
		Configuration:         execution.Configuration.Data(),
		Data:                  inputs,
		MetadataContext:       contexts.NewMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(execution),
	}

	if err := component.Execute(ctx); err != nil {
		w.log("Component execution failed for %s (execution=%s): %v", node.Ref.Component.Name, execution.ID, err)
		return execution.Fail(models.WorkflowNodeExecutionResultReasonError, err.Error())
	}

	w.log("Component execution completed successfully for %s (execution=%s)", node.Ref.Component.Name, execution.ID)
	return database.Conn().Save(execution).Error
}

func (w *PendingNodeExecutionWorker) log(format string, v ...any) {
	log.Printf("[PendingNodeExecutionWorker] "+format, v...)
}
