package claude

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

func Test__Claude__SetupProvider__OnStepSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}
	capCtx := &contexts.CapabilityContext{RequestedCapabilties: []string{"claude.textPrompt"}}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step:         core.StepInfo{Name: integrationsetup.StepEnterCredentials, Inputs: map[string]any{"apiKey": "claude-key"}},
		Logger:       logger.DiscardLogger(),
		HTTP:         httpCtx,
		Properties:   contexts.NewIntegrationPropertyStorage(),
		Secrets:      intCtx.Secrets(),
		Capabilities: capCtx,
	})
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, core.SetupStepTypeDone, next.Type)
	assert.Equal(t, defaultBaseURL+"/models", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "claude-key", httpCtx.Requests[0].Header.Get("x-api-key"))

	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "claude-key", value)
	assert.Equal(t, []string{"claude.textPrompt"}, capCtx.EnabledCapabilities)
}

func Test__Claude__SetupProvider__OnSecretUpdate(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}

	_, err := provider.OnSecretUpdate(core.SecretUpdateContext{
		SecretName: "apiKey",
		Value:      "updated-key",
		Logger:     logger.DiscardLogger(),
		HTTP:       httpCtx,
		Properties: contexts.NewIntegrationPropertyStorage(),
		Secrets:    intCtx.Secrets(),
	})
	require.NoError(t, err)
	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "updated-key", value)
}
