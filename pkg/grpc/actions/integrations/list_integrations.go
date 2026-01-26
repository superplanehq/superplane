package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry) (*pb.ListIntegrationsResponse, error) {
	integrations := registry.ListIntegrations()

	return &pb.ListIntegrationsResponse{
		Integrations: serializeIntegrations(integrations),
	}, nil
}

func serializeIntegrations(in []core.Integration) []*pb.IntegrationDefinition {
	out := make([]*pb.IntegrationDefinition, len(in))
	for i, integration := range in {
		configFields := integration.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.IntegrationDefinition{
			Name:                     integration.Name(),
			Label:                    integration.Label(),
			Icon:                     integration.Icon(),
			Description:              integration.Description(),
			InstallationInstructions: integration.InstallationInstructions(),
			Configuration:            configuration,
			Components:               actions.SerializeComponents(integration.Components()),
			Triggers:                 actions.SerializeTriggers(integration.Triggers()),
		}
	}
	return out
}
