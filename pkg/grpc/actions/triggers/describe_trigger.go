package triggers

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	compb "github.com/superplanehq/superplane/pkg/protos/components"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribeTrigger(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeTriggerResponse, error) {
	trigger, err := registry.GetTrigger(name)
	if err != nil {
		return nil, err
	}

	configFields := trigger.Configuration()
	configuration := make([]*compb.ConfigurationField, len(configFields))
	for i, field := range configFields {
		configuration[i] = actions.ConfigurationFieldToProto(field)
	}

	return &pb.DescribeTriggerResponse{
		Trigger: &pb.Trigger{
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Icon:          trigger.Icon(),
			Color:         trigger.Color(),
			Configuration: configuration,
		},
	}, nil
}
