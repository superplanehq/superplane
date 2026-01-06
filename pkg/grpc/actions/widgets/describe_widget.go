package widgets

import (
	"context"

	"github.com/superplanehq/superplane/pkg/grpc/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/widgets"
	"github.com/superplanehq/superplane/pkg/registry"
)

func DescribeWidget(ctx context.Context, registry *registry.Registry, name string) (*pb.DescribeWidgetResponse, error) {
	widget, err := registry.GetWidget(name)
	if err != nil {
		return nil, err
	}

	configFields := widget.Configuration()
	configuration := make([]*configpb.Field, len(configFields))
	for i, field := range configFields {
		configuration[i] = actions.ConfigurationFieldToProto(field)
	}

	return &pb.DescribeWidgetResponse{
		Widget: &pb.Widget{
			Name:          widget.Name(),
			Label:         widget.Label(),
			Description:   widget.Description(),
			Icon:          widget.Icon(),
			Color:         widget.Color(),
			Configuration: configuration,
		},
	}, nil
}
