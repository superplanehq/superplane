package claude

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

func Test__Claude__SetupProvider__FirstStep(t *testing.T) {
	s := &SetupProvider{}
	step := s.FirstStep(core.SetupStepContext{Logger: logger.DiscardLogger()})
	assert.Equal(t, core.SetupStepTypeCapabilitySelection, step.Type)
	assert.Equal(t, SetupStepCapabilitySelection, step.Name)
	assert.Equal(t, "Select capabilities", step.Label)
	assert.ElementsMatch(t, []string{"claude.textPrompt", "claude.runAgent"}, step.Capabilities)
}

func Test__Claude__SetupProvider__CapabilityGroups(t *testing.T) {
	s := &SetupProvider{}
	groups := s.CapabilityGroups()
	require.Len(t, groups, 2)

	assert.Equal(t, "Messages & prompts", groups[0].Label)
	require.Len(t, groups[0].Capabilities, 1)
	assert.Equal(t, "claude.textPrompt", groups[0].Capabilities[0].Name)

	assert.Equal(t, "Agents", groups[1].Label)
	require.Len(t, groups[1].Capabilities, 1)
	assert.Equal(t, "claude.runAgent", groups[1].Capabilities[0].Name)
}

func Test__Claude__SetupProvider__OnCapabilityUpdate(t *testing.T) {
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
				core.IntegrationCapabilityStateRequested: {"claude.textPrompt"},
			},
			Capabilities: localCap,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"claude.textPrompt"}, localCap.EnabledCapabilities)
	})
}

func Test__Claude__SetupProvider__OnSecretUpdate(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	intCtx := &contexts.IntegrationContext{}
	secrets := intCtx.Secrets()

	t.Run("unknown secret", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: "other",
			Value:      "x",
			HTTP:       &contexts.HTTPContext{},
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown secret")
	})

	t.Run("api key required", func(t *testing.T) {
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: SecretAPIKey,
			Value:      "   ",
			HTTP:       &contexts.HTTPContext{},
			Secrets:    secrets,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "value is required")
	})

	t.Run("verify fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"bad key"}}`)),
				},
			},
		}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: SecretAPIKey,
			Value:      "sk-invalid",
			HTTP:       httpCtx,
			Secrets:    secrets,
		})
		require.Error(t, err)
		require.Len(t, httpCtx.Requests, 1)
		assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/v1/models"))
	})

	t.Run("success updates secret", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		_, err := s.OnSecretUpdate(core.SecretUpdateContext{
			Logger:     logger,
			SecretName: SecretAPIKey,
			Value:      "sk-valid",
			HTTP:       httpCtx,
			Secrets:    secrets,
		})
		require.NoError(t, err)
		v, getErr := secrets.Get(SecretAPIKey)
		require.NoError(t, getErr)
		assert.Equal(t, "sk-valid", v)
	})
}

func Test__Claude__SetupProvider__OnStepSubmit(t *testing.T) {
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

	t.Run("capabilitySelection advances to API key step", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{}
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step: core.StepInfo{
				Name:         SetupStepCapabilitySelection,
				Capabilities: []string{"claude.textPrompt", "claude.runAgent"},
			},
			Logger:       logger,
			Capabilities: capCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepEnterAPIKey, next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)
		assert.Contains(t, next.Instructions, "platform.claude.com")
		assert.Len(t, next.Inputs, 1)
		assert.Equal(t, SecretAPIKey, next.Inputs[0].Name)
	})

	t.Run("enterAPIKey validation", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: "not-a-map"},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: map[string]any{SecretAPIKey: 1}},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API key")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: map[string]any{SecretAPIKey: ""}},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API key is required")
	})

	t.Run("enterAPIKey verify fails", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("err")),
				},
			},
		}

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: map[string]any{SecretAPIKey: "k"}},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         httpCtx,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "request failed")
		_, getErr := intCtx.Secrets().Get(SecretAPIKey)
		require.Error(t, getErr, "secret must not be stored when verification fails")
	})

	t.Run("enterAPIKey success", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"claude.textPrompt", "claude.runAgent"},
		}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: map[string]any{SecretAPIKey: "sk-final"}},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: capCtx,
			HTTP:         httpCtx,
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, SetupStepDone, next.Name)

		v, getErr := intCtx.Secrets().Get(SecretAPIKey)
		require.NoError(t, getErr)
		assert.Equal(t, "sk-final", v)
		assert.ElementsMatch(t, []string{"claude.textPrompt", "claude.runAgent"}, capCtx.EnabledCapabilities)
	})

	t.Run("enterAPIKey trims whitespace before storing", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":[]}`)),
				},
			},
		}
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"claude.textPrompt"},
		}

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         core.StepInfo{Name: SetupStepEnterAPIKey, Inputs: map[string]any{SecretAPIKey: "  sk-trimmed  \t"}},
			Logger:       logger,
			Secrets:      intCtx.Secrets(),
			Capabilities: capCtx,
			HTTP:         httpCtx,
		})
		require.NoError(t, err)

		v, getErr := intCtx.Secrets().Get(SecretAPIKey)
		require.NoError(t, getErr)
		assert.Equal(t, "sk-trimmed", v)
	})
}

func Test__Claude__SetupProvider__OnStepRevert(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		err := s.OnStepRevert(core.SetupStepContext{
			Step:   core.StepInfo{Name: "x"},
			Logger: logger,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("enterAPIKey clears secret", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.SetSecret(SecretAPIKey, []byte("sek")))

		require.NoError(t, s.OnStepRevert(core.SetupStepContext{
			Step:    core.StepInfo{Name: SetupStepEnterAPIKey},
			Logger:  logger,
			Secrets: intCtx.Secrets(),
		}))
		_, err := intCtx.Secrets().Get(SecretAPIKey)
		require.Error(t, err)
	})
}

func Test__Claude__SetupProvider__OnPropertyUpdate(t *testing.T) {
	s := &SetupProvider{}
	_, err := s.OnPropertyUpdate(core.PropertyUpdateContext{
		Logger: logger.DiscardLogger(),
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "not supported")
}
