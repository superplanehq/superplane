package triggers

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/pkg/triggers"
)

func ListTriggers(ctx context.Context, registry *registry.Registry) (*pb.ListTriggersResponse, error) {
	return &pb.ListTriggersResponse{
		Triggers: serializeTriggers(registry.ListTriggers()),
	}, nil
}

func serializeTriggers(in []triggers.Trigger) []*pb.Trigger {
	out := make([]*pb.Trigger, len(in))
	for i, trigger := range in {
		configFields := trigger.Configuration()
		configuration := make([]*compb.ConfigurationField, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.Trigger{
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Configuration: configuration,
		}
	}
	return out
}
