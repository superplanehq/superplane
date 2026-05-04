package integrations

import (
	"context"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/impl"
)

func TestListIntegrationsIncludesDefaultRunTitleForTriggerCapabilities(t *testing.T) {
	trigger := impl.NewDummyTrigger(impl.DummyTriggerOptions{
		Name:            "dummy.onPush",
		DefaultRunTitle: "{{ root().data.head_commit.message }}",
	})

	integration := impl.NewDummyIntegration(impl.DummyIntegrationOptions{
		Triggers: []core.Trigger{trigger},
	})

	setupProvider := impl.NewDummyIntegrationSetupProvider(impl.DummyIntegrationSetupProviderOptions{
		CapabilityGroups: []core.CapabilityGroup{
			{
				Capabilities: []core.Capability{
					{
						Type: core.IntegrationCapabilityTypeTrigger,
						Name: "dummy.onPush",
					},
				},
			},
		},
	})

	response, err := ListIntegrations(context.Background(), &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": integration,
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{
			"dummy": setupProvider,
		},
	})

	require.NoError(t, err)
	require.Len(t, response.Integrations, 1)
	require.Len(t, response.Integrations[0].Capabilities, 1)
	require.Equal(t, "{{ root().data.head_commit.message }}", response.Integrations[0].Capabilities[0].DefaultRunTitle)
}
