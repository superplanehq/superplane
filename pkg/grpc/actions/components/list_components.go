package components

import (
	"context"

	"github.com/superplanehq/superplane/pkg/database"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListComponents(ctx context.Context, registry *registry.Registry, organizationID string) (*pb.ListComponentsResponse, error) {
	return &pb.ListComponentsResponse{
		Components: actions.SerializeComponents(registry.ListComponentsForOrganization(database.Conn(), organizationID)),
	}, nil
}
