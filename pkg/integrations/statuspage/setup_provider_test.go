package statuspage

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

func Test__Statuspage__SetupProvider__OnStepSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage()
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
		},
	}
	capCtx := &contexts.CapabilityContext{RequestedCapabilties: []string{"statuspage.createIncident"}}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step: core.StepInfo{
			Name: integrationsetup.StepEnterCredentials,
			Inputs: map[string]any{
				"baseURL": "https://status.example.com/v1/",
				"apiKey":  "status-key",
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
	assert.Equal(t, "https://status.example.com/v1/pages", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "OAuth status-key", httpCtx.Requests[0].Header.Get("Authorization"))

	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "status-key", value)
	baseURL, err := props.GetString("baseURL")
	require.NoError(t, err)
	assert.Equal(t, "https://status.example.com/v1/", baseURL)
	assert.Equal(t, []string{"statuspage.createIncident"}, capCtx.EnabledCapabilities)
}

func Test__Statuspage__SetupProvider__OnSecretUpdate(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	props := contexts.NewIntegrationPropertyStorage()
	require.NoError(t, props.Create(core.IntegrationPropertyDefinition{Name: "baseURL", Value: "https://status.example.com/v1"}))
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`[]`))},
		},
	}

	_, err := provider.OnSecretUpdate(core.SecretUpdateContext{
		SecretName: "apiKey",
		Value:      "updated-status",
		Logger:     logger.DiscardLogger(),
		HTTP:       httpCtx,
		Properties: props,
		Secrets:    intCtx.Secrets(),
	})
	require.NoError(t, err)
	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "updated-status", value)
	assert.Equal(t, "https://status.example.com/v1/pages", httpCtx.Requests[0].URL.String())
}
