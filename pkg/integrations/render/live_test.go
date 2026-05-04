package render

import (
	"net/http"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

type liveHTTPContext struct {
	client *http.Client
}

func (c liveHTTPContext) Do(request *http.Request) (*http.Response, error) {
	return c.client.Do(request)
}

func Test__Render_LiveCustomDomainActions(t *testing.T) {
	if os.Getenv("SUPERPLANE_RENDER_LIVE_TEST") != "1" {
		t.Skip("set SUPERPLANE_RENDER_LIVE_TEST=1 to run live Render custom-domain tests")
	}

	apiKey := os.Getenv("RENDER_API_KEY")
	addServiceID := os.Getenv("RENDER_TEST_ADD_SERVICE_ID")

	require.NotEmpty(t, apiKey, "RENDER_API_KEY is required")
	require.NotEmpty(t, addServiceID, "RENDER_TEST_ADD_SERVICE_ID is required")

	httpCtx := liveHTTPContext{client: &http.Client{Timeout: 30 * time.Second}}
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": apiKey}}
	client, err := NewClient(httpCtx, integration)
	require.NoError(t, err)

	domainSuffix := time.Now().UTC().Format("20060102150405")
	noWaitDomain := "sp-live-no-wait-" + domainSuffix + ".elffie.com"
	waitDomain := "sp-live-wait-" + domainSuffix + ".elffie.com"
	t.Cleanup(func() {
		_ = client.RemoveCustomDomain(addServiceID, noWaitDomain)
		_ = client.RemoveCustomDomain(addServiceID, waitDomain)
	})

	t.Run("add custom domain with wait disabled -> emits payload", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		t.Cleanup(func() {
			_ = client.RemoveCustomDomain(addServiceID, noWaitDomain)
		})

		err := (&AddCustomDomain{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    integration,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Configuration: map[string]any{
				"service":             addServiceID,
				"domain":              noWaitDomain,
				"waitForVerification": false,
			},
		})

		require.NoError(t, err)
		assert.Equal(t, core.DefaultOutputChannel.Name, executionState.Channel)
		assert.Equal(t, AddCustomDomainPayloadType, executionState.Type)
		require.NotEmpty(t, executionState.KVs[addCustomDomainExecutionKey])
	})

	t.Run("add custom domain with wait enabled -> triggers verification and schedules poll", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}
		t.Cleanup(func() {
			_ = client.RemoveCustomDomain(addServiceID, waitDomain)
		})

		err := (&AddCustomDomain{}).Execute(core.ExecutionContext{
			HTTP:           httpCtx,
			Integration:    integration,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       requests,
			Configuration: map[string]any{
				"service":             addServiceID,
				"domain":              waitDomain,
				"waitForVerification": true,
			},
		})

		require.NoError(t, err)
		assert.Empty(t, executionState.Channel)
		require.NotEmpty(t, executionState.KVs[addCustomDomainExecutionKey])
		assert.Equal(t, "poll", requests.Action)
		assert.Equal(t, AddCustomDomainPollInterval, requests.Duration)
	})
}
