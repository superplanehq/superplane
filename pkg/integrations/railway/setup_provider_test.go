package railway

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__Railway__SetupProvider__OnCapabilityUpdate(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("returns error when no requested capabilities entry", func(t *testing.T) {
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger:       logger,
			Changes:      map[core.IntegrationCapabilityState][]string{},
			Capabilities: &contexts.CapabilityContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "no requested capabilities")
	})

	t.Run("delegates Enable for requested capability names", func(t *testing.T) {
		localCap := &contexts.CapabilityContext{}
		_, err := s.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: logger,
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"railway.triggerDeploy"},
			},
			Capabilities: localCap,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"railway.triggerDeploy"}, localCap.EnabledCapabilities)
	})
}

func Test__Railway__SetupProvider__OnSecretUpdate(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	intCtx := &contexts.IntegrationContext{}
	props := intCtx.Properties()
	secrets := intCtx.Secrets()

	t.Run("unknown secret", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: "other",
			Value:      "x",
			HTTP:       &contexts.HTTPContext{},
			Properties: props,
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})

	t.Run("api token required", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: "apiToken",
			Value:      "   ",
			HTTP:       &contexts.HTTPContext{},
			Properties: props,
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})

	t.Run("verification fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("GraphQL error")),
				},
			},
		}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: "apiToken",
			Value:      "tok",
			HTTP:       httpCtx,
			Properties: props,
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to verify new API token")
	})

	t.Run("success updates secret", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"apiToken":{"workspaces":[{"id":"w-1"}]}}}`)),
				},
			},
		}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: "apiToken",
			Value:      "valid-token",
			HTTP:       httpCtx,
			Properties: props,
			Secrets:    secrets,
		})
		require.NoError(t, err)
		v, getErr := secrets.Get("apiToken")
		require.NoError(t, getErr)
		assert.Equal(t, "valid-token", v)
	})
}

func Test__Railway__SetupProvider__OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:   core.StepInfo{Name: "nope"},
			Logger: logger,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("capabilitySelection success", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{}
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepCapabilitySelection, Capabilities: []string{"railway.triggerDeploy"}},
			Logger:       logger,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepEnterAPIToken, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.ElementsMatch(t, []string{"railway.triggerDeploy"}, capCtx.RequestedCapabilties)
	})

	t.Run("enterAPIToken validation", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		props := intCtx.Properties()
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIToken, Inputs: "not-a-map"},
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIToken, Inputs: map[string]any{"apiToken": 123}},
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API token")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIToken, Inputs: map[string]any{"apiToken": ""}},
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API token is required")
	})

	t.Run("enterAPIToken success", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		props := intCtx.Properties()
		secrets := intCtx.Secrets()
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"railway.triggerDeploy"},
		}

		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"apiToken":{"workspaces":[{"id":"w-1"}]}}}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"apiToken":{"workspaces":[{"id":"w-1","name":"Main Workspace"}]}}}`)),
				},
			},
		}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIToken, Inputs: map[string]any{"apiToken": "good-token"}},
			Logger:       logger,
			Properties:   props,
			Secrets:      secrets,
			Capabilities: capCtx,
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, SetupStepDone, next.Name)

		wsID, _ := props.GetString("workspaceId")
		assert.Equal(t, "w-1", wsID)
		wsName, _ := props.GetString("workspaceName")
		assert.Equal(t, "Main Workspace", wsName)

		tok, _ := secrets.Get("apiToken")
		assert.Equal(t, "good-token", tok)

		assert.ElementsMatch(t, []string{"railway.triggerDeploy"}, capCtx.EnabledCapabilities)
	})
}

func Test__Railway__SetupProvider__OnStepRevert(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		err := s.OnStepRevert(core.SetupStepContext{
			Step:   core.StepInfo{Name: "nope"},
			Logger: logger,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("enterAPIToken Revert deletes properties and secrets", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		props := intCtx.Properties()
		_ = props.Create(core.IntegrationPropertyDefinition{Name: "workspaceId", Value: "w-1"})
		_ = intCtx.SetSecret("apiToken", []byte("tok"))

		err := s.OnStepRevert(core.SetupStepContext{
			Step:       core.StepInfo{Name: SetupStepEnterAPIToken},
			Logger:     logger,
			Properties: props,
			Secrets:    intCtx.Secrets(),
		})
		require.NoError(t, err)

		_, err = props.GetString("workspaceId")
		assert.Error(t, err)
		_, err = intCtx.Secrets().Get("apiToken")
		assert.Error(t, err)
	})
}
