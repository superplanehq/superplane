package organizations

import (
	"context"
	"fmt"
	"maps"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/datatypes"
)

func UpdateApplication(ctx context.Context, registry *registry.Registry, baseURL string, orgID string, installationID string, configuration map[string]any) (*pb.UpdateApplicationResponse, error) {
	appInstallation, err := models.FindUnscopedAppInstallation(uuid.MustParse(installationID))
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
		return nil, status.Errorf(codes.Internal, "failed to encrypt sensitive configuration: %v", err)
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
		Configuration:   appInstallation.Configuration.Data(),
		BaseURL:         baseURL,
		OrganizationID:  orgID,
		InstallationID:  installationID,
		AppInstallation: appCtx,
	})

	if syncErr != nil {
		appInstallation.State = "error"
		appInstallation.StateDescription = fmt.Sprintf("Sync failed: %v", syncErr)
	} else {
		appInstallation.StateDescription = ""
	}

	err = database.Conn().Save(appInstallation).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save application installation: %v", err)
	}

	proto, err := serializeAppInstallation(registry, appInstallation, []models.WorkflowNodeReference{})
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize application installation: %v", err)
	}

	return &pb.UpdateApplicationResponse{
		Installation: proto,
	}, nil
}
