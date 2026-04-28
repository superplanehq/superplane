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
			Name:             integration.Name(),
			Label:            integration.Label(),
			Icon:             integration.Icon(),
			Description:      integration.Description(),
			Instructions:     integration.Instructions(),
			Configuration:    configuration,
			Actions:          actions.SerializeActions(integration.Actions()),
			Triggers:         actions.SerializeTriggers(integration.Triggers()),
			Capabilities:     serializeCapabilities(registry, integration),
			CapabilityGroups: serializeCapabilityGroups(registry, integration),
		}
	}
	return out
}

func serializeCapabilityGroups(registry *registry.Registry, integration core.Integration) []*pb.CapabilityGroup {
	setupProvider, err := registry.GetSetupProvider(integration.Name())
	if err != nil {
		return []*pb.CapabilityGroup{}
	}

	groups := []*pb.CapabilityGroup{}
	for _, group := range setupProvider.CapabilityGroups() {
		g := &pb.CapabilityGroup{
			Label:        group.Label,
			Capabilities: []string{},
		}

		for _, capability := range group.Capabilities {
			g.Capabilities = append(g.Capabilities, capability.Name)
		}

		groups = append(groups, g)
	}

	return groups
}

func serializeCapabilities(registry *registry.Registry, integration core.Integration) []*pb.CapabilityDefinition {
	setupProvider, err := registry.GetSetupProvider(integration.Name())
	if err != nil {
		return []*pb.CapabilityDefinition{}
	}

	capabilityGroups := setupProvider.CapabilityGroups()

	out := []*pb.CapabilityDefinition{}
	for _, group := range capabilityGroups {
		for _, capability := range group.Capabilities {
			capabilityDef := &pb.CapabilityDefinition{
				Type:           actions.CapabilityTypeToProto(string(capability.Type)),
				Name:           capability.Name,
				Label:          capability.Label,
				Description:    capability.Description,
				Configuration:  []*configpb.Field{},
				OutputChannels: []*actionpb.OutputChannel{},
			}

			for _, field := range capability.Configuration {
				capabilityDef.Configuration = append(capabilityDef.Configuration, actions.ConfigurationFieldToProto(field))
			}

			for _, channel := range capability.OutputChannels {
				capabilityDef.OutputChannels = append(capabilityDef.OutputChannels, &actionpb.OutputChannel{
					Name:        channel.Name,
					Label:       channel.Label,
					Description: channel.Description,
				})
			}

			out = append(out, capabilityDef)
		}
	}

	return out
}
