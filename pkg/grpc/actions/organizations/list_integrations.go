package organizations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/organizations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry, orgID string) (*pb.ListIntegrationsResponse, error) {
	integrations, err := models.ListIntegrations(uuid.MustParse(orgID))
	if err != nil {
		return nil, err
	}

	protos := []*pb.Integration{}
	for _, integration := range integrations {
		proto, err := serializeIntegration(registry, &integration, []models.WorkflowNodeReference{})
		if err != nil {
			return nil, err
		}

		protos = append(protos, proto)
	}

	return &pb.ListIntegrationsResponse{
		Integrations: protos,
	}, nil
}
