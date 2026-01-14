package workflows

import (
	"context"
	"fmt"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	"github.com/superplanehq/superplane/pkg/logging"
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
	encryptor crypto.Encryptor,
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

	tx := database.Conn()
	logger := logging.ForExecution(execution, nil)
	actionCtx := core.ActionContext{
		Name:           actionName,
		Parameters:     parameters,
		Configuration:  node.Configuration.Data(),
		HTTP:           contexts.NewHTTPContext(registry.GetHTTPClient()),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution),
		Auth:           contexts.NewAuthContext(tx, orgID, authService, user),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Integration:    contexts.NewIntegrationContext(tx, registry),
		Notifications:  contexts.NewNotificationContext(tx, orgID, workflow.ID),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, status.Error(codes.Internal, "error building context")
		}

		logger = logging.WithAppInstallation(logger, *appInstallation)
		actionCtx.AppInstallation = contexts.NewAppInstallationContext(tx, node, appInstallation, encryptor, registry)
	}

	actionCtx.Logger = logger
	err = component.HandleAction(actionCtx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action execution failed: %v", err)
	}

	messages.NewWorkflowExecutionMessage(
		execution.WorkflowID.String(),
		execution.ID.String(),
		execution.NodeID,
	).Publish()

	return &pb.InvokeNodeExecutionActionResponse{}, nil
}

func findAction(component core.Component, actionName string) *core.Action {
	for _, action := range component.Actions() {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
