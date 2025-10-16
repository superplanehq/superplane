package components

import (
	"context"

	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribeComponent(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeComponentResponse, error) {
	component, err := registry.GetComponent(name)
	if err != nil {
		return nil, err
	}

	outputChannels := component.OutputChannels(nil)
	channels := make([]*pb.OutputChannel, len(outputChannels))
	for i, channel := range outputChannels {
		channels[i] = &pb.OutputChannel{
			Name: channel.Name,
		}
	}

	configFields := component.Configuration()
	configuration := make([]*pb.ConfigurationField, len(configFields))
	for i, field := range configFields {
		configuration[i] = ConfigurationFieldToProto(field)
	}

	return &pb.DescribeComponentResponse{
		Component: &pb.Component{
			Name:           component.Name(),
			OutputChannels: channels,
			Configuration:  configuration,
		},
	}, nil
}
