package integrations

import (
	"context"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListIntegrations(ctx context.Context, domainType, domainID string) (*pb.ListIntegrationsResponse, error) {
	integrations, err := models.ListIntegrations(domainType, uuid.MustParse(domainID))
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
