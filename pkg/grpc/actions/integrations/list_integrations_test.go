package integrations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support/impl"
)

type testAction struct {
	name    string
	example map[string]any
}

func (a *testAction) Name() string                  { return a.name }
func (a *testAction) Label() string                 { return a.name }
func (a *testAction) Description() string           { return a.name }
func (a *testAction) Documentation() string         { return "" }
func (a *testAction) Icon() string                  { return "" }
func (a *testAction) Color() string                 { return "" }
func (a *testAction) ExampleOutput() map[string]any { return a.example }
func (a *testAction) OutputChannels(any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}
func (a *testAction) Configuration() []configuration.Field                          { return nil }
func (a *testAction) Setup(core.SetupContext) error                                 { return nil }
func (a *testAction) ProcessQueueItem(core.ProcessQueueContext) (*uuid.UUID, error) { return nil, nil }
func (a *testAction) Execute(core.ExecutionContext) error                           { return nil }
func (a *testAction) Hooks() []core.Hook                                            { return nil }
func (a *testAction) HandleHook(core.ActionHookContext) error                       { return nil }
func (a *testAction) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (a *testAction) Cancel(core.ExecutionContext) error { return nil }
func (a *testAction) Cleanup(core.SetupContext) error    { return nil }

type testTrigger struct {
	name    string
	example map[string]any
}

func (t *testTrigger) Name() string                         { return t.name }
func (t *testTrigger) Label() string                        { return t.name }
func (t *testTrigger) Description() string                  { return t.name }
func (t *testTrigger) Documentation() string                { return "" }
func (t *testTrigger) Icon() string                         { return "" }
func (t *testTrigger) Color() string                        { return "" }
func (t *testTrigger) ExampleData() map[string]any          { return t.example }
func (t *testTrigger) Configuration() []configuration.Field { return nil }
func (t *testTrigger) HandleWebhook(core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}
func (t *testTrigger) Setup(core.TriggerContext) error                            { return nil }
func (t *testTrigger) Hooks() []core.Hook                                         { return nil }
func (t *testTrigger) HandleHook(core.TriggerHookContext) (map[string]any, error) { return nil, nil }
func (t *testTrigger) Cleanup(core.TriggerContext) error                          { return nil }

type testSetupProvider struct {
	groups []core.CapabilityGroup
}

func (p *testSetupProvider) CapabilityGroups() []core.CapabilityGroup       { return p.groups }
func (p *testSetupProvider) FirstStep(core.SetupStepContext) core.SetupStep { return core.SetupStep{} }
func (p *testSetupProvider) OnStepSubmit(core.SetupStepContext) (*core.SetupStep, error) {
	return nil, nil
}
func (p *testSetupProvider) OnStepRevert(core.SetupStepContext) error { return nil }
func (p *testSetupProvider) OnPropertyUpdate(core.PropertyUpdateContext) (*core.SetupStep, error) {
	return nil, nil
}
func (p *testSetupProvider) OnSecretUpdate(core.SecretUpdateContext) (*core.SetupStep, error) {
	return nil, nil
}
func (p *testSetupProvider) OnCapabilityUpdate(core.CapabilityUpdateContext) (*core.SetupStep, error) {
	return nil, nil
}

func TestListIntegrationsIncludesExamplePayloadsForLegacyCapabilities(t *testing.T) {
	r := &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": impl.NewDummyIntegration(impl.DummyIntegrationOptions{
				Actions: []core.Action{
					&testAction{
						name:    "dummy.action",
						example: map[string]any{"id": "123"},
					},
				},
				Triggers: []core.Trigger{
					&testTrigger{
						name:    "dummy.trigger",
						example: map[string]any{"event": "created"},
					},
				},
			}),
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{},
	}

	resp, err := ListIntegrations(context.Background(), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 2)

	require.Equal(t, "123", resp.Integrations[0].Capabilities[0].GetExampleOutput().GetFields()["id"].GetStringValue())
	require.Equal(t, "created", resp.Integrations[0].Capabilities[1].GetExampleData().GetFields()["event"].GetStringValue())
}

func TestListIntegrationsIncludesExamplePayloadsForSetupProviderCapabilities(t *testing.T) {
	r := &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": impl.NewDummyIntegration(impl.DummyIntegrationOptions{}),
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{
			"dummy": &testSetupProvider{
				groups: []core.CapabilityGroup{
					{
						Label: "Test",
						Capabilities: []core.Capability{
							{
								Type:          core.IntegrationCapabilityTypeAction,
								Name:          "dummy.action",
								ExampleOutput: map[string]any{"status": "ok"},
							},
							{
								Type:        core.IntegrationCapabilityTypeTrigger,
								Name:        "dummy.trigger",
								ExampleData: map[string]any{"kind": "push"},
							},
						},
					},
				},
			},
		},
	}

	resp, err := ListIntegrations(context.Background(), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 2)

	require.Equal(t, "ok", resp.Integrations[0].Capabilities[0].GetExampleOutput().GetFields()["status"].GetStringValue())
	require.Equal(t, "push", resp.Integrations[0].Capabilities[1].GetExampleData().GetFields()["kind"].GetStringValue())
}
