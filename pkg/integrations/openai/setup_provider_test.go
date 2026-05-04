package openai

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	integrationsetup "github.com/superplanehq/superplane/pkg/integrations/setup"
	"github.com/superplanehq/superplane/test/support/contexts"
	"github.com/superplanehq/superplane/test/support/logger"
)

func Test__OpenAI__SetupProvider__OnStepSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage()
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}
	capCtx := &contexts.CapabilityContext{RequestedCapabilties: []string{"openai.textPrompt"}}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step: core.StepInfo{
			Name: integrationsetup.StepEnterCredentials,
			Inputs: map[string]any{
				"baseURL": "https://custom.example.com/v1",
				"apiKey":  "sk-test",
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
	assert.Equal(t, "https://custom.example.com/v1/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer sk-test", httpCtx.Requests[0].Header.Get("Authorization"))

	apiKey, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "sk-test", apiKey)
	baseURL, err := props.GetString("baseURL")
	require.NoError(t, err)
	assert.Equal(t, "https://custom.example.com/v1", baseURL)
	assert.Equal(t, []string{"openai.textPrompt"}, capCtx.EnabledCapabilities)
}

func Test__OpenAI__SetupProvider__OnSecretUpdate(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage()
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
