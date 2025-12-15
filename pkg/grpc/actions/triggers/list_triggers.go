package triggers

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListTriggers(ctx context.Context, registry *registry.Registry) (*pb.ListTriggersResponse, error) {
	return &pb.ListTriggersResponse{
		Triggers: actions.SerializeTriggers(registry.ListTriggers()),
	}, nil
}
