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

func UpdateApplication(ctx context.Context, registry *registry.Registry, oidcSigner *oidc.Signer, baseURL string, webhooksBaseURL string, orgID string, installationID string, configuration map[string]any) (*pb.UpdateApplicationResponse, error) {
	installation, err := uuid.Parse(installationID)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid installation ID: %v", err)
	}

	appInstallation, err := models.FindUnscopedAppInstallation(installation)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "application installation not found: %v", err)
	}

	app, err := registry.GetApplication(appInstallation.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "application %s not found", appInstallation.AppName)
	}

	existingConfig := appInstallation.Configuration.Data()
	configuration, err = encryptConfigurationIfNeeded(ctx, registry, app, configuration, appInstallation.ID, existingConfig)
	if err != nil {
		log.Errorf("failed to encrypt sensitive configuration for app installation %s: %v", appInstallation.ID, err)
		return nil, status.Error(codes.Internal, "failed to encrypt sensitive configuration")
	}

	maps.Copy(existingConfig, configuration)
	appInstallation.Configuration = datatypes.NewJSONType(existingConfig)

	appCtx := contexts.NewAppInstallationContext(
		database.Conn(),
		nil,
		appInstallation,
		registry.Encryptor,
		registry,
	)

	syncErr := app.Sync(core.SyncContext{
		HTTP:            contexts.NewHTTPContext(registry.GetHTTPClient()),
		Configuration:   appInstallation.Configuration.Data(),
		BaseURL:         baseURL,
		WebhooksBaseURL: webhooksBaseURL,
		OrganizationID:  orgID,
		InstallationID:  installationID,
		AppInstallation: appCtx,
		OIDCSigner:      oidcSigner,
	})

	if syncErr != nil {
		appInstallation.State = "error"
		appInstallation.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		appInstallation.StateDescription = ""
	}

	err = database.Conn().Save(appInstallation).Error
	if err != nil {
		log.Errorf("failed to save application installation %s: %v", appInstallation.ID, err)
		return nil, status.Error(codes.Internal, "failed to save application installation")
	}

	proto, err := serializeAppInstallation(registry, appInstallation, []models.WorkflowNodeReference{})
	if err != nil {
		log.Errorf("failed to serialize application installation %s: %v", appInstallation.ID, err)
		return nil, status.Error(codes.Internal, "failed to serialize application installation")
	}

	return &pb.UpdateApplicationResponse{
		Installation: proto,
	}, nil
}
