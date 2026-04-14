package components

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"google.golang.org/protobuf/types/known/structpb"
)

func DescribeComponent(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeComponentResponse, error) {
	component, err := registry.GetComponent(name)
	if err != nil {
		return nil, status.Errorf(codes.NotFound, "component %s not found", name)
	}

	outputChannels := component.OutputChannels(nil)
	channels := make([]*pb.OutputChannel, len(outputChannels))
	for i, channel := range outputChannels {
		channels[i] = &pb.OutputChannel{
			Name: channel.Name,
		}
	}

	configFields := component.Configuration()
	configuration := make([]*configpb.Field, len(configFields))
	for i, field := range configFields {
		configuration[i] = actions.ConfigurationFieldToProto(field)
	}

	var exampleOutput *structpb.Struct
	if output := component.ExampleOutput(); output != nil {
		exampleOutput, _ = structpb.NewStruct(output)
	}

	return &pb.DescribeComponentResponse{
		Component: &pb.Component{
			Name:           component.Name(),
			Label:          component.Label(),
			Description:    component.Description(),
			Icon:           component.Icon(),
			Color:          component.Color(),
			OutputChannels: channels,
			Configuration:  configuration,
			ExampleOutput:  exampleOutput,
		},
	}, nil
}
