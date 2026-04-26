package canvases

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
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InvokeNodeExecutionHook(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	executionID uuid.UUID,
	hookName string,
	parameters map[string]any,
) (*pb.InvokeNodeExecutionHookResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return nil, fmt.Errorf("canvas not found: %w", err)
	}

	execution, err := models.FindNodeExecution(canvas.ID, executionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	node, err := canvas.FindNode(execution.NodeID)
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

	hookProvider, hookDef, err := registry.FindActionHook(node.Ref.Data().Component.Name, hookName)
	if err != nil {
		return nil, fmt.Errorf("hook not found: %w", err)
	}

	if hookDef.Type != core.HookTypeUser {
		return nil, fmt.Errorf("hook '%s' cannot be invoked", hookName)
	}

	if err := configuration.ValidateConfiguration(hookDef.Parameters, parameters); err != nil {
		return nil, fmt.Errorf("hook parameters validation failed: %w", err)
	}

	user, err := models.FindActiveUserByID(orgID.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	tx := database.Conn()
	logger := logging.ForExecution(execution, nil)
	actionCtx := core.ActionHookContext{
		Name:           hookName,
		Parameters:     parameters,
		Configuration:  node.Configuration.Data(),
		HTTP:           registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Auth:           contexts.NewAuthReader(tx, orgID, authService, user),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
		Notifications:  contexts.NewNotificationContext(tx, orgID, canvas.ID),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, status.Error(codes.Internal, "error building context")
		}

		logger = logging.WithIntegration(logger, *integration)
		actionCtx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, onNewEvents)
	}

	actionCtx.Logger = logger
	err = hookProvider.HandleHook(actionCtx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action execution failed: %v", err)
	}

	messages.NewCanvasExecutionMessage(
		execution.WorkflowID.String(),
		execution.ID.String(),
		execution.NodeID,
	).Publish()

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	return &pb.InvokeNodeExecutionHookResponse{}, nil
}
