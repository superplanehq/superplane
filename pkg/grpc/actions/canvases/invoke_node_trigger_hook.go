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

func InvokeNodeTriggerHook(
	ctx context.Context,
	authService authorization.Authorization,
	encryptor crypto.Encryptor,
	registry *registry.Registry,
	db *gorm.DB,
	canvas *models.Canvas,
	nodeID string,
	hookName string,
	parameters map[string]any,
	webhookBaseURL string,
) (*pb.InvokeNodeTriggerHookResponse, error) {
	userID, userIsSet := authentication.GetUserIdFromMetadata(ctx)
	if !userIsSet {
		return nil, grpcerrors.Unauthenticated(nil, "user not authenticated")
	}

	node, err := canvas.FindNode(nodeID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "node not found")
	}

	// Only trigger nodes have trigger actions
	if node.Ref.Data().Trigger == nil {
		return nil, grpcerrors.InvalidArgument(nil, "node is not a trigger node")
	}

	hookProvider, hookDef, err := registry.FindTriggerHook(node.Ref.Data().Trigger.Name, hookName)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "hook not found")
	}

	// Check if hook is user accessible
	if hookDef.Type != core.HookTypeUser {
		return nil, grpcerrors.PermissionDenied(nil, fmt.Sprintf("hook '%s' cannot be invoked by user", hookName))
	}

	if err := configuration.ValidateConfiguration(hookDef.Parameters, parameters); err != nil {
		return nil, grpcerrors.InvalidArgument(err, "hook parameter validation failed")
	}

	_, err = models.FindActiveUserByIDInTransaction(db, canvas.OrganizationID.String(), userID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "user not found")
	}

	logger := logging.ForNode(*node)

	newEvents := []models.CanvasEvent{}
	onNewEvents := func(events []models.CanvasEvent) {
		newEvents = append(newEvents, events...)
	}

	expressionParameters := buildHookExpressionParameters(node.Ref.Data().Trigger.Name, hookName, node.Configuration.Data(), parameters)

	userUUID, err := uuid.Parse(userID)
	if err != nil {
		return nil, grpcerrors.Unauthenticated(err, "user not authenticated")
	}

	shouldRecordTriggeredBy := node.Ref.Data().Trigger.Name == "start" && hookName == "run" ||
		node.Ref.Data().Trigger.Name == "schedule" && hookName == "run"

	var hookResult map[string]any
	err = db.Transaction(func(tx *gorm.DB) error {
		resolvedConfiguration, err := contexts.NewNodeConfigurationBuilder(tx, node.WorkflowID).
			WithNodeID(node.NodeID).
			WithExpressionVariables(map[string]any{
				"parameters": expressionParameters,
			}).
			WithConfigurationFields(hookProvider.Configuration()).
			Build(contexts.WithoutRunTitleConfiguration(node.Configuration.Data()))
		if err != nil {
			return grpcerrors.InvalidArgument(err, "failed to resolve trigger configuration")
		}

		var run *models.CanvasRun
		if shouldRecordTriggeredBy {
			run, err = models.CreateCanvasRunInTransaction(tx, canvas.ID, node.NodeID, models.CanvasRunStateStarted, "")
			if err != nil {
				return err
			}

			if err := tx.Model(run).Update("triggered_by", userUUID).Error; err != nil {
				return err
			}

			run.TriggeredBy = &userUUID
		}

		hookCtx := core.TriggerHookContext{
			Name:          hookName,
			Parameters:    parameters,
			Configuration: resolvedConfiguration,
			HTTP:          registry.HTTPContext(),
			Metadata:      contexts.NewNodeMetadataContext(tx, node),
			Requests:      contexts.NewNodeRequestContext(tx, node),
			Webhook:       contexts.NewNodeWebhookContext(ctx, tx, encryptor, node, webhookBaseURL),
			Events:        contexts.NewEventContext(tx, node, run, onNewEvents),
		}

		if node.AppInstallationID != nil {
			integration, err := models.FindUnscopedIntegrationInTransaction(tx, *node.AppInstallationID)
			if err != nil {
				logger.Errorf("error finding app installation: %v", err)
				return grpcerrors.Internal(err, "error building context")
			}

			logger = logging.WithIntegration(logger, *integration)
			hookCtx.Integration = contexts.NewIntegrationContext(tx, node, integration, encryptor, registry, onNewEvents)
		}

		hookCtx.Logger = logger
		hookResult, err = hookProvider.HandleHook(hookCtx)
		if err != nil {
			logger.Errorf("trigger hook %q execution failed: %v", hookName, err)
			return grpcerrors.InvalidArgument(err, "hook execution failed")
		}

		return nil
	})
	if err != nil {
		return nil, err
	}

	if len(newEvents) > 0 {
		if hookResult == nil {
			hookResult = map[string]any{}
		}

		if _, exists := hookResult["event_id"]; !exists {
			hookResult["event_id"] = newEvents[0].ID.String()
		}
	}

	for _, event := range newEvents {
		messages.PublishCanvasEventCreatedMessage(&event)
	}

	// Convert result to protobuf struct
	resultStruct, err := newStructpbStruct(hookResult)
	if err != nil {
		return nil, grpcerrors.Internal(err, "failed to create result struct")
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
		case configuration.FieldTypeString, configuration.FieldTypeText, configuration.FieldTypeSelect:
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
