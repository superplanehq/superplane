package jira

import (
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__SetupProvider__CapabilityGroups(t *testing.T) {
	s := &SetupProvider{}
	groups := s.CapabilityGroups()

	require.Len(t, groups, 1)
	names := map[string]bool{}
	for _, c := range groups[0].Capabilities {
		names[c.Name] = true
	}
	assert.True(t, names["jira.createIssue"])
	assert.True(t, names["jira.onIssue"])
}

func Test__SetupProvider__FirstStep(t *testing.T) {
	s := &SetupProvider{}
	step := s.FirstStep(core.SetupStepContext{
		BaseURL:       "https://superplane.example.com",
		IntegrationID: uuid.MustParse("11111111-1111-1111-1111-111111111111"),
	})

	assert.Equal(t, core.SetupStepTypeInputs, step.Type)
	assert.Equal(t, SetupStepEnterAppCredentials, step.Name)
	assert.Contains(t, step.Instructions, "https://superplane.example.com/api/v1/integrations/11111111-1111-1111-1111-111111111111/redirect")
	require.Len(t, step.Inputs, 2)
}

func Test__SetupProvider__OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}

	t.Run("enterAppCredentials requires a client id", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		_, err := s.OnStepSubmit(core.SetupStepContext{
			BaseURL:       "https://superplane.example.com",
			IntegrationID: uuid.New(),
			Properties:    intCtx.Properties(),
			Secrets:       intCtx.Secrets(),
			Step: core.StepInfo{
				Name:   SetupStepEnterAppCredentials,
				Inputs: map[string]any{PropertyClientID: "", SecretOAuthClientSecret: "secret"},
			},
		})
		require.ErrorContains(t, err, "client id is required")
	})

	t.Run("enterAppCredentials requires a client secret", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		_, err := s.OnStepSubmit(core.SetupStepContext{
			BaseURL:       "https://superplane.example.com",
			IntegrationID: uuid.New(),
			Properties:    intCtx.Properties(),
			Secrets:       intCtx.Secrets(),
			Step: core.StepInfo{
				Name:   SetupStepEnterAppCredentials,
				Inputs: map[string]any{PropertyClientID: "client-1", SecretOAuthClientSecret: ""},
			},
		})
		require.ErrorContains(t, err, "client secret is required")
	})

	t.Run("enterAppCredentials stores credentials and returns a redirect step", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		integrationID := uuid.New()
		step, err := s.OnStepSubmit(core.SetupStepContext{
			BaseURL:       "https://superplane.example.com",
			IntegrationID: integrationID,
			Properties:    intCtx.Properties(),
			Secrets:       intCtx.Secrets(),
			Step: core.StepInfo{
				Name:   SetupStepEnterAppCredentials,
				Inputs: map[string]any{PropertyClientID: "client-1", SecretOAuthClientSecret: "secret-1"},
			},
		})
		require.NoError(t, err)
		require.NotNil(t, step)
		assert.Equal(t, core.SetupStepTypeRedirectPrompt, step.Type)
		assert.Equal(t, SetupStepAuthorize, step.Name)
		require.NotNil(t, step.RedirectPrompt)
		assert.Equal(t, "GET", step.RedirectPrompt.Method)
		assert.Contains(t, step.RedirectPrompt.URL, "https://auth.atlassian.com/authorize?")
		assert.Contains(t, step.RedirectPrompt.URL, "client_id=client-1")

		clientID, err := intCtx.Properties().GetString(PropertyClientID)
		require.NoError(t, err)
		assert.Equal(t, "client-1", clientID)

		clientSecret, err := intCtx.Secrets().Get(SecretOAuthClientSecret)
		require.NoError(t, err)
		assert.Equal(t, "secret-1", clientSecret)

		state, err := intCtx.Properties().GetString(PropertyOAuthState)
		require.NoError(t, err)
		assert.NotEmpty(t, state)
		assert.Contains(t, step.RedirectPrompt.URL, "state=")
	})

	t.Run("authorize step is not directly submitted", func(t *testing.T) {
		step, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: SetupStepAuthorize},
		})
		require.NoError(t, err)
		require.Nil(t, step)
	})

	t.Run("unknown step returns an error", func(t *testing.T) {
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{Name: "bogus"},
		})
		require.ErrorContains(t, err, "unknown step")
	})
}

func Test__SetupProvider__OnStepRevert(t *testing.T) {
	s := &SetupProvider{}

	t.Run("enterAppCredentials clears stored properties", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.Properties().Create(core.IntegrationPropertyDefinition{Name: PropertyClientID, Value: "client-1"}))
		require.NoError(t, intCtx.Properties().Create(core.IntegrationPropertyDefinition{Name: PropertyOAuthState, Value: "state-1"}))

		err := s.OnStepRevert(core.SetupStepContext{
			Properties: intCtx.Properties(),
			Step:       core.StepInfo{Name: SetupStepEnterAppCredentials},
		})
		require.NoError(t, err)

		_, err = intCtx.Properties().GetString(PropertyClientID)
		require.Error(t, err)
	})

	t.Run("unknown step returns an error", func(t *testing.T) {
		err := s.OnStepRevert(core.SetupStepContext{Step: core.StepInfo{Name: "bogus"}})
		require.ErrorContains(t, err, "unknown step")
	})
}

func Test__SetupProvider__OnCapabilityUpdate(t *testing.T) {
	s := &SetupProvider{}
	log := logger.DiscardLogger()

	t.Run("returns nil when nothing is requested", func(t *testing.T) {
		step, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       log,
			Capabilities: &contexts.CapabilityContext{},
		})
		require.NoError(t, err)
		require.Nil(t, step)
	})

	t.Run("enables whatever is requested unconditionally", func(t *testing.T) {
		capabilities := &contexts.CapabilityContext{}
		step, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: log,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"jira.onIssue"},
			},
			Capabilities: capabilities,
		})
		require.NoError(t, err)
		require.Nil(t, step)
		assert.Contains(t, capabilities.EnabledCapabilities, "jira.onIssue")
	})
}

func Test__SetupProvider__OnPropertyAndSecretUpdate(t *testing.T) {
	s := &SetupProvider{}

	_, err := s.OnPropertyUpdate(core.PropertyUpdateContext{})
	require.Error(t, err)

	_, err = s.OnSecretUpdate(core.SecretUpdateContext{})
	require.Error(t, err)
}
