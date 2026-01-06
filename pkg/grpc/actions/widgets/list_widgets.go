package widgets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListWidgets(ctx context.Context, registry *registry.Registry) (*pb.ListWidgetsResponse, error) {
	return &pb.ListWidgetsResponse{
		Widgets: actions.SerializeWidgets(registry.ListWidgets()),
	}, nil
}
