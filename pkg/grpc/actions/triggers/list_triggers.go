package triggers

import (
	"context"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListTriggers(ctx context.Context, registry *registry.Registry) (*pb.ListTriggersResponse, error) {
	return &pb.ListTriggersResponse{
		Triggers: serializeTriggers(registry.ListTriggers()),
	}, nil
}

func serializeTriggers(in []core.Trigger) []*pb.Trigger {
	out := make([]*pb.Trigger, len(in))
	for i, trigger := range in {
		configFields := trigger.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.Trigger{
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Icon:          trigger.Icon(),
			Color:         trigger.Color(),
			Configuration: configuration,
		}
	}
	return out
}
