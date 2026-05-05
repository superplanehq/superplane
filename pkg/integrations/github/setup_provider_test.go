package github

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__GitHub__SetupProvider__OnCapabilityUpdate(t *testing.T) {
	g := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("returns nil when there are no changes", func(t *testing.T) {
		step, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       log,
			Changes:      nil,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.NoError(t, err)
		require.Nil(t, step)
	})

	t.Run("returns error when requested capabilities list is empty", func(t *testing.T) {
		_, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateEnabled: {"github.getIssue"},
			},
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no requested capabilities")
	})

	t.Run("returns error when authentication method property is missing", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		_, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"github.runWorkflow"},
			},
			Properties:   props,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "authentication method")
	})

	t.Run("returns error for invalid authentication method", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyAuthMethod,
			Value: "not-a-real-method",
		}))

		_, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"github.runWorkflow"},
			},
			Properties:   props,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid authentication method")
	})

	t.Run("enables capabilities when no new permissions are required", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyAuthMethod,
			Value: common.AuthMethodPAT,
		}))

		capCtx := &contexts.CapabilityContext{
			EnabledCapabilities: []string{"github.getIssue"},
		}

		step, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"github.onIssue"},
			},
			Properties:   props,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.Nil(t, step)
		assert.Contains(t, capCtx.EnabledCapabilities, "github.getIssue")
		assert.Contains(t, capCtx.EnabledCapabilities, "github.onIssue")
	})

	t.Run("PAT requests permission update step when new permissions are needed", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyAuthMethod,
			Value: common.AuthMethodPAT,
		}))
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwner,
			Value: "acme-corp",
		}))

		capCtx := &contexts.CapabilityContext{}

		step, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"github.runWorkflow"},
			},
			Properties:   props,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, step)
		assert.Equal(t, SetupStepUpdatePATPermissions, step.Name)
		assert.Equal(t, core.SetupStepTypeInputs, step.Type)
		assert.Contains(t, step.Instructions, "acme-corp")
		assert.Contains(t, capCtx.RequestedCapabilties, "github.runWorkflow")
	})

	t.Run("GitHub App requests permission update step when new permissions are needed", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.CreateMany([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyAuthMethod, Value: common.AuthMethodGitHubApp},
			{Name: common.PropertyOwner, Value: "acme-corp"},
			{Name: common.PropertyOwnerType, Value: common.OwnerTypeUser},
			{Name: common.PropertyAppSlug, Value: "superplane-dev"},
		}))

		capCtx := &contexts.CapabilityContext{}

		step, err := g.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"github.runWorkflow"},
			},
			Properties:   props,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, step)
		assert.Equal(t, SetupStepUpdateAppPermissions, step.Name)
		assert.Contains(t, step.Instructions, "superplane-dev")
		assert.Contains(t, step.Instructions, "settings/apps/superplane-dev/permissions")
		assert.Contains(t, capCtx.RequestedCapabilties, "github.runWorkflow")
	})
}

func Test__GitHub__SetupProvider__OnPropertyUpdate(t *testing.T) {
	g := &SetupProvider{}
	_, err := g.OnPropertyUpdate(core.PropertyUpdateContext{})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no property updates are supported")
}

func Test__GitHub__SetupProvider__OnSecretUpdate(t *testing.T) {
	g := &SetupProvider{}
	log := logger.DiscardLogger()
	props := contexts.NewIntegrationPropertyStorage()
	intCtx := &contexts.IntegrationContext{}
	secrets := intCtx.Secrets()

	t.Run("unknown secret", func(t *testing.T) {
		_, err := g.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     log,
			SecretName: "other",
			Value:      "x",
			Properties: props,
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})

	t.Run("PAT value is required", func(t *testing.T) {
		_, err := g.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     log,
			SecretName: common.SecretPAT,
			Value:      "   ",
			Properties: props,
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})
}

func Test__GitHub__SetupProvider__OnStepSubmit(t *testing.T) {
	g := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		_, err := g.OnStepSubmit(core.SetupStepContext{
			Step:   core.StepInfo{Name: "nope"},
			Logger: log,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("selectOwner validation", func(t *testing.T) {
		_, err := g.OnStepSubmit(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepSelectOwner, Inputs: "not-a-map"},
			Logger:     log,
			Properties: contexts.NewIntegrationPropertyStorage(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectOwner, Inputs: map[string]any{
				common.PropertyOwnerType: 1,
				common.PropertyOwner:     "x",
			}},
			Logger:     log,
			Properties: contexts.NewIntegrationPropertyStorage(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid owner type")

		_, err = g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectOwner, Inputs: map[string]any{
				common.PropertyOwnerType: common.OwnerTypeUser,
				common.PropertyOwner:     "",
			}},
			Logger:     log,
			Properties: contexts.NewIntegrationPropertyStorage(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "owner is required")
	})

	t.Run("selectOwner success moves to capability selection", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectOwner, Inputs: map[string]any{
				common.PropertyOwnerType: common.OwnerTypeUser,
				common.PropertyOwner:     "superplanehq",
			}},
			Logger:     log,
			Properties: props,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepCapabilitySelection, next.Name)
		assert.Equal(t, core.SetupStepTypeCapabilitySelection, next.Type)

		ownerType, err := props.GetString(common.PropertyOwnerType)
		require.NoError(t, err)
		assert.Equal(t, common.OwnerTypeUser, ownerType)
		owner, err := props.GetString(common.PropertyOwner)
		require.NoError(t, err)
		assert.Equal(t, "superplanehq", owner)
	})

	t.Run("capabilitySelection leads to auth method step", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwnerType,
			Value: common.OwnerTypeUser,
		}))
		capCtx := &contexts.CapabilityContext{}

		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:         SetupStepCapabilitySelection,
				Capabilities: []string{"github.getIssue"},
			},
			Logger:       log,
			Properties:   props,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepSelectAuthMethod, next.Name)
		assert.Contains(t, capCtx.RequestedCapabilties, "github.getIssue")
	})

	t.Run("selectAuthMethod validation", func(t *testing.T) {
		_, err := g.OnStepSubmit(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepSelectAuthMethod, Inputs: 42},
			Logger:     log,
			Properties: contexts.NewIntegrationPropertyStorage(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectAuthMethod, Inputs: map[string]any{
				common.PropertyAuthMethod: "nope",
			}},
			Logger:     log,
			Properties: contexts.NewIntegrationPropertyStorage(),
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid authentication method")
	})

	t.Run("selectAuthMethod PAT produces enter PAT step", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwner,
			Value: "my-org",
		}))
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"github.getRepositoryPermission"},
		}

		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectAuthMethod, Inputs: map[string]any{
				common.PropertyAuthMethod: common.AuthMethodPAT,
			}},
			Logger:       log,
			Properties:   props,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepEnterPAT, next.Name)
		authMethod, err := props.GetString(common.PropertyAuthMethod)
		require.NoError(t, err)
		assert.Equal(t, common.AuthMethodPAT, authMethod)
		assert.Contains(t, next.Instructions, "my-org")
	})

	t.Run("selectAuthMethod GitHub App produces redirect step for user owner", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwnerType,
			Value: common.OwnerTypeUser,
		}))
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwner,
			Value: "devuser",
		}))
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"github.getRepositoryPermission"},
		}
		integrationID := uuid.MustParse("6ba7b810-9dad-11d1-80b4-00c04fd430c8")

		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectAuthMethod, Inputs: map[string]any{
				common.PropertyAuthMethod: common.AuthMethodGitHubApp,
			}},
			Logger:          log,
			Properties:      props,
			Capabilities:    capCtx,
			IntegrationID:   integrationID,
			BaseURL:         "https://app.superplane.test",
			WebhooksBaseURL: "https://hooks.superplane.test",
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepCreateApp, next.Name)
		assert.Equal(t, core.SetupStepTypeRedirectPrompt, next.Type)
		require.NotNil(t, next.RedirectPrompt)
		assert.Equal(t, "POST", next.RedirectPrompt.Method)
		assert.Equal(t, "https://github.com/settings/apps/new", next.RedirectPrompt.URL)
		assert.Contains(t, next.RedirectPrompt.FormData["manifest"], "SuperPlane")
		assert.Contains(t, next.RedirectPrompt.FormData["manifest"], integrationID.String())
		_, err = props.GetString(common.PropertyAppState)
		require.NoError(t, err)
	})

	t.Run("selectAuthMethod GitHub App uses organization app URL when owner type is org", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwnerType,
			Value: common.OwnerTypeOrganization,
		}))
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyOwner,
			Value: "bigcorp",
		}))
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"github.getRepositoryPermission"},
		}

		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepSelectAuthMethod, Inputs: map[string]any{
				common.PropertyAuthMethod: common.AuthMethodGitHubApp,
			}},
			Logger:          log,
			Properties:      props,
			Capabilities:    capCtx,
			IntegrationID:   uuid.New(),
			BaseURL:         "https://app.superplane.test",
			WebhooksBaseURL: "https://hooks.superplane.test",
		})
		require.NoError(t, err)
		require.NotNil(t, next.RedirectPrompt)
		assert.Equal(t, "https://github.com/organizations/bigcorp/settings/apps/new", next.RedirectPrompt.URL)
	})

	t.Run("enterPAT validation", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		intCtx := &contexts.IntegrationContext{}

		_, err := g.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterPAT, Inputs: "not-map"},
			Logger:       log,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = g.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterPAT, Inputs: map[string]any{common.SecretPAT: ""}},
			Logger:       log,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "personal access token is required")
	})

	t.Run("update PAT permissions submit enables requested capabilities", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"github.getIssue"},
		}
		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepUpdatePATPermissions},
			Logger:       log,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.Nil(t, next)
		assert.Contains(t, capCtx.EnabledCapabilities, "github.getIssue")
	})

	t.Run("update app permissions submit leads to accept step", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  common.PropertyAppInstallationID,
			Value: "12345",
		}))

		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepUpdateAppPermissions},
			Logger:     log,
			Properties: props,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepAcceptAppPermissionUpdate, next.Name)
		assert.Contains(t, next.Instructions, "12345")
		assert.Contains(t, next.Instructions, "permissions/update")
	})

	t.Run("accept app permission update enables requested capabilities", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"github.runWorkflow"},
		}
		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepAcceptAppPermissionUpdate},
			Logger:       log,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.Nil(t, next)
		assert.Contains(t, capCtx.EnabledCapabilities, "github.runWorkflow")
	})

	t.Run("setup app step submit is a no-op", func(t *testing.T) {
		next, err := g.OnStepSubmit(core.SetupStepContext{
			Step:   core.StepInfo{Name: SetupStepCreateApp},
			Logger: log,
		})
		require.NoError(t, err)
		require.Nil(t, next)
	})
}

func Test__GitHub__SetupProvider__OnStepRevert(t *testing.T) {
	g := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		err := g.OnStepRevert(core.SetupStepContext{
			Step:   core.StepInfo{Name: "unknown"},
			Logger: log,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("selectOwner clears owner properties", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.CreateMany([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyOwnerType, Value: common.OwnerTypeUser},
			{Name: common.PropertyOwner, Value: "acme"},
		}))

		require.NoError(t, g.OnStepRevert(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepSelectOwner},
			Logger:     log,
			Properties: props,
		}))
		_, err := props.GetString(common.PropertyOwner)
		require.Error(t, err)
	})

	t.Run("capabilitySelection clears capability state", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties:  []string{"a"},
			EnabledCapabilities:   []string{"b"},
			AvailableCapabilities: []string{"c"},
		}
		require.NoError(t, g.OnStepRevert(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepCapabilitySelection},
			Logger:       log,
			Capabilities: capCtx,
		}))
		assert.Empty(t, capCtx.RequestedCapabilties)
		assert.Empty(t, capCtx.EnabledCapabilities)
	})

	t.Run("selectAuthMethod clears auth properties", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.CreateMany([]core.IntegrationPropertyDefinition{
			{Name: common.PropertyAuthMethod, Value: common.AuthMethodPAT},
			{Name: common.PropertyAppState, Value: "state-token"},
		}))

		require.NoError(t, g.OnStepRevert(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepSelectAuthMethod},
			Logger:     log,
			Properties: props,
		}))
		_, err := props.GetString(common.PropertyAuthMethod)
		require.Error(t, err)
	})

	t.Run("enterPAT clears secret", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.SetSecret(common.SecretPAT, []byte("sek")))

		require.NoError(t, g.OnStepRevert(core.SetupStepContext{
			Step:    core.StepInfo{Name: SetupStepEnterPAT},
			Logger:  log,
			Secrets: intCtx.Secrets(),
		}))
		_, err := intCtx.Secrets().Get(common.SecretPAT)
		require.Error(t, err)
	})

	t.Run("setup app revert is a no-op", func(t *testing.T) {
		require.NoError(t, g.OnStepRevert(core.SetupStepContext{
			Step:   core.StepInfo{Name: SetupStepCreateApp},
			Logger: log,
		}))
	})
}
