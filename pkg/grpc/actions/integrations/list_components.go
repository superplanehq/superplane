package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func ListComponents(ctx context.Context, registry *registry.Registry, integrationType string) (*pb.ListComponentsResponse, error) {
	components, err := registry.ListIntegrationComponents(integrationType)
	if err != nil {
		return nil, status.Error(codes.NotFound, "integration not found")
	}

	return &pb.ListComponentsResponse{
		Components: actions.SerializeComponents(components),
	}, nil
}
