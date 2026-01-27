package cloudflare

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__UpdateRedirectRule__Setup(t *testing.T) {
	component := &UpdateRedirectRule{}

	t.Run("missing zone returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "https://example.com/new",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "zone is required")
	})

	t.Run("missing ruleId returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "https://example.com/new",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "ruleId is required")
	})

	t.Run("missing sourceUrlPattern for wildcard match type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "",
				"targetUrl":        "https://example.com/new",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "sourceUrlPattern is required")
	})

	t.Run("missing expression for expression match type returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"ruleId":     "rule123",
				"matchType":  "expression",
				"expression": "",
				"targetUrl":  "https://example.com/new",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "expression is required")
	})

	t.Run("missing targetUrl returns error", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "",
			},
		}

		err := component.Setup(ctx)
		require.ErrorContains(t, err, "targetUrl is required")
	})

	t.Run("valid wildcard configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "https://example.com/new-path",
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("wildcard configuration with dynamic placeholder passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "https://example.com/new/${1}",
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("valid expression configuration passes", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"zone":       "zone123",
				"ruleId":     "rule123",
				"matchType":  "expression",
				"expression": "(http.host eq \"example.com\")",
				"targetUrl":  "https://example.com/new",
			},
		}

		err := component.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__UpdateRedirectRule__Execute(t *testing.T) {
	component := &UpdateRedirectRule{}

	t.Run("successful update with wildcard pattern emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Response for GetRulesetForPhase
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "ruleset123",
								"name": "default",
								"phase": "http_request_dynamic_redirect",
								"rules": []
							}
						}
					`)),
				},
				// Response for UpdateRedirectRule
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "ruleset123",
								"name": "default",
								"phase": "http_request_dynamic_redirect",
								"rules": [
									{
										"id": "rule123",
										"action": "redirect",
										"expression": "(http.request.full_uri wildcard r\"https://example.com/*\")",
										"description": "Test redirect",
										"enabled": true,
										"action_parameters": {
											"from_value": {
												"status_code": 301,
												"target_url": {
													"expression": "wildcard_replace(http.request.full_uri, r\"https://example.com/*\", r\"https://example.com/new/${1}\")"
												}
											}
										}
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":                "zone123",
				"ruleId":              "rule123",
				"description":         "Test redirect",
				"matchType":           "wildcard",
				"sourceUrlPattern":    "https://example.com/*",
				"targetUrl":           "https://example.com/new/${1}",
				"statusCode":          "301",
				"preserveQueryString": false,
				"enabled":             true,
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.redirectRule", execState.Type)
		assert.Len(t, execState.Payloads, 1)

		require.Len(t, httpContext.Requests, 2)
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/phases/http_request_dynamic_redirect/entrypoint", httpContext.Requests[0].URL.String())
		assert.Equal(t, "https://api.cloudflare.com/client/v4/zones/zone123/rulesets/ruleset123/rules/rule123", httpContext.Requests[1].URL.String())
	})

	t.Run("successful update with expression emits result", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Response for GetRulesetForPhase
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "ruleset123",
								"name": "default",
								"phase": "http_request_dynamic_redirect",
								"rules": []
							}
						}
					`)),
				},
				// Response for UpdateRedirectRule
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`
						{
							"success": true,
							"result": {
								"id": "ruleset123",
								"name": "default",
								"phase": "http_request_dynamic_redirect",
								"rules": [
									{
										"id": "rule123",
										"action": "redirect",
										"expression": "(http.host eq \"example.com\")",
										"description": "Test redirect",
										"enabled": true,
										"action_parameters": {
											"from_value": {
												"status_code": 301,
												"target_url": {
													"value": "https://example.com/new"
												}
											}
										}
									}
								]
							}
						}
					`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":                "zone123",
				"ruleId":              "rule123",
				"description":         "Test redirect",
				"matchType":           "expression",
				"expression":          "(http.host eq \"example.com\")",
				"targetUrl":           "https://example.com/new",
				"statusCode":          "301",
				"preserveQueryString": false,
				"enabled":             true,
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execState,
		}

		err := component.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, execState.Passed)
		assert.Equal(t, "default", execState.Channel)
		assert.Equal(t, "cloudflare.redirectRule", execState.Type)
		assert.Len(t, execState.Payloads, 1)
	})

	t.Run("API error returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// Response for GetRulesetForPhase - failure
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"success": false, "errors": [{"message": "Ruleset not found"}]}`)),
				},
			},
		}

		appCtx := &contexts.AppInstallationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
			},
		}

		execState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			Configuration: map[string]any{
				"zone":             "zone123",
				"ruleId":           "rule123",
				"matchType":        "wildcard",
				"sourceUrlPattern": "https://example.com/*",
				"targetUrl":        "https://example.com/new-path",
				"statusCode":       "301",
				"enabled":          true,
			},
			HTTP:            httpContext,
			AppInstallation: appCtx,
			ExecutionState:  execState,
		}

		err := component.Execute(ctx)

		require.Error(t, err)
		assert.Contains(t, err.Error(), "error getting ruleset")
	})
}
