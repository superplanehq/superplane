package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/primitives"
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

	// Blueprint nodes don't have actions for now
	if node.Ref.Primitive == nil {
		return nil, fmt.Errorf("node is not a primitive node")
	}

	primitive, err := registry.GetPrimitive(node.Ref.Primitive.Name)
	if err != nil {
		return nil, fmt.Errorf("primitive not found: %w", err)
	}

	event, err := models.FindWorkflowEvent(execution.EventID.String())
	if err != nil {
		return nil, fmt.Errorf("workflow event not found: %w", err)
	}

	// TODO: Get user ID from context
	actionCtx := primitives.ActionContext{
		Name:       actionName,
		Parameters: parameters,
		Metadata:   contexts.NewMetadataContext(&execution),
		State:      contexts.NewExecutionStateContext(&execution, event),
	}

	err = primitive.HandleAction(actionCtx)
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
