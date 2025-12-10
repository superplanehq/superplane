package applications

import (
	"context"

	"github.com/superplanehq/superplane/pkg/applications"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	pb "github.com/superplanehq/superplane/pkg/protos/applications"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListApplications(ctx context.Context, registry *registry.Registry) (*pb.ListApplicationsResponse, error) {
	applications := registry.ListApplications()

	return &pb.ListApplicationsResponse{
		Applications: serializeApplications(applications),
	}, nil
}

func serializeApplications(in []applications.Application) []*pb.ApplicationDefinition {
	out := make([]*pb.ApplicationDefinition, len(in))
	for i, application := range in {
		configFields := application.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.ApplicationDefinition{
			Name:          application.Name(),
			Label:         application.Label(),
			Configuration: configuration,
			Components:    actions.SerializeComponents(application.Components()),
			Triggers:      actions.SerializeTriggers(application.Triggers()),
		}
	}
	return out
}
