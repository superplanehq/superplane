package organizations

import (
	"context"
	"errors"
	"slices"

	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/database"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
	"gorm.io/gorm"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry, orgID, factoryRef string) (*pb.ListIntegrationsResponse, error) {
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

	var usage *workspaceIntegrationUsage
	if factoryRef != "" {
		factory, err := models.FindFactoryByRef(db, org, factoryRef)
		if errors.Is(err, models.ErrFactoryNotFound) || errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, grpcerrors.NotFound(err, "workspace not found")
		}
		if err != nil {
			return nil, grpcerrors.Internal(err, "could not load workspace")
		}
		usage, err = integrationUsageForWorkspace(db, registry, factory, integrations)
		if err != nil {
			return nil, grpcerrors.Internal(err, "could not load workspace integration usage")
		}
		integrations = slices.DeleteFunc(integrations, func(integration models.Integration) bool {
			return !usage.integrationIDs[integration.ID]
		})
	}

	failures, err := integrationWebhookFailures(db, integrations)
	if err != nil {
		return nil, grpcerrors.Internal(err, "could not load webhook failures")
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

		proto.Status.WebhookFailures = failures[integration.ID]
		if usage != nil {
			proto.Status.WebhookFailures = slices.DeleteFunc(proto.Status.WebhookFailures, func(failure *pb.IntegrationWebhookFailure) bool {
				return !usage.webhookIDs[failure.Id]
			})
		}
		protos = append(protos, proto)
	}

	return &pb.ListIntegrationsResponse{
		Integrations: protos,
	}, nil
}
