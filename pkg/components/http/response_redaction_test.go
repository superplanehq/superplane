package http

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestHTTPResponseHeaderRedaction(t *testing.T) {
	const body = `{
		"message":"safe-message",
		"request":{"headers":[["AUTHORIZATION","private-header-value"],["Accept","application/json"]],"cookies":"private-header-value"},
		"entries":[
			{"data":{"headers":{"Cookie":"private-header-value","X_API_KEY":"private-header-value","User-Agent":"safe-agent"}}},
			{"data":{"headers":[{"name":"X-Auth-Token","value":"private-header-value"},{"key":"X-Secret","value":"private-header-value"},{"name":"Accept","value":"text/plain"}]}}
		],
		"nested":{"Headers":"Authorization: private-header-value"},
		"singleHeader":{"headers":{"name":"Authorization","value":"private-header-value"}}
	}`
	for _, tc := range []struct {
		name      string
		redact    bool
		retry     bool
		hook      bool
		exhausted bool
		status    int
		body      string
	}{
		{name: "unchanged by default", status: 200},
		{name: "success", redact: true, status: 200},
		{name: "failure", redact: true, status: 503},
		{name: "initial retry success", redact: true, retry: true, status: 200},
		{name: "initial retry metadata", redact: true, retry: true, status: 503},
		{name: "successful retry", redact: true, retry: true, hook: true, status: 200},
		{name: "rescheduled retry metadata", redact: true, retry: true, hook: true, status: 503},
		{name: "exhausted retry", redact: true, retry: true, hook: true, exhausted: true, status: 503},
		{name: "malformed JSON", redact: true, status: 200, body: `{"headers":{"Authorization":"private-header-value"}`},
		{name: "text error", redact: true, retry: true, hook: true, exhausted: true, status: 503, body: "Authorization: private-header-value"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			config := map[string]any{"method": "GET", "url": "https://api.example.com"}
			if tc.redact {
				config["redactSensitiveHeaders"] = true
			}
			if tc.retry {
				config["retry"] = RetrySpec{Enabled: true, Strategy: RetryStrategyFixed, MaxAttempts: 2, IntervalSeconds: 5}
			}
			ctx, state, metadata := createExecutionContext(config)
			responseBody := body
			if tc.body != "" {
				responseBody = tc.body
			}
			ctx.HTTP = &contexts.HTTPContext{Responses: []*http.Response{newResponse(tc.status, responseBody, http.Header{
				"Set-Cookie": {"private-header-value"}, "X-Api-Key": {"private-header-value"}, "Content-Type": {"application/json"},
			})}}
			ctx.Requests = &contexts.RequestContext{}
			action := &HTTP{}
			if tc.hook {
				attempts := 1
				if tc.exhausted {
					attempts = 2
				}
				require.NoError(t, metadata.Set(Metadata{Retry: &RetryMetadata{Attempts: attempts, MaxAttempts: 2, Strategy: RetryStrategyFixed}}))
				require.NoError(t, action.HandleHook(core.ActionHookContext{
					Name: "retryRequest", Logger: ctx.Logger, Configuration: config, HTTP: ctx.HTTP,
					ExecutionState: state, Metadata: metadata, Requests: ctx.Requests,
				}))
			} else {
				require.NoError(t, action.Execute(ctx))
			}
			stored, err := json.Marshal(map[string]any{"outputs": state.Payloads, "metadata": metadata.Get()})
			require.NoError(t, err)
			if !tc.redact {
				assert.Contains(t, string(stored), "private-header-value")
				return
			}
			assert.NotContains(t, string(stored), "private-header-value")
			if tc.body == "" {
				assert.Contains(t, string(stored), "safe-message")
				assert.Contains(t, string(stored), "safe-agent")
				assert.Contains(t, string(stored), "application/json")
			}
			if tc.retry && tc.status == 503 && !tc.exhausted {
				assert.Equal(t, "retryRequest", ctx.Requests.(*contexts.RequestContext).Action)
				assert.Empty(t, state.Payloads)
				return
			}
			assert.Equal(t, tc.status, responsePayload(t, state)["status"])
		})
	}
}
