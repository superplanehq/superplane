package semaphore

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

func Test__Semaphore__SetupProvider__OnCapabilityUpdate(t *testing.T) {
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
				core.IntegrationCapabilityStateRequested: {"semaphore.runWorkflow"},
			},
			Capabilities: localCap,
		})
		require.NoError(t, err)
		assert.Equal(t, []string{"semaphore.runWorkflow"}, localCap.EnabledCapabilities)
	})
}

func Test__Semaphore__SetupProvider__OnSecretUpdate(t *testing.T) {
	orgURL := "https://example.semaphoreci.com"
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	props := contexts.NewIntegrationPropertyStorage()
	require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
		Name:  "organizationUrl",
		Value: orgURL,
	}))

	intCtx := &contexts.IntegrationContext{}
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

	t.Run("listing projects fails", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader("boom")),
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
		assert.Contains(t, err.Error(), "error listing projects")
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, orgURL+"/api/v1alpha/projects", httpCtx.Requests[0].URL.String())
	})

	t.Run("success updates secret", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("[]")),
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

func Test__Semaphore__SetupProvider__OnStepSubmit(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:   "nope",
			Logger: logger,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("selectOrganization validation", func(t *testing.T) {
		base := core.SetupStepContext{
			Step:       "selectOrganization",
			Logger:     logger,
			Properties: &contexts.IntegrationPropertyStorage{},
		}

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:       base.Step,
			Logger:     base.Logger,
			Properties: base.Properties,
			Inputs:     "not-a-map",
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:       base.Step,
			Logger:     base.Logger,
			Properties: base.Properties,
			Inputs:     map[string]any{"organizationUrl": 42},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid organization URL")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:       base.Step,
			Logger:     base.Logger,
			Properties: base.Properties,
			Inputs:     map[string]any{"organizationUrl": ""},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "organization URL is required")
	})

	t.Run("selectOrganization success", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		orgURL := "https://org.semaphoreci.com"
		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step:       "selectOrganization",
			Logger:     logger,
			Properties: props,
			Inputs:     map[string]any{"organizationUrl": orgURL},
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, "enterAPIToken", next.Name)
		assert.Equal(t, core.SetupStepTypeInputs, next.Type)

		stored, getErr := props.GetString("organizationUrl")
		require.NoError(t, getErr)
		assert.Equal(t, orgURL, stored)
		assert.Contains(t, next.Instructions, orgURL)
	})

	t.Run("enterAPIToken validation", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  "organizationUrl",
			Value: "https://example.semaphoreci.com",
		}))

		base := core.SetupStepContext{
			Step:         "enterAPIToken",
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         &contexts.HTTPContext{},
		}

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         base.Step,
			Logger:       base.Logger,
			Properties:   base.Properties,
			Secrets:      base.Secrets,
			Capabilities: base.Capabilities,
			HTTP:         base.HTTP,
			Inputs:       123,
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid input")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         base.Step,
			Logger:       base.Logger,
			Properties:   base.Properties,
			Secrets:      base.Secrets,
			Capabilities: base.Capabilities,
			HTTP:         base.HTTP,
			Inputs:       map[string]any{"apiToken": 1},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid API token")

		_, err = s.OnStepSubmit(core.SetupStepContext{
			Step:         base.Step,
			Logger:       base.Logger,
			Properties:   base.Properties,
			Secrets:      base.Secrets,
			Capabilities: base.Capabilities,
			HTTP:         base.HTTP,
			Inputs:       map[string]any{"apiToken": ""},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "API token is required")
	})

	t.Run("enterAPIToken ListProjects fails", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  "organizationUrl",
			Value: "https://example.semaphoreci.com",
		}))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusBadGateway,
					Body:       io.NopCloser(strings.NewReader("err")),
				},
			},
		}

		_, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         "enterAPIToken",
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: &contexts.CapabilityContext{},
			HTTP:         httpCtx,
			Inputs:       map[string]any{"apiToken": "t"},
		})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "error listing projects")
	})

	t.Run("enterAPIToken success", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		orgURL := "https://good.semaphoreci.com"
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  "organizationUrl",
			Value: orgURL,
		}))
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader("[]")),
				},
			},
		}
		capCtx := &contexts.CapabilityContext{
			RequestedCapabilties: []string{"semaphore.runWorkflow", "semaphore.onPipelineDone"},
		}

		next, err := s.OnStepSubmit(core.SetupStepContext{
			Step:         "enterAPIToken",
			Logger:       logger,
			Properties:   props,
			Secrets:      intCtx.Secrets(),
			Capabilities: capCtx,
			HTTP:         httpCtx,
			Inputs:       map[string]any{"apiToken": "final-token"},
		})
		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, core.SetupStepTypeDone, next.Type)
		assert.Equal(t, "done", next.Name)

		v, getErr := intCtx.Secrets().Get("apiToken")
		require.NoError(t, getErr)
		assert.Equal(t, "final-token", v)
		assert.ElementsMatch(t, []string{"semaphore.runWorkflow", "semaphore.onPipelineDone"}, capCtx.EnabledCapabilities)
		assert.Contains(t, next.Instructions, orgURL)
	})
}

func Test__Semaphore__SetupProvider__OnStepRevert(t *testing.T) {
	s := &SetupProvider{}
	logger := logger.DiscardLogger()

	t.Run("unknown step", func(t *testing.T) {
		err := s.OnStepRevert(core.SetupStepContext{Step: "x", Logger: logger})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown step")
	})

	t.Run("selectOrganization clears property", func(t *testing.T) {
		props := contexts.NewIntegrationPropertyStorage()
		require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
			Name:  "organizationUrl",
			Value: "https://example.semaphoreci.com",
		}))
		require.NoError(t, s.OnStepRevert(core.SetupStepContext{
			Step:       "selectOrganization",
			Logger:     logger,
			Properties: props,
		}))
		_, err := props.GetString("organizationUrl")
		require.Error(t, err)
	})

	t.Run("enterAPIToken clears secret", func(t *testing.T) {
		intCtx := &contexts.IntegrationContext{}
		require.NoError(t, intCtx.SetSecret("apiToken", []byte("sek")))

		require.NoError(t, s.OnStepRevert(core.SetupStepContext{
			Step:    "enterAPIToken",
			Logger:  logger,
			Secrets: intCtx.Secrets(),
		}))
		_, err := intCtx.Secrets().Get("apiToken")
		require.Error(t, err)
	})
}
