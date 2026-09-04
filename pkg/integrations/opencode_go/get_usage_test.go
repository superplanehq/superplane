package opencodego

import (
	"net/http"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

// planUsageBody mirrors a real gateway response for GET /usage.
const planUsageBody = `{
	"usage": {
		"rolling": {"status": "ok", "percent": 0, "resetsAt": "2026-08-22T23:29:53.432Z"},
		"weekly": {"status": "ok", "percent": 51, "resetsAt": "2026-08-24T00:00:00.432Z"},
		"monthly": {"status": "ok", "percent": 48, "resetsAt": "2026-09-16T15:27:05.432Z"}
	}
}`

func Test__GetUsage__Execute(t *testing.T) {
	c := &GetUsage{}

	t.Run("success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{response(http.StatusOK, planUsageBody)}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(core.ExecutionContext{
			Logger:         logrus.NewEntry(logrus.New()),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    connectedIntegration(map[string]any{"apiKey": "oc-test-key"}),
			ExecutionState: state,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)

		request := httpContext.Requests[0]
		assert.Contains(t, request.URL.String(), baseURL+"/usage")
		assert.Equal(t, "Bearer oc-test-key", request.Header.Get("Authorization"), "the regular subscription key is enough")

		require.Len(t, state.Payloads, 1)
		assert.Equal(t, GetUsagePayloadType, state.Type)
		assert.Equal(t, core.DefaultOutputChannel.Name, state.Channel)

		payload := state.Payloads[0].(map[string]any)["data"].(*GetUsagePayload)
		assert.Equal(t, "ok", payload.RollingStatus)
		assert.Equal(t, float64(0), payload.RollingPercent)
		assert.Equal(t, "2026-08-22T23:29:53.432Z", payload.RollingResetsAt)
		assert.Equal(t, float64(51), payload.WeeklyPercent)
		assert.Equal(t, "2026-08-24T00:00:00.432Z", payload.WeeklyResetsAt)
		assert.Equal(t, float64(48), payload.MonthlyPercent)
		assert.Equal(t, "2026-09-16T15:27:05.432Z", payload.MonthlyResetsAt)
		require.NotNil(t, payload.Response)
		assert.Equal(t, "ok", payload.Response.Weekly.Status)
	})

	t.Run("a throttled window is reported as it arrives", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusOK, `{"usage":{
				"rolling": {"status": "limit_exceeded", "percent": 100, "resetsAt": "2026-08-23T01:00:00Z"},
				"weekly": {"status": "ok", "percent": 51, "resetsAt": "2026-08-24T00:00:00Z"},
				"monthly": {"status": "ok", "percent": 12, "resetsAt": "2026-09-16T00:00:00Z"}
			}}`),
		}}
		state := &contexts.ExecutionStateContext{}

		err := c.Execute(core.ExecutionContext{
			Logger:         logrus.NewEntry(logrus.New()),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    connectedIntegration(map[string]any{"apiKey": "oc-test-key"}),
			ExecutionState: state,
		})

		require.NoError(t, err)
		payload := state.Payloads[0].(map[string]any)["data"].(*GetUsagePayload)
		assert.Equal(t, "limit_exceeded", payload.RollingStatus)
		assert.Equal(t, float64(100), payload.RollingPercent)
	})
}

func Test__GetUsage__ExecuteErrors(t *testing.T) {
	c := &GetUsage{}

	t.Run("an unauthorized key surfaces the gateway error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{Responses: []*http.Response{
			response(http.StatusUnauthorized, `{"type":"error","error":{"type":"AuthError","message":"Missing API key."}}`),
		}}

		err := c.Execute(core.ExecutionContext{
			Logger:         logrus.NewEntry(logrus.New()),
			Configuration:  map[string]any{},
			HTTP:           httpContext,
			Integration:    connectedIntegration(map[string]any{"apiKey": "oc-bad-key"}),
			ExecutionState: &contexts.ExecutionStateContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "401")
		assert.Contains(t, err.Error(), "Missing API key.")
	})
}
