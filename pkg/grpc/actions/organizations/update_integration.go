package organizations

import (
	"context"
	"errors"
	"fmt"
	"maps"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
	"gorm.io/gorm"
)

func UpdateIntegration(
	ctx context.Context,
	registry *registry.Registry,
	oidcProvider oidc.Provider,
	baseURL string,
	webhooksBaseURL string,
	orgID string,
	integrationID string,
	configuration map[string]any,
	name string,
) (*pb.UpdateIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid integration ID: %v", err)
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "integration not found: %v", err)
	}

	if name != "" && name != instance.InstallationName {
		existing, err := models.FindIntegrationByName(org, name)
		if err == nil && existing.ID != instance.ID {
			return nil, status.Errorf(codes.AlreadyExists, "an integration with the name %s already exists in this organization", name)
		}

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, status.Error(codes.Internal, "failed to verify integration name uniqueness")
		}

		instance.InstallationName = name
	}

	integration, err := registry.GetIntegration(instance.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "integration %s not found", instance.AppName)
	}

	if configuration == nil {
		configuration = map[string]any{}
	}

	existingConfig := instance.Configuration.Data()
	configuration, err = encryptConfigurationIfNeeded(ctx, registry, integration, configuration, instance.ID, existingConfig)
	if err != nil {
		log.Errorf("failed to encrypt sensitive configuration for integration %s: %v", instance.ID, err)
		return nil, status.Error(codes.Internal, "failed to encrypt sensitive configuration")
	}

	maps.Copy(existingConfig, configuration)
	instance.Configuration = datatypes.NewJSONType(existingConfig)

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		registry.Encryptor,
		registry,
	)

	syncErr := integration.Sync(core.SyncContext{
		Logger:          logging.ForIntegration(*instance),
		HTTP:            registry.HTTPContext(),
		Configuration:   instance.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		Integration:     integrationCtx,
		OIDC:            oidcProvider,
	})

	if syncErr != nil {
		instance.State = "error"
		instance.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		instance.StateDescription = ""
	}

	err = database.Conn().Save(instance).Error
	if err != nil {
		log.Errorf("failed to save integration %s: %v", instance.ID, err)
		return nil, status.Error(codes.Internal, "failed to save integration")
	}

	proto, err := serializeIntegration(registry, instance, []models.CanvasNodeReference{})
	if err != nil {
		log.Errorf("failed to serialize integration %s: %v", instance.ID, err)
		return nil, status.Error(codes.Internal, "failed to serialize integration")
	}

	return &pb.UpdateIntegrationResponse{
		Integration: proto,
	}, nil
}
