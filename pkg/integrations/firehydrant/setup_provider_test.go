package firehydrant

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

func Test__FireHydrant__SetupProvider__OnStepSubmit(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}
	capCtx := &contexts.CapabilityContext{RequestedCapabilties: []string{"firehydrant.createIncident"}}

	next, err := provider.OnStepSubmit(core.SetupStepContext{
		Step:         core.StepInfo{Name: integrationsetup.StepEnterCredentials, Inputs: map[string]any{"apiKey": "fh-key"}},
		Logger:       logger.DiscardLogger(),
		HTTP:         httpCtx,
		Properties:   contexts.NewIntegrationPropertyStorage(),
		Secrets:      intCtx.Secrets(),
		Capabilities: capCtx,
	})
	require.NoError(t, err)
	require.NotNil(t, next)
	assert.Equal(t, core.SetupStepTypeDone, next.Type)
	assert.Equal(t, BaseURL+"/severities", httpCtx.Requests[0].URL.String())
	assert.Equal(t, "Bearer fh-key", httpCtx.Requests[0].Header.Get("Authorization"))

	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "fh-key", value)
	assert.Equal(t, []string{"firehydrant.createIncident"}, capCtx.EnabledCapabilities)
}

func Test__FireHydrant__SetupProvider__OnSecretUpdate(t *testing.T) {
	provider := newSetupProvider()
	intCtx := &contexts.IntegrationContext{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
		},
	}

	_, err := provider.OnSecretUpdate(core.SecretUpdateContext{
		SecretName: "apiKey",
		Value:      "updated-fh",
		Logger:     logger.DiscardLogger(),
		HTTP:       httpCtx,
		Properties: contexts.NewIntegrationPropertyStorage(),
		Secrets:    intCtx.Secrets(),
	})
	require.NoError(t, err)
	value, err := intCtx.Secrets().Get("apiKey")
	require.NoError(t, err)
	assert.Equal(t, "updated-fh", value)
}
