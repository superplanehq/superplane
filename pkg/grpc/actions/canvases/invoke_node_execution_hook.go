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
	"github.com/superplanehq/superplane/pkg/grpc/actions/messages"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"gorm.io/gorm"
)

func InvokeNodeExecutionHook(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	db *gorm.DB,
	canvas *models.Canvas,
	executionID uuid.UUID,
	hookName string,
	parameters map[string]any,
) (*pb.InvokeNodeExecutionHookResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	execution, err := models.FindNodeExecution(canvas.ID, executionID)
	if err != nil {
		return nil, fmt.Errorf("execution not found: %w", err)
	}

	node, err := canvas.FindNode(execution.NodeID)
	if err != nil {
		return nil, fmt.Errorf("node not found: %w", err)
	}

	if node.Type != models.NodeTypeComponent || node.Ref.Data().Component == nil {
		return nil, grpcerrors.InvalidArgument(nil, "node is not a component node")
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

	user, err := models.FindActiveUserByIDInTransaction(db, canvas.OrganizationID.String(), userID)
	if err != nil {
		return nil, fmt.Errorf("user not found: %w", err)
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	logger := logging.ForExecution(execution)
	actionCtx := core.ActionHookContext{
		Name:           hookName,
		Parameters:     parameters,
		Configuration:  node.Configuration.Data(),
		HTTP:           registry.HTTPContext(),
		Metadata:       contexts.NewExecutionMetadataContext(db, execution),
		ExecutionState: contexts.NewExecutionStateContext(db, execution, onNewEvents),
		Auth:           contexts.NewAuthReader(db, canvas.OrganizationID, authService, user),
		Requests:       contexts.NewExecutionRequestContext(db, execution),
		Runs:           contexts.NewRunExecutionContext(db, canvas, node, execution),
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(db, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, grpcerrors.Internal(err, "error building context")
		}

		logger = logging.WithIntegration(logger, *integration)
		actionCtx.Integration = contexts.NewIntegrationContext(db, node, integration, encryptor, registry, onNewEvents)
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
