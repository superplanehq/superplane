package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func InvokeIntegrationAction(
	ctx context.Context,
	registry *registry.Registry,
	webhooksBaseURL string,
	orgID string,
	integrationID string,
	actionName string,
	parameters map[string]any,
) (*pb.InvokeIntegrationActionResponse, error) {
	orgUUID, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	integrationUUID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid integration ID: %v", err)
	}

	if actionName == "" {
		return nil, status.Error(codes.InvalidArgument, "action_name is required")
	}

	integration, err := models.FindIntegration(orgUUID, integrationUUID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "integration not found: %v", err)
	}

	integrationImpl, err := registry.GetIntegration(integration.AppName)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "integration %s not found", integration.AppName)
	}

	actionDef := findIntegrationAction(integrationImpl, actionName)
	if actionDef == nil {
		return nil, status.Errorf(codes.NotFound, "action '%s' not found for integration '%s'", actionName, integration.AppName)
	}

	if parameters == nil {
		parameters = map[string]any{}
	}

	if err := configuration.ValidateConfiguration(actionDef.Parameters, parameters); err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action parameter validation failed: %v", err)
	}

	logger := logging.ForIntegration(*integration)
	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		integration,
		registry.Encryptor,
		registry,
	)

	actionCtx := core.IntegrationActionContext{
		WebhooksBaseURL: webhooksBaseURL,
		Name:            actionName,
		Parameters:      parameters,
		Configuration:   integration.Configuration.Data(),
		Logger:          logger,
		Integration:     integrationCtx,
		HTTP:            registry.HTTPContext(),
	}

	actionErr := integrationImpl.HandleAction(actionCtx)

	if err := database.Conn().Save(integration).Error; err != nil {
		logger.Errorf("failed to save integration %s: %v", integration.ID, err)
		return nil, status.Error(codes.Internal, "failed to save integration")
	}

	if actionErr != nil {
		return nil, status.Errorf(codes.InvalidArgument, "action execution failed: %v", actionErr)
	}

	return &pb.InvokeIntegrationActionResponse{}, nil
}

func findIntegrationAction(integration core.Integration, actionName string) *core.Action {
	for _, action := range integration.Actions() {
		if action.Name == actionName {
			return &action
		}
	}

	return nil
}
