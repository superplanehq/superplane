package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InvokeNodeExecutionAction(
	ctx context.Context,
	authService authorization.Authorization,
	registry *registry.Registry,
	orgID uuid.UUID,
	workflowID uuid.UUID,
	executionID uuid.UUID,
	actionName string,
	parameters map[string]any,
) (*pb.InvokeNodeExecutionActionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	workflow, err := models.FindWorkflow(orgID, workflowID)
	if err != nil {
		return nil, fmt.Errorf("workflow not found: %w", err)
	}

	execution, err := models.FindNodeExecution(workflow.ID, executionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	node, err := workflow.FindNode(execution.NodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	//
	// TODO
	// Blueprint nodes don't expose actions for now.
	//
	if node.Ref.Data().Component == nil {
		return nil, fmt.Errorf("node is not a component node")
	}

	component, err := registry.GetComponent(node.Ref.Data().Component.Name)
	if err != nil {
		return nil, fmt.Errorf("component not found: %w", err)
	}

	actionDef := findAction(component, actionName)
	if actionDef == nil {
		return nil, fmt.Errorf("action '%s' not found for component '%s'", actionName, node.Ref.Data().Component.Name)
	}

	if err := configuration.ValidateConfiguration(actionDef.Parameters, parameters); err != nil {
		return nil, fmt.Errorf("action parameter validation failed: %w", err)
	}

	user, err := models.FindActiveUserByID(orgID.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	actionCtx := components.ActionContext{
		Name:                  actionName,
		Parameters:            parameters,
		Configuration:         node.Configuration.Data(),
		MetadataContext:       contexts.NewExecutionMetadataContext(execution),
		ExecutionStateContext: contexts.NewExecutionStateContext(database.Conn(), execution),
		AuthContext:           contexts.NewAuthContext(orgID, authService, user),
		RequestContext:        contexts.NewExecutionRequestContext(database.Conn(), execution),
		IntegrationContext:    contexts.NewIntegrationContext(registry),
	}

	err = component.HandleAction(actionCtx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action execution failed: %v", err)
	}

	err = database.Conn().Save(&execution).Error
	if err != nil {
		return nil, fmt.Errorf("failed to save execution: %w", err)
	}

	return &pb.InvokeNodeExecutionActionResponse{}, nil
}

func findAction(component components.Component, actionName string) *components.Action {
	for _, action := range component.Actions() {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
