package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/workers/contexts"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListApplicationResources(ctx context.Context, registry *registry.Registry, orgID, installationID, resourceType string) (*pb.ListApplicationResourcesResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	installation, err := uuid.Parse(installationID)
	if err != nil {
		return nil, status.Error(codes.InvalidArgument, "invalid installation ID")
	}

	appInstallation, err := models.FindAppInstallation(org, installation)
	if err != nil {
		return nil, err
	}

	app, err := registry.GetApplication(appInstallation.AppName)
	if err != nil {
		return nil, status.Errorf(codes.Internal, "application %s not found", appInstallation.AppName)
	}

	appCtx := contexts.NewAppInstallationContext(
		database.Conn(),
		nil,
		appInstallation,
		registry.Encryptor,
		registry,
	)

	listCtx := core.ListResourcesContext{
		Logger: log.WithFields(log.Fields{
			"app_installation_id": appInstallation.ID.String(),
			"app_name":            appInstallation.AppName,
			"resource_type":       resourceType,
		}),
		HTTP:            contexts.NewHTTPContext(registry.GetHTTPClient()),
		AppInstallation: appCtx,
	}

	resources, err := app.ListResources(resourceType, listCtx)
	if err != nil {
		log.WithError(err).Errorf("failed to list resources for app installation %s", appInstallation.ID)
		return nil, status.Error(codes.Internal, "failed to list application resources")
	}

	return &pb.ListApplicationResourcesResponse{
		Resources: serializeAppInstallationResources(resources),
	}, nil
}

func serializeAppInstallationResources(resources []core.ApplicationResource) []*pb.AppInstallationResourceRef {
	out := make([]*pb.AppInstallationResourceRef, 0, len(resources))
	for _, resource := range resources {
		out = append(out, &pb.AppInstallationResourceRef{
			Type: resource.Type,
			Name: resource.Name,
			Id:   resource.ID,
		})
	}
	return out
}
