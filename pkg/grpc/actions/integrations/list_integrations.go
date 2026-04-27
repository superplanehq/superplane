package integrations

import (
	"context"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	actionpb "github.com/superplanehq/superplane/pkg/protos/actions"
	configpb "github.com/superplanehq/superplane/pkg/protos/configuration"
	pb "github.com/superplanehq/superplane/pkg/protos/integrations"
	"github.com/superplanehq/superplane/pkg/registry"
)

func ListIntegrations(ctx context.Context, registry *registry.Registry) (*pb.ListIntegrationsResponse, error) {
	integrations := registry.ListIntegrations()

	return &pb.ListIntegrationsResponse{
		Integrations: serializeIntegrations(registry, integrations),
	}, nil
}

func serializeIntegrations(registry *registry.Registry, in []core.Integration) []*pb.IntegrationDefinition {
	out := make([]*pb.IntegrationDefinition, len(in))
	for i, integration := range in {
		configFields := integration.Configuration()
		configuration := make([]*configpb.Field, len(configFields))
		for j, field := range configFields {
			configuration[j] = actions.ConfigurationFieldToProto(field)
		}

		out[i] = &pb.IntegrationDefinition{
			Name:          integration.Name(),
			Label:         integration.Label(),
			Icon:          integration.Icon(),
			Description:   integration.Description(),
			Instructions:  integration.Instructions(),
			Configuration: configuration,
			Actions:       actions.SerializeActions(integration.Actions()),
			Triggers:      actions.SerializeTriggers(integration.Triggers()),
			Capabilities:  serializeCapabilities(registry, integration),
		}
	}
	return out
}

func serializeCapabilities(registry *registry.Registry, integration core.Integration) []*pb.CapabilityDefinition {
	setupProvider, err := registry.GetSetupProvider(integration.Name())
	if err != nil {
		return []*pb.CapabilityDefinition{}
	}

	capabilities := setupProvider.Capabilities()
	out := make([]*pb.CapabilityDefinition, len(capabilities))
	for i, capability := range capabilities {
		out[i] = &pb.CapabilityDefinition{
			Type:           actions.CapabilityTypeToProto(string(capability.Type)),
			Name:           capability.Name,
			Label:          capability.Label,
			Description:    capability.Description,
			Configuration:  []*configpb.Field{},
			OutputChannels: []*actionpb.OutputChannel{},
		}

		for _, field := range capability.Configuration {
			out[i].Configuration = append(out[i].Configuration, actions.ConfigurationFieldToProto(field))
		}

		for _, channel := range capability.OutputChannels {
			out[i].OutputChannels = append(out[i].OutputChannels, &actionpb.OutputChannel{
				Name:        channel.Name,
				Label:       channel.Label,
				Description: channel.Description,
			})
		}
	}

	return out
}
