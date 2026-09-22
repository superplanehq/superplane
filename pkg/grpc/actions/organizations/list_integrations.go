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
	org, err := resolveOrganizationID(ctx, orgID)
	if err != nil {
		return nil, err
	}

	db := database.DB(ctx)
	integrations, err := models.ListIntegrations(db, org)
	if err != nil {
		log.Errorf("failed to list integrations for organization %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to list integrations")
	}

	installationIDs := make([]uuid.UUID, len(integrations))
	for i, integration := range integrations {
		installationIDs[i] = integration.ID
	}
	secretsByInstallation, err := models.ListIntegrationSecretsForInstallations(db, installationIDs)
	if err != nil {
		log.Errorf("failed to list integration secrets for organization %s: %v", orgID, err)
		return nil, grpcerrors.Internal(err, "failed to list integrations")
	}

	protos := []*pb.Integration{}
	for _, integration := range integrations {
		proto, err := serializeIntegrationWithSecrets(registry, &integration, []models.CanvasNodeReference{}, secretsByInstallation[integration.ID])

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
