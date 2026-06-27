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
	"github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
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
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "canvas not found")
	}

	execution, err := models.FindNodeExecution(canvas.ID, executionID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "execution not found")
	}

	node, err := canvas.FindNode(execution.NodeID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "node not found")
	}

	if node.Type != models.NodeTypeComponent || node.Ref.Data().Component == nil {
		return nil, grpcerrors.InvalidArgument(nil, "node is not a component node")
	}

	hookProvider, hookDef, err := registry.FindActionHook(node.Ref.Data().Component.Name, hookName)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "hook not found")
	}

	if hookDef.Type != core.HookTypeUser {
		return nil, grpcerrors.PermissionDenied(nil, fmt.Sprintf("hook '%s' cannot be invoked by user", hookName))
	}

	if err := configuration.ValidateConfiguration(hookDef.Parameters, parameters); err != nil {
		return nil, grpcerrors.InvalidArgument(err, "hook parameter validation failed")
	}

	user, err := models.FindActiveUserByID(orgID.String(), userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	tx := database.Conn()
	logger := logging.ForExecution(execution)
	actionCtx := core.ActionHookContext{
		Name:           hookName,
		Parameters:     parameters,
		Configuration:  node.Configuration.Data(),
		HTTP:           registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(tx, execution),
		ExecutionState: contexts.NewExecutionStateContext(tx, execution, onNewEvents),
		Auth:           contexts.NewAuthReader(tx, orgID, authService, user),
		Requests:       contexts.NewExecutionRequestContext(tx, execution),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, grpcerrors.Internal(err, "error building context")
		}

		logger = logging.WithIntegration(logger, *integration)
		actionCtx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, onNewEvents)
	}

	actionCtx.Logger = logger
	err = hookProvider.HandleHook(actionCtx)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "action execution failed")
	}

	if err := messages.PublishCanvasExecutionByID(execution.WorkflowID, execution.ID); err != nil {
		logger.Errorf("failed to publish execution state RabbitMQ message: %v", err)
	}

	for _, event := range newEvents {
		if err := messages.PublishCanvasEventCreatedMessage(&event); err != nil {
			logger.Errorf("failed to publish canvas event created RabbitMQ message: %v", err)
		}
	}

	return &pb.InvokeNodeExecutionHookResponse{}, nil
}
