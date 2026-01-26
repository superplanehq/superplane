package organizations

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/oidc"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func UpdateIntegration(ctx context.Context, registry *registry.Registry, oidcProvider oidc.Provider, baseURL string, webhooksBaseURL string, orgID string, integrationID string, configuration map[string]any) (*pb.UpdateIntegrationResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid organization ID: %v", err)
	}

	ID, err := uuid.Parse(integrationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid integration ID: %v", err)
	}

	i, err := models.FindAppInstallation(org, ID)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "integration not found: %v", err)
	}

	integration, err := registry.GetIntegration(i.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "integration %s not found", i.AppName)
	}

	existingConfig := i.Configuration.Data()
	configuration, err = encryptConfigurationIfNeeded(ctx, registry, integration, configuration, i.ID, existingConfig)
	if err != nil {
		log.Errorf("failed to encrypt sensitive configuration for integration %s: %v", i.ID, err)
		return nil, status.Error(codes.Internal, "failed to encrypt sensitive configuration")
	}

	maps.Copy(existingConfig, configuration)
	i.Configuration = datatypes.NewJSONType(existingConfig)

	appCtx := contexts.NewAppInstallationContext(
		database.Conn(),
		nil,
		i,
		registry.Encryptor,
		registry,
	)

	syncErr := integration.Sync(core.SyncContext{
		HTTP:            contexts.NewHTTPContext(registry.GetHTTPClient()),
		Configuration:   i.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		InstallationID:  i.ID.String(),
		AppInstallation: appCtx,
		OIDC:            oidcProvider,
	})

	if syncErr != nil {
		i.State = "error"
		i.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		i.StateDescription = ""
	}

	err = database.Conn().Save(i).Error
	if err != nil {
		log.Errorf("failed to save integration %s: %v", i.ID, err)
		return nil, status.Error(codes.Internal, "failed to save integration")
	}

	proto, err := serializeIntegration(registry, i, []models.WorkflowNodeReference{})
	if err != nil {
		log.Errorf("failed to serialize integration %s: %v", i.ID, err)
		return nil, status.Error(codes.Internal, "failed to serialize integration")
	}

	return &pb.UpdateIntegrationResponse{
		Integration: proto,
	}, nil
}
