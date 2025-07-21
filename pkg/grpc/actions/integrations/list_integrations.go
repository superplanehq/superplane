package integrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/superplane"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrations(ctx context.Context, domainType string, domainID uuid.UUID) (*pb.ListIntegrationsResponse, error) {
	integrations, err := models.ListIntegrations(domainType, domainID)
	if err != nil {
		return nil, status.Error(codes.Internal, "failed to list integrations")
	}

	return &pb.ListIntegrationsResponse{
		Integrations: serializeIntegrations(integrations),
	}, nil
}

func serializeIntegrations(integrations []*models.Integration) []*pb.Integration {
	out := []*pb.Integration{}
	for _, integration := range integrations {
		out = append(out, serializeIntegration(*integration))
	}
	return out
}
