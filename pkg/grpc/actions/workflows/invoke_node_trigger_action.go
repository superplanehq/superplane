package workflows

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/authentication"
	"github.com/superplanehq/superplane/pkg/authorization"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/crypto"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/workflows"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func InvokeNodeTriggerAction(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	orgID uuid.UUID,
	workflowID uuid.UUID,
	nodeID string,
	actionName string,
	parameters map[string]any,
	webhookBaseURL string,
) (*pb.InvokeNodeTriggerActionResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, status.Error(codes.Unauthenticated, "user not authenticated")
	}

	workflow, err := models.FindWorkflow(orgID, workflowID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "workflow not found: %v", err)
	}

	node, err := workflow.FindNode(nodeID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "node not found: %v", err)
	}

	// Only trigger nodes have trigger actions
	if node.Ref.Data().Trigger == nil {
		return nil, status.Error(codes.InvalidArgument, "node is not a trigger node")
	}

	trigger, err := registry.GetTrigger(node.Ref.Data().Trigger.Name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "trigger not found: %v", err)
	}

	actionDef := findTriggerAction(trigger, actionName)
	if actionDef == nil {
		return nil, status.Errorf(codes.NotFound, "action '%s' not found for trigger '%s'", actionName, node.Ref.Data().Trigger.Name)
	}

	// Check if action is user accessible
	if !actionDef.UserAccessible {
		return nil, status.Errorf(codes.PermissionDenied, "action '%s' is not user accessible", actionName)
	}

	if err := configuration.ValidateConfiguration(actionDef.Parameters, parameters); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action parameter validation failed: %v", err)
	}

	_, err = models.FindActiveUserByID(orgID.String(), userID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "user not found: %v", err)
	}

	tx := database.Conn()
	logger := logging.ForNode(*node)

	actionCtx := core.TriggerActionContext{
		Name:            actionName,
		Parameters:      parameters,
		Configuration:   node.Configuration.Data(),
		MetadataContext: contexts.NewNodeMetadataContext(tx, node),
		RequestContext:  contexts.NewNodeRequestContext(tx, node),
		WebhookContext:  contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
	}

	if node.AppInstallationID != nil {
		appInstallation, err := models.FindUnscopedAppInstallationInTransaction(tx, *node.AppInstallationID)
		if err != nil {
			logger.Errorf("error finding app installation: %v", err)
			return nil, status.Error(codes.Internal, "error building context")
		}

		logger = logging.WithAppInstallation(logger, *appInstallation)
		actionCtx.AppInstallationContext = contexts.NewAppInstallationContext(tx, node, appInstallation, encryptor, registry)
	}

	actionCtx.Logger = logger
	result, err := trigger.HandleAction(actionCtx)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action execution failed: %v", err)
	}

	// Convert result to protobuf struct
	resultStruct, err := structpb.NewStruct(result)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create result struct: %v", err)
	}

	return &pb.InvokeNodeTriggerActionResponse{
		Result: resultStruct,
	}, nil
}

func findTriggerAction(trigger core.Trigger, actionName string) *core.Action {
	for _, action := range trigger.Actions() {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
