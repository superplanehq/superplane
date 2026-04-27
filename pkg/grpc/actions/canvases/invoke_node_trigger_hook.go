package canvases

import (
	"context"

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
	"google.golang.org/protobuf/types/known/structpb"
)

func InvokeNodeTriggerHook(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	orgID uuid.UUID,
	canvasID uuid.UUID,
	nodeID string,
	hookName string,
	parameters map[string]any,
	webhookBaseURL string,
) (*pb.InvokeNodeTriggerHookResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	canvas, err := models.FindCanvas(orgID, canvasID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "canvas not found: %v", err)
	}

	node, err := canvas.FindNode(nodeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "node not found: %v", err)
	}

	// Only trigger nodes have trigger actions
	if node.Ref.Data().Trigger == nil {
		return nil, status.Error(codes.InvalidArgument, "node is not a trigger node")
	}

	hookProvider, hookDef, err := registry.FindTriggerHook(node.Ref.Data().Trigger.Name, hookName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "hook not found: %v", err)
	}

	// Check if hook is user accessible
	if hookDef.Type != core.HookTypeUser {
		return nil, status.Errorf(codes.PermissionDenied, "hook '%s' cannot be invoked by user", hookName)
	}

	if err := configuration.ValidateConfiguration(hookDef.Parameters, parameters); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "hook parameter validation failed: %v", err)
	}

	_, err = models.FindActiveUserByID(orgID.String(), userID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	tx := database.Conn()
	logger := logging.ForNode(*node)

	hookCtx := core.TriggerHookContext{
		Name:          hookName,
		Parameters:    parameters,
		Configuration: node.Configuration.Data(),
		HTTP:          registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
	}

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	if node.AppInstallationID != nil {
		integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, status.Error(codes.Internal, "error building context")
		}

		logger = logging.WithIntegration(logger, *integration)
		hookCtx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, onNewEvents)
	}

	hookCtx.Logger = logger
	result, err := hookProvider.HandleHook(hookCtx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "hook execution failed: %v", err)
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	// Convert result to protobuf struct
	resultStruct, err := structpb.NewStruct(result)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create result struct: %v", err)
	}

	return &pb.InvokeNodeTriggerHookResponse{
		Result: resultStruct,
	}, nil
}
