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
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/logging"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
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
		return nil, grpcerrors.InvalidArgument(err, "invalid organization ID")
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, grpcerrors.InvalidArgument(err, "invalid integration ID")
	}

	instance, err := models.FindIntegration(org, ID)
	if err != nil {
		return nil, grpcerrors.NotFound(err, "integration not found")
	}

	if !instance.IsLegacy() {
		return nil, grpcerrors.InvalidArgument(nil, fmt.Sprintf("integration %s is not a legacy setup", instance.ID.String()))
	}

	if name != "" && name != instance.InstallationName {
		existing, err := models.FindIntegrationByName(database.Conn(), org, name)
		if err == nil && existing.ID != instance.ID {
			return nil, grpcerrors.AlreadyExists(nil, fmt.Sprintf("an integration with the name %s already exists in this organization", name))
		}

		if err != nil && !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.Internal(err, "failed to verify integration name uniqueness")
		}

		instance.InstallationName = name
	}

	integration, err := registry.GetIntegration(instance.AppName)
	if err != nil {
		return nil, grpcerrors.Internal(err, "integration not found")
	}

	if configuration == nil {
		configuration = map[string]any{}
	}

	existingConfig := instance.Configuration.Data()
	configuration, err = encryptConfigurationIfNeeded(ctx, registry, integration, configuration, instance.ID, existingConfig)
	if err != nil {
		log.Errorf("failed to encrypt sensitive configuration for integration %s: %v", instance.ID, err)
		return nil, grpcerrors.Internal(err, "failed to encrypt sensitive configuration")
	}

	maps.Copy(existingConfig, configuration)
	instance.Configuration = datatypes.NewJSONType(existingConfig)

	integrationCtx := contexts.NewIntegrationContext(
		database.Conn(),
		nil,
		instance,
		registry.Encryptor,
		registry,
		nil,
	)

	logging.ForIntegration(*instance).WithField("source", "integration_update").Info("Integration operation may write secrets")
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
		return nil, grpcerrors.Internal(err, "failed to save integration")
	}

	proto, err := serializeIntegration(registry, instance, []models.CanvasNodeReference{})
	if err != nil {
		log.Errorf("failed to serialize integration %s: %v", instance.ID, err)
		return nil, grpcerrors.Internal(err, "failed to serialize integration")
	}

	return &pb.UpdateIntegrationResponse{
		Integration: proto,
	}, nil
}
