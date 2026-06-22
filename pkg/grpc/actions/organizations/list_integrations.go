package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry, orgID string) (*pb.ListIntegrationsResponse, error) {
	org, err := uuid.Parse(orgID)
	if err != nil {
		log.WithError(err).
			WithField("organization_id", orgID).
			Error("list integrations received an invalid organization id")
		return nil, status.Error(codes.InvalidArgument, "invalid organization ID")
	}

	integrations, err := models.ListIntegrations(org)
	if err != nil {
		log.WithError(err).
			WithField("organization_id", orgID).
			Error("failed to list integrations for organization")
		return nil, status.Error(codes.Internal, "failed to list integrations")
	}

	protos := []*pb.Integration{}
	for _, integration := range integrations {
		proto, err := serializeIntegration(registry, &integration, []models.CanvasNodeReference{})

		//
		// If we have an issue serializing an integration,
		// we log the error and continue, to avoid failing the entire request.
		//
		if err != nil {
			log.WithError(err).
				WithField("organization_id", orgID).
				WithField("integration_id", integration.ID.String()).
				WithField("app_name", integration.AppName).
				Error("failed to serialize integration")
			continue
		}

		protos = append(protos, proto)
	}

	return &pb.ListIntegrationsResponse{
		Integrations: protos,
	}, nil
}
