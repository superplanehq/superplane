package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry, orgID string) (*pb.ListIntegrationsResponse, error) {
	db := database.DB(ctx)
	integrations, err := models.ListIntegrations(db, uuid.MustParse(orgID))
	if err != nil {
		log.Errorf("failed to list integrations for organization %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to list integrations")
	}

	protos := []*pb.Integration{}
	for _, integration := range integrations {
		proto, err := serializeIntegration(registry, &integration, []models.CanvasNodeReference{})

		//
		// If we have an issue serializing an integration,
		// we log the error and continue, to avoid failing the entire request.
		//
		if err != nil {
			log.Errorf("failed to serialize integration %s: %v", integration.AppName, err)
			continue
		}

		protos = append(protos, proto)
	}

	return &pb.ListIntegrationsResponse{
		Integrations: protos,
	}, nil
}
