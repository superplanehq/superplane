package cloudflare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__DeleteOriginRule__Execute(t *testing.T) {
	component := &DeleteOriginRule{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			originRuleResponse(http.StatusOK, `
				{
					"success": true,
					"result": {
						"id": "ruleset123",
						"phase": "http_request_origin",
						"rules": [
							{
								"id": "rule123",
								"action": "route",
								"expression": "http.host eq \"example.com\"",
								"enabled": true,
								"action_parameters": {"origin": {"host": "origin.example.com"}}
							}
						]
					}
				}
			`),
			originRuleResponse(http.StatusOK, `{"success": true, "result": {"id": "ruleset123", "rules": []}}`),
		},
	}

	execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
	err := component.Execute(core.ExecutionContext{
		Configuration:  map[string]any{"rule": "zone123/rule123"},
		HTTP:           httpContext,
		Integration:    cloudflareIntegrationContext(),
		ExecutionState: execState,
	})

	require.NoError(t, err)
	assert.True(t, execState.Passed)
	assert.Equal(t, "cloudflare.deleteOriginRule", execState.Type)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/ruleset123/rules/rule123", httpContext.Requests[1].URL.String())
}

func Test__DeleteOriginRule__ExampleOutput(t *testing.T) {
	output := (&DeleteOriginRule{}).ExampleOutput()
	assert.Equal(t, "cloudflare.deleteOriginRule", output["type"])
	assert.Equal(t, "2026-05-06T12:00:00Z", output["timestamp"])
	assert.Contains(t, output, "data")
}
