package organizations

import (
	"context"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListIntegrationsV2(ctx context.Context, registry *registry.Registry, orgID string) (*pb.ListIntegrationsV2Response, error) {
	integrations, err := models.ListIntegrationsV2(uuid.MustParse(orgID))
	if err != nil {
		return nil, err
	}

	protos := []*pb.IntegrationV2{}
	for _, integration := range integrations {
		proto, err := serializeIntegrationV2(&integration)

		//
		// If we have an issue serializing an integration,
		// we log the error and continue, to avoid failing the entire request.
		//
		if err != nil {
			log.Errorf("failed to serialize integration %s: %v", integration.IntegrationName, err)
			continue
		}

		protos = append(protos, proto)
	}

	return &pb.ListIntegrationsV2Response{
		Integrations: protos,
	}, nil
}
