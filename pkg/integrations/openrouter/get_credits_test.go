package openrouter

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const creditsBody = `{"data":{"total_credits":100.5,"total_usage":25.75}}`

const keyBody = `{"data":{
	"label": "sk-or-v1-...c0de",
	"limit": 50,
	"limit_remaining": 24.25,
	"limit_reset": null,
	"usage": 25.75,
	"usage_daily": 1.2,
	"usage_weekly": 8.4,
	"usage_monthly": 25.75,
	"is_free_tier": false
}}`

// creditsContext is an integration connected via OAuth that also has the
// provisioning key /credits requires.
func creditsContext(httpContext *contexts.HTTPContext, state *contexts.ExecutionStateContext) core.ExecutionContext {
	return core.ExecutionContext{
		Logger:         logrus.NewEntry(logrus.New()),
		Configuration:  map[string]any{},
		HTTP:           httpContext,
		Integration:    connectedIntegration(map[string]any{"managementKey": "sk-or-provisioning"}),
		ExecutionState: state,
	}
}

func Test__GetCredits__Execute(t *testing.T) {
	c := &GetCredits{}

	t.Run("requires a provisioning key", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{}

		err := c.Execute(core.ExecutionContext{
			Logger:         logrus.NewEntry(logrus.New()),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    connectedIntegration(map[string]any{}),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.ErrorContains(t, err, "provisioning API key is not configured")
		assert.Empty(t, httpContext.Requests)
	})

	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, creditsBody),
			response(http.StatusOK, keyBody),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(creditsContext(httpContext, state))
		require.NoError(t, err)

		require.Len(t, httpContext.Requests, 2)
		assert.Contains(t, httpContext.Requests[0].URL.String(), "/credits")
		assert.Contains(t, httpContext.Requests[1].URL.String(), "/key")

		require.Len(t, state.Payloads, 1)
		output := state.Payloads[0].(map[string]any)["data"].(GetCreditsOutput)
		assert.Equal(t, 100.5, output.TotalCredits)
		assert.Equal(t, 25.75, output.TotalUsage)
		assert.Equal(t, 74.75, output.Balance)
		assert.Equal(t, "sk-or-v1-...c0de", output.Key.Label)
		require.NotNil(t, output.Key.Limit)
		assert.Equal(t, 50.0, *output.Key.Limit)
		require.NotNil(t, output.Key.LimitRemaining)
		assert.Equal(t, 24.25, *output.Key.LimitRemaining)
		assert.Nil(t, output.Key.LimitReset)
		assert.Equal(t, 1.2, output.Key.UsageDaily)
		assert.False(t, output.Key.IsFreeTier)
		assert.Equal(t, GetCreditsPayloadType, state.Type)
	})

	t.Run("keeps a missing credit limit null", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, creditsBody),
			response(http.StatusOK, `{"data":{"label":"free","limit":null,"limit_remaining":null,"usage":0,"is_free_tier":true}}`),
		}}
		state := &contexts.ExecutionStateContext{}

		require.NoError(t, c.Execute(creditsContext(httpContext, state)))

		output := state.Payloads[0].(map[string]any)["data"].(GetCreditsOutput)
		assert.Nil(t, output.Key.Limit)
		assert.Nil(t, output.Key.LimitRemaining)
		assert.True(t, output.Key.IsFreeTier)
	})

	t.Run("unauthorized", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusUnauthorized, `{"error":{"message":"User not found.","code":401}}`),
		}}

		err := c.Execute(creditsContext(httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "User not found.")
	})

	t.Run("fails when the key lookup fails", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, creditsBody),
			response(http.StatusTooManyRequests, `{"error":{"message":"Rate limit exceeded","code":429,"metadata":{"retry_after_seconds":5}}}`),
		}}

		err := c.Execute(creditsContext(httpContext, &contexts.ExecutionStateContext{}))

		require.Error(t, err)
		assert.Contains(t, err.Error(), "retry after 5 seconds")
	})
}
