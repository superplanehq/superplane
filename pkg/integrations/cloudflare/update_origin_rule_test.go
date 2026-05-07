package cloudflare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateOriginRule__Execute(t *testing.T) {
	component := &UpdateOriginRule{}
	t.Run("updates rule", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				originRuleResponse(http.StatusOK, `
				{
					"success": true,
					"result": {
						"id": "ruleset123",
						"phase": "http_request_origin",
						"rules": [
							{"id": "rule123", "action": "route", "expression": "http.host eq \"example.com\"", "enabled": true}
						]
					}
				}
			`),
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
								"description": "Route API",
								"enabled": true,
								"action_parameters": {"origin": {"host": "new-origin.example.com"}}
							}
						]
					}
				}
			`),
			},
		}

		execState := &contexts.ExecutionStateContext{KVs: make(map[string]string)}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"rule":        "zone123/rule123",
				"description": "Route API",
				"expression":  "http.host eq \"example.com\"",
				"originHost":  "new-origin.example.com",
				"enabled":     true,
			},
			HTTP:           httpContext,
			Integration:    cloudflareIntegrationContext(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "cloudflare.updateOriginRule", execState.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/ruleset123/rules/rule123", httpContext.Requests[1].URL.String())
	})

	t.Run("preserves disabled update fields", func(t *testing.T) {
		port := 8443
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
								"description": "Existing route",
								"enabled": false,
								"action_parameters": {
									"host_header": "app.example.com",
									"origin": {"host": "old-origin.example.com", "port": 8443},
									"sni": {"value": "tls.example.com"}
								}
							}
						]
					}
				}
			`),
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
								"description": "Existing route",
								"enabled": false,
								"action_parameters": {
									"host_header": "app.example.com",
									"origin": {"host": "new-origin.example.com", "port": 8443},
									"sni": {"value": "tls.example.com"}
								}
							}
						]
					}
				}
			`),
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"rule":       "zone123/rule123",
				"originHost": "new-origin.example.com",
			},
			HTTP:           httpContext,
			Integration:    cloudflareIntegrationContext(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)

		payload := requestJSON(t, httpContext.Requests[1])
		assert.Equal(t, `http.host eq "example.com"`, payload["expression"])
		assert.Equal(t, "Existing route", payload["description"])
		assert.Equal(t, false, payload["enabled"])

		actionParameters := payload["action_parameters"].(map[string]any)
		assert.Equal(t, "app.example.com", actionParameters["host_header"])
		assert.Equal(t, map[string]any{"host": "new-origin.example.com", "port": float64(port)}, actionParameters["origin"])
		assert.Equal(t, map[string]any{"value": "tls.example.com"}, actionParameters["sni"])
	})
}

func Test__UpdateOriginRule__ExampleOutput(t *testing.T) {
	output := (&UpdateOriginRule{}).ExampleOutput()
	assert.Equal(t, "cloudflare.updateOriginRule", output["type"])
	assert.Equal(t, "2026-05-06T12:00:00Z", output["timestamp"])
	assert.Contains(t, output, "data")
}
