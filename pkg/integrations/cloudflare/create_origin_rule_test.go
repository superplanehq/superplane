package cloudflare

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateOriginRule__Setup(t *testing.T) {
	component := &CreateOriginRule{}

	t.Run("missing zone returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"expression": "http.host eq \"example.com\"",
				"originHost": "origin.example.com",
			},
		})

		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing expression returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"originHost": "origin.example.com",
			},
		})

		require.ErrorContains(t, err, "matchRules is required")
	})

	t.Run("missing originHost returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"expression": "http.host eq \"example.com\"",
			},
		})

		require.ErrorContains(t, err, "at least one origin parameter must be rewritten")
	})

	t.Run("invalid port returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"expression": "http.host eq \"example.com\"",
				"originHost": "origin.example.com",
				"originPort": 70000,
			},
		})

		require.ErrorContains(t, err, "originPort must be between 1 and 65535")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"expression": "http.host eq \"example.com\"",
				"originHost": "origin.example.com",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid match rule configuration passes", func(t *testing.T) {
		metadata := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"matchMode":  originRuleMatchCustom,
				"originHost": "origin.example.com",
				"matchRules": []map[string]any{
					{"field": "fullUri", "operator": "wildcard", "value": "/*", "conjunction": "and"},
					{"field": "uriPath", "operator": "startsWith", "value": "/api"},
				},
			},
			Metadata:    metadata,
			Integration: cloudflareIntegrationContext(),
		})

		require.NoError(t, err)
		nodeMetadata := metadata.Metadata.(OriginRuleNodeMetadata)
		assert.Equal(t, "zone123", nodeMetadata.Zone)
		assert.Equal(t, "example.com", nodeMetadata.ZoneName)
		assert.Equal(t, `(http.request.full_uri wildcard r"/*" and starts_with(http.request.uri.path, "/api"))`, nodeMetadata.Expression)
		assert.Equal(t, "origin.example.com", nodeMetadata.OriginHost)
		assert.Equal(t, []string{"DNS Record"}, nodeMetadata.Rewrites)
	})

	t.Run("all incoming requests configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"matchMode":  originRuleMatchAll,
				"originHost": "origin.example.com",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateOriginRule__Execute(t *testing.T) {
	component := &CreateOriginRule{}

	t.Run("creates rule in existing ruleset", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				originRuleResponse(http.StatusOK, `
					{
						"success": true,
						"result": {
							"id": "ruleset123",
							"phase": "http_request_origin",
							"rules": []
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
									"expression": "starts_with(http.request.uri.path, \"/api/\")",
									"description": "Route API",
									"enabled": true,
									"action_parameters": {
										"host_header": "api.example.com",
										"origin": {"host": "origin.example.com", "port": 8443}
									}
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
				"zone":        "zone123",
				"description": "Route API",
				"originHost":  "origin.example.com",
				"originPort":  8443,
				"hostHeader":  "api.example.com",
				"enabled":     true,
				"matchMode":   originRuleMatchCustom,
				"matchRules": []map[string]any{
					{"field": "uriPath", "operator": "startsWith", "value": "/api/", "conjunction": "and"},
				},
			},
			HTTP:           httpContext,
			Integration:    cloudflareIntegrationContext(),
			ExecutionState: execState,
		})

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "cloudflare.createOriginRule", execState.Type)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/phases/http_request_origin/entrypoint", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/ruleset123/rules", httpContext.Requests[1].URL.String())

		payload := requestJSON(t, httpContext.Requests[1])
		assert.Equal(t, "route", payload["action"])
		assert.Equal(t, `(starts_with(http.request.uri.path, "/api/"))`, payload["expression"])
		assert.Equal(t, "origin.example.com", payload["action_parameters"].(map[string]any)["origin"].(map[string]any)["host"])
	})

	t.Run("creates phase ruleset when origin ruleset is missing", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				originRuleResponse(http.StatusNotFound, `{"success": false, "errors": [{"message": "not found"}]}`),
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
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"expression": "http.host eq \"example.com\"",
				"originHost": "origin.example.com",
			},
			HTTP:           httpContext,
			Integration:    cloudflareIntegrationContext(),
			ExecutionState: &contexts.ExecutionStateContext{KVs: make(map[string]string)},
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets", httpContext.Requests[1].URL.String())

		payload := requestJSON(t, httpContext.Requests[1])
		assert.Equal(t, "zone", payload["kind"])
		assert.Equal(t, originRulePhase, payload["phase"])
	})
}

func Test__CreateOriginRule__ExampleOutput(t *testing.T) {
	output := (&CreateOriginRule{}).ExampleOutput()
	assert.Equal(t, "cloudflare.createOriginRule", output["type"])
	assert.Equal(t, "2026-05-06T12:00:00Z", output["timestamp"])
	assert.Contains(t, output, "data")
}
