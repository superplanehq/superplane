package integrations

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/require"
	appconfig "github.com/superplanehq/superplane/pkg/config"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/features"
	grpcerrors "github.com/superplanehq/superplane/pkg/grpc/errors"
	ghub "github.com/superplanehq/superplane/pkg/integrations/github"
	"github.com/superplanehq/superplane/pkg/models"
	"github.com/superplanehq/superplane/pkg/registry"
	"github.com/superplanehq/superplane/test/support"
	"github.com/superplanehq/superplane/test/support/impl"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
)

func contextWithOrganizationID(orgID string) context.Context {
	return metadata.NewIncomingContext(context.Background(), metadata.Pairs(
		"x-organization-id", orgID,
	))
}

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
func (a *testAction) Configuration() []configuration.Field    { return nil }
func (a *testAction) Setup(core.SetupContext) error           { return nil }
func (a *testAction) Execute(core.ExecutionContext) error     { return nil }
func (a *testAction) Hooks() []core.Hook                      { return nil }
func (a *testAction) HandleHook(core.ActionHookContext) error { return nil }
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

	resp, err := ListIntegrations(contextWithOrganizationID(uuid.New().String()), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 2)

	require.Equal(t, "123", resp.Integrations[0].Capabilities[0].GetExampleOutput().GetFields()["id"].GetStringValue())
	require.Equal(t, "created", resp.Integrations[0].Capabilities[1].GetExampleData().GetFields()["event"].GetStringValue())
}

func TestListIntegrationsAddsGlobalFieldsToLegacyTriggerCapabilities(t *testing.T) {
	r := &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": impl.NewDummyIntegration(impl.DummyIntegrationOptions{
				Triggers: []core.Trigger{
					&testTrigger{name: "github.onPush"},
				},
			}),
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{},
	}

	resp, err := ListIntegrations(contextWithOrganizationID(uuid.New().String()), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 1)

	configuration := resp.Integrations[0].Capabilities[0].Configuration
	require.Len(t, configuration, 1)
	require.Equal(t, "customName", configuration[0].Name)
	require.Equal(t, "Run title", configuration[0].Label)
	require.Equal(t, "{{ root().data.head_commit.message }} - {{ root().data.head_commit.id[:7] }}", configuration[0].GetDefaultValue())
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

	resp, err := ListIntegrations(contextWithOrganizationID(uuid.New().String()), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 2)

	require.Equal(t, "ok", resp.Integrations[0].Capabilities[0].GetExampleOutput().GetFields()["status"].GetStringValue())
	require.Equal(t, "push", resp.Integrations[0].Capabilities[1].GetExampleData().GetFields()["kind"].GetStringValue())
}

func TestListIntegrationsAddsGlobalFieldsToSetupProviderTriggerCapabilities(t *testing.T) {
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
								Type: core.IntegrationCapabilityTypeTrigger,
								Name: "github.onPush",
								Configuration: []configuration.Field{
									{
										Name:     "repository",
										Label:    "Repository",
										Type:     configuration.FieldTypeString,
										Required: true,
									},
								},
							},
						},
					},
				},
			},
		},
	}

	resp, err := ListIntegrations(contextWithOrganizationID(uuid.New().String()), r)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.Len(t, resp.Integrations[0].Capabilities, 1)

	configuration := resp.Integrations[0].Capabilities[0].Configuration
	require.Len(t, configuration, 2)
	require.Equal(t, "repository", configuration[0].Name)
	require.Equal(t, "customName", configuration[1].Name)
	require.Equal(t, "Run title", configuration[1].Label)
	require.Equal(t, "{{ root().data.head_commit.message }} - {{ root().data.head_commit.id[:7] }}", configuration[1].GetDefaultValue())
}

func TestListIntegrationsLegacySetupOnlyRespectsExperimentalFeature(t *testing.T) {
	setup := support.Setup(t)
	reg := &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": impl.NewDummyIntegration(impl.DummyIntegrationOptions{}),
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{
			"dummy": &testSetupProvider{},
		},
	}

	ctx := contextWithOrganizationID(setup.Organization.ID.String())

	resp, err := ListIntegrations(ctx, reg)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.True(t, resp.Integrations[0].LegacySetupOnly)

	require.NoError(t, models.EnableExperimentalFeature(setup.Organization.ID, features.FeatureNewIntegrationSetupFlow))

	resp, err = ListIntegrations(ctx, reg)
	require.NoError(t, err)
	require.Len(t, resp.Integrations, 1)
	require.False(t, resp.Integrations[0].LegacySetupOnly)
}

func TestListIntegrationsHostedGitHubAppInstall(t *testing.T) {
	setup := support.Setup(t)
	reg := &registry.Registry{
		Integrations: map[string]core.Integration{
			"github": &ghub.GitHub{},
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{
			"github": &ghub.SetupProvider{},
		},
	}
	ctx := contextWithOrganizationID(setup.Organization.ID.String())
	require.NoError(t, models.EnableExperimentalFeature(setup.Organization.ID, features.FeatureNewIntegrationSetupFlow))
	require.NoError(t, models.EnableExperimentalFeature(setup.Organization.ID, features.FeatureFactories))

	t.Run("false without hosted app env", func(t *testing.T) {
		t.Setenv(appconfig.EnvGitHubAppID, "")
		t.Setenv(appconfig.EnvGitHubAppSlug, "")
		t.Setenv(appconfig.EnvGitHubAppPrivateKey, "")
		t.Setenv(appconfig.EnvGitHubAppWebhookSecret, "")

		resp, err := ListIntegrations(ctx, reg)
		require.NoError(t, err)
		require.Len(t, resp.Integrations, 1)
		require.False(t, resp.Integrations[0].HostedAppInstall)
		require.False(t, resp.Integrations[0].LegacySetupOnly)
	})

	t.Run("true when factories and env are set", func(t *testing.T) {
		t.Setenv(appconfig.EnvGitHubAppID, "99")
		t.Setenv(appconfig.EnvGitHubAppSlug, "superplane")
		t.Setenv(appconfig.EnvGitHubAppPrivateKey, "pem")
		t.Setenv(appconfig.EnvGitHubAppWebhookSecret, "whsec")

		resp, err := ListIntegrations(ctx, reg)
		require.NoError(t, err)
		require.Len(t, resp.Integrations, 1)
		require.True(t, resp.Integrations[0].HostedAppInstall)
		require.False(t, resp.Integrations[0].LegacySetupOnly)
	})

	t.Run("hosted install does not enable the setup wizard when the feature is off", func(t *testing.T) {
		org, err := models.CreateOrganization(support.RandomName("org"), "")
		require.NoError(t, err)
		require.NoError(t, models.EnableExperimentalFeature(org.ID, features.FeatureFactories))
		t.Setenv(appconfig.EnvGitHubAppID, "99")
		t.Setenv(appconfig.EnvGitHubAppSlug, "superplane")
		t.Setenv(appconfig.EnvGitHubAppPrivateKey, "pem")
		t.Setenv(appconfig.EnvGitHubAppWebhookSecret, "whsec")

		resp, err := ListIntegrations(contextWithOrganizationID(org.ID.String()), reg)
		require.NoError(t, err)
		require.Len(t, resp.Integrations, 1)
		require.True(t, resp.Integrations[0].HostedAppInstall)
		require.True(t, resp.Integrations[0].LegacySetupOnly)
	})
}

func TestListIntegrationsRequiresOrganization(t *testing.T) {
	reg := &registry.Registry{
		Integrations: map[string]core.Integration{
			"dummy": impl.NewDummyIntegration(impl.DummyIntegrationOptions{}),
		},
		SetupProviders: map[string]core.IntegrationSetupProvider{},
	}

	t.Run("missing organization header", func(t *testing.T) {
		resp, err := ListIntegrations(context.Background(), reg)
		require.Nil(t, resp)
		require.Equal(t, codes.Unauthenticated, grpcerrors.Code(err))
	})

	t.Run("invalid organization id", func(t *testing.T) {
		resp, err := ListIntegrations(contextWithOrganizationID("not-a-uuid"), reg)
		require.Nil(t, resp)
		require.Equal(t, codes.InvalidArgument, grpcerrors.Code(err))
	})
}
