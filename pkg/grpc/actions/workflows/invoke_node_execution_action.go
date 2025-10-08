package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
)

func InvokeNodeExecutionAction(ctx context.Context, registry *registry.Registry, executionID, actionName string, parameters map[string]any) (*pb.InvokeNodeExecutionActionResponse, error) {
	executionUUID, err := uuid.Parse(executionID)
	if err != nil {
		return nil, fmt.Errorf("invalid execution_id: %w", err)
	}

	var execution models.WorkflowNodeExecution
	err = database.Conn().
		Where("id = ?", executionUUID).
		First(&execution).
		Error

	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	// Find the workflow to get node information
	workflow, err := models.FindWorkflow(execution.WorkflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	node, err := workflow.FindNode(execution.NodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	//
	// TODO
	// Blueprint nodes don't expose actions for now.
	//
	if node.Ref.Component == nil {
		return nil, fmt.Errorf("node is not a component node")
	}

	component, err := registry.GetComponent(node.Ref.Component.Name)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	// Validate action exists and parameters
	var actionDef *components.Action
	for _, action := range component.Actions() {
		if action.Name == actionName {
			actionDef = &action
			break
		}
	}
	if actionDef == nil {
		return nil, fmt.Errorf("action '%s' not found for component '%s'", actionName, node.Ref.Component.Name)
	}

	// Validate action parameters
	if err := components.ValidateConfiguration(actionDef.Parameters, parameters); err != nil {
		return nil, fmt.Errorf("action parameter validation failed: %w", err)
	}

	event, err := models.FindWorkflowEvent(execution.EventID.String())
	if err != nil {
		return nil, fmt.Errorf("workflow event not found: %w", err)
	}

	// TODO: Get user ID from context
	actionCtx := components.ActionContext{
		Name:                  actionName,
		Parameters:            parameters,
		MetadataContext:       contexts.NewMetadataContext(&execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(&execution, event),
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return nil, fmt.Errorf("action execution failed: %w", err)
	}

	// Save any state/metadata changes
	err = database.Conn().Save(&execution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	return &pb.InvokeNodeExecutionActionResponse{}, nil
}
