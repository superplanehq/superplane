package components

import (
	"context"

	"github.com/superplanehq/superplane/pkg/components"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/components"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListComponents(ctx context.Context, registry *registry.Registry) (*pb.ListComponentsResponse, error) {
	return &pb.ListComponentsResponse{
		Components: serializeComponents(registry.ListComponents()),
	}, nil
}

func serializeComponents(in []components.Component) []*pb.Component {
	out := make([]*pb.Component, len(in))
	for i, component := range in {
		outputChannels := component.OutputChannels(nil)
		channels := make([]*pb.OutputChannel, len(outputChannels))
		for j, channel := range outputChannels {
			channels[j] = &pb.OutputChannel{
				Name: channel.Name,
			}
		}

		configFields := component.Configuration()
		configuration := make([]*pb.ConfigurationField, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.Component{
			Name:           component.Name(),
			Label:          component.Label(),
			Description:    component.Description(),
			Icon:           component.Icon(),
			Color:          component.Color(),
			OutputChannels: channels,
			Configuration:  configuration,
		}
	}
	return out
}
