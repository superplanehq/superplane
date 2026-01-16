package triggers

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/triggers"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/protobuf/types/known/structpb"
)

func DescribeTrigger(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeTriggerResponse, error) {
	trigger, err := registry.GetTrigger(name)
	if err != nil {
		return nil, err
	}

	configFields := trigger.Configuration()
	configFields = actions.AppendGlobalTriggerFields(configFields)
	configuration := make([]*configpb.Field, len(configFields))
	for i, field := range configFields {
		configuration[i] = actions.ConfigurationFieldToProto(field)
	}

	var exampleData *structpb.Struct
	if data := trigger.ExampleData(); data != nil {
		exampleData, _ = structpb.NewStruct(data)
	}

	return &pb.DescribeTriggerResponse{
		Trigger: &pb.Trigger{
			Name:          trigger.Name(),
			Label:         trigger.Label(),
			Description:   trigger.Description(),
			Icon:          trigger.Icon(),
			Color:         trigger.Color(),
			Configuration: configuration,
			ExampleData:   exampleData,
		},
	}, nil
}
