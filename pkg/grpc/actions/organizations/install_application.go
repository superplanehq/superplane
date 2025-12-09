package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func InstallApplication(ctx context.Context, registry *registry.Registry, baseURL string, orgID string, appName, installationName string, appConfig *structpb.Struct) (*pb.InstallApplicationResponse, error) {
	app, err := registry.GetApplication(appName)
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "application %s not found", appName)
	}

	//
	// TODO: do not save sensitive values as plain text here
	//
	appInstallation, err := models.CreateAppInstallation(uuid.MustParse(orgID), appName, installationName, appConfig.AsMap())
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to create application installation: %v", err)
	}

	syncErr := app.Sync(applications.SyncContext{
		Configuration:  appInstallation.Configuration.Data(),
		BaseURL:        baseURL,
		OrganizationID: orgID,
		InstallationID: appInstallation.ID.String(),
		AppContext:     contexts.NewAppContext(database.Conn(), appInstallation),
	})

	err = database.Conn().Save(appInstallation).Error
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to save application installation after sync: %v", err)
	}

	if syncErr != nil {
		appInstallation.State = "error"
		appInstallation.StateDescription = syncErr.Error()
		err = database.Conn().Save(appInstallation).Error
		if err != nil {
			return nil, status.Errorf(codes.Internal, "failed to save application installation after sync: %v", err)
		}
	}

	proto, err := serializeAppInstallation(appInstallation)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "failed to serialize application installation: %v", err)
	}

	return &pb.InstallApplicationResponse{
		Installation: proto,
	}, nil
}

func serializeAppInstallation(appInstallation *models.AppInstallation) (*pb.AppInstallation, error) {
	config, err := structpb.NewStruct(appInstallation.Configuration.Data())
	if err != nil {
		return nil, err
	}

	metadata, err := structpb.NewStruct(appInstallation.Metadata.Data())
	if err != nil {
		return nil, err
	}

	proto := &pb.AppInstallation{
		Id:               appInstallation.ID.String(),
		AppName:          appInstallation.AppName,
		InstallationName: appInstallation.InstallationName,
		State:            appInstallation.State,
		Configuration:    config,
		Metadata:         metadata,
	}

	if appInstallation.BrowserAction != nil {
		proto.BrowserAction = &pb.BrowserAction{
			Url:        appInstallation.BrowserAction.Data().URL,
			Method:     appInstallation.BrowserAction.Data().Method,
			FormFields: appInstallation.BrowserAction.Data().FormFields,
		}
	}

	return proto, nil
}
