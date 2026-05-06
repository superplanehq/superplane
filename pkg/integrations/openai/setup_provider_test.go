package openai

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/openai/common"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__OpenAI__SetupProvider__OnStepSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage(intCtx)
	capCtx := &contexts.CapabilityContext{RequestedCapabilties: []string{"openai.textPrompt"}}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step: core.StepInfo{
			Name: SetupStepEnterBaseURL,
			Inputs: map[string]any{
				"baseURL": common.DefaultBaseURL,
			},
		},
		Logger:       logger.DiscardLogger(),
		Properties:   props,
		Secrets:      intCtx.Secrets(),
		Capabilities: capCtx,
	})
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, SetupStepEnterAPIKey, next.Name)
	assert.Contains(t, next.Instructions, "API key")
	assert.Contains(t, next.Instructions, "Restricted")
	assert.Contains(t, next.Instructions, "List models")
	assert.Contains(t, next.Instructions, "Read")
	assert.Contains(t, next.Instructions, "Responses (/v1/responses)")
	assert.Contains(t, next.Instructions, "Write")
	assert.NotContains(t, next.Instructions, "Model capabilities")
	assert.NotContains(t, next.Instructions, "Read Only permissions are enough")

	baseURL, err := props.GetString("baseURL")
	require.NoError(t, err)
	assert.Equal(t, common.DefaultBaseURL, baseURL)

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}

	next, err = provider.OnStepSubmit(core.SetupStepContext{
		Step: core.StepInfo{
			Name: SetupStepEnterAPIKey,
			Inputs: map[string]any{
				"apiKey": "sk-test",
			},
		},
		Logger:       logger.DiscardLogger(),
		HTTP:         httpCtx,
		Properties:   props,
		Secrets:      intCtx.Secrets(),
		Capabilities: capCtx,
	})
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, core.SetupStepTypeDone, next.Type)
	assert.Equal(t, common.DefaultBaseURL+"/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer sk-test", httpCtx.Requests[0].Header.Get("Authorization"))

	apiKey, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "sk-test", apiKey)
	assert.Equal(t, []string{"openai.textPrompt"}, capCtx.EnabledCapabilities)
}

func Test__OpenAI__SetupProvider__OnSecretUpdate(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage(intCtx)
	require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: "baseURL", Value: "https://custom.example.com/v1"}))
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}

	_, err := provider.OnSecretUpdate(core.SecretUpdateContext{
		SecretName:   "apiKey",
		Value:        "sk-new",
		Logger:       logger.DiscardLogger(),
		HTTP:         httpCtx,
		Properties:   props,
		Secrets:      intCtx.Secrets(),
		Capabilities: &contexts.CapabilityContext{},
	})
	require.NoError(t, err)
	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "sk-new", value)
	assert.Equal(t, "https://custom.example.com/v1/models", httpCtx.Requests[0].URL.String())
}

func Test__OpenAI__SetupProvider__OnCapabilityUpdate(t *testing.T) {
	provider := newSetupProvider()

	t.Run("enables immediately when permissions are already covered", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{
			EnabledCapabilities: []string{"openai.textPrompt"},
		}

		next, err := provider.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: logger.DiscardLogger(),
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"openai.textPrompt"},
			},
			Capabilities: capCtx,
		})

		require.NoError(t, err)
		require.Nil(t, next)
		assert.Contains(t, capCtx.EnabledCapabilities, "openai.textPrompt")
	})

	t.Run("returns update permissions step when new write permission is needed", func(t *testing.T) {
		capCtx := &contexts.CapabilityContext{}

		next, err := provider.OnCapabilityUpdate(core.CapabilityUpdateContext{
			Logger: logger.DiscardLogger(),
			Changes: map[core.IntegrationCapabilityState][]string{
				core.IntegrationCapabilityStateRequested: {"openai.textPrompt"},
			},
			Capabilities: capCtx,
		})

		require.NoError(t, err)
		require.NotNil(t, next)
		assert.Equal(t, SetupStepUpdatePermissions, next.Name)
		assert.Contains(t, next.Instructions, "Responses (/v1/responses)")
		assert.Contains(t, next.Instructions, "Write")
		assert.NotContains(t, next.Instructions, "List models")
		assert.NotContains(t, next.Instructions, "Model capabilities")
		assert.Contains(t, capCtx.RequestedCapabilties, "openai.textPrompt")
	})
}

func Test__OpenAI__SetupProvider__OnUpdatePermissionsSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	require.NoError(t, intCtx.SetSecret("apiKey", []byte("sk-old")))

	props := contexts.NewIntegrationPropertyStorage(intCtx)
	require.NoError(t, props.Create(core.IntegrationPropertyDefinition{
		Name:  "baseURL",
		Value: "https://custom.example.com/v1",
	}))

	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}

	capCtx := &contexts.CapabilityContext{
		RequestedCapabilties: []string{"openai.textPrompt"},
	}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step: core.StepInfo{
			Name: SetupStepUpdatePermissions,
			Inputs: map[string]any{
				"apiKey": "sk-new",
			},
		},
		Logger:       logger.DiscardLogger(),
		HTTP:         httpCtx,
		Properties:   props,
		Secrets:      intCtx.Secrets(),
		Capabilities: capCtx,
	})

	require.NoError(t, err)
	require.Nil(t, next)
	assert.Equal(t, "https://custom.example.com/v1/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer sk-new", httpCtx.Requests[0].Header.Get("Authorization"))
	assert.Contains(t, capCtx.EnabledCapabilities, "openai.textPrompt")

	apiKey, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "sk-new", apiKey)
}
