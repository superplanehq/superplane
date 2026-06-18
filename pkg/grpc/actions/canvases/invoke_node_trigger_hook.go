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

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	expressionParameters := buildHookExpressionParameters(node.Ref.Data().Trigger.Name, hookName, node.Configuration.Data(), parameters)

	resolvedConfiguration, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
		WithNodeID(node.NodeID).
		WithExpressionVariables(map[string]any{
			"parameters": expressionParameters,
		}).
		WithConfigurationFields(hookProvider.Configuration()).
		Build(contexts.WithoutRunTitleConfiguration(node.Configuration.Data()))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "failed to resolve trigger configuration: %v", err)
	}

	hookCtx := core.TriggerHookContext{
		Name:          hookName,
		Parameters:    parameters,
		Configuration: resolvedConfiguration,
		HTTP:          registry.HTTPContext(),
		Metadata:      contexts.NewNodeMetadataContext(tx, node),
		Requests:      contexts.NewNodeRequestContext(tx, node),
		Webhook:       contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
		Events:        contexts.NewEventContext(tx, node, onNewEvents),
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

	if len(newEvents) > 0 {
		if result == nil {
			result = map[string]any{}
		}

		if _, exists := result["event_id"]; !exists {
			result["event_id"] = newEvents[0].ID.String()
		}
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	// Convert result to protobuf struct
	resultStruct, err := newStructpbStruct(result)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create result struct: %v", err)
	}

	return &pb.InvokeNodeTriggerHookResponse{
		Result: resultStruct,
	}, nil
}

func buildHookExpressionParameters(triggerName string, hookName string, configuration map[string]any, hookParameters map[string]any) map[string]any {
	parameters := map[string]any{}

	if triggerName == "start" && hookName == "run" {
		for key, value := range startTemplateDefaultParameters(configuration, hookParameters) {
			parameters[key] = value
		}
	}

	for key, value := range hookParameters {
		parameters[key] = value
	}

	return parameters
}

func startTemplateDefaultParameters(configuration map[string]any, hookParameters map[string]any) map[string]any {
	templateName, _ := hookParameters["template"].(string)
	if templateName == "" {
		return nil
	}

	rawTemplates, _ := configuration["templates"].([]any)
	for _, rawTemplate := range rawTemplates {
		template, ok := rawTemplate.(map[string]any)
		if !ok {
			continue
		}
		name, _ := template["name"].(string)
		if name != templateName {
			continue
		}
		return defaultsFromTemplateParameters(template)
	}

	return nil
}

func defaultsFromTemplateParameters(template map[string]any) map[string]any {
	rawParameters, _ := template["parameters"].([]any)
	if len(rawParameters) == 0 {
		return nil
	}

	parameters := map[string]any{}
	for _, rawParameter := range rawParameters {
		parameter, ok := rawParameter.(map[string]any)
		if !ok {
			continue
		}

		name, _ := parameter["name"].(string)
		if name == "" {
			continue
		}

		switch parameterType, _ := parameter["type"].(string); parameterType {
		case configuration.FieldTypeNumber:
			if value, exists := parameter["defaultNumber"]; exists && value != nil {
				parameters[name] = value
			}
		case configuration.FieldTypeBool:
			if value, exists := parameter["defaultBoolean"]; exists && value != nil {
				parameters[name] = value
			}
		case configuration.FieldTypeString, configuration.FieldTypeSelect:
			if value, exists := parameter["defaultString"]; exists && value != nil {
				if textValue, isString := value.(string); isString && textValue == "" {
					continue
				}
				parameters[name] = value
			}
		}
	}

	if len(parameters) == 0 {
		return nil
	}

	return parameters
}
