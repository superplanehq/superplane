package sentry

import (
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IssueDescription(t *testing.T) {
	fullIssue := map[string]any{
		"id":        "123",
		"shortId":   "IPE-1",
		"title":     "Error #1: This is a test error!",
		"culprit":   "SentryCustomError(frontend/src/util)",
		"level":     "error",
		"status":    "unresolved",
		"substatus": "new",
		"firstSeen": "2022-04-04T18:17:18.320000Z",
		"lastSeen":  "2022-04-04T18:17:18.320000Z",
		"permalink": "https://your-org.sentry.io/issues/123/",
		"web_url":   "https://your-org.sentry.io/issues/123/",
		"count":     "42",
		"project": map[string]any{
			"id":   "456",
			"name": "ipe",
			"slug": "ipe",
		},
		"assignedTo": map[string]any{
			"type": "user",
			"id":   "789",
			"name": "Person",
		},
		"tags": []any{
			map[string]any{"key": "environment", "value": "production"},
		},
	}

	event := &IssueEventDetail{
		EventID: "evt-1",
		Title:   "Error #1: This is a test error!",
		Culprit: "SentryCustomError(frontend/src/util)",
		SDK:     map[string]any{"name": "sentry.javascript.browser", "version": "8.0.0"},
		Contexts: map[string]any{
			"browser": map[string]any{"name": "Chrome", "version": "151.0"},
			"os":      map[string]any{"name": "macOS", "version": "15.7"},
			"trace":   map[string]any{"trace_id": "abc123"},
			"replay":  map[string]any{"replay_id": "rep-9"},
		},
		Tags: []IssueTag{{Key: "environment", Value: "production"}},
		Entries: []IssueEventEntry{
			{
				Type: "exception",
				Data: map[string]any{
					"values": []any{
						map[string]any{
							"type":  "Error",
							"value": "This is a test error!",
							"stacktrace": map[string]any{
								"frames": []any{
									map[string]any{
										"filename": "vendor.js",
										"function": "wrap",
										"lineNo":   1.0,
										"inApp":    false,
									},
									map[string]any{
										"filename": "frontend/src/util.ts",
										"function": "SentryCustomError",
										"lineNo":   44.0,
										"inApp":    true,
									},
								},
							},
						},
					},
				},
			},
			{
				Type: "request",
				Data: map[string]any{
					"method": "GET",
					"url":    "https://app.example.com/checkout",
				},
			},
			{
				Type: "breadcrumbs",
				Data: map[string]any{
					"values": []any{
						map[string]any{"level": "info", "category": "auth", "message": "user logged in"},
						map[string]any{"level": "warning", "category": "cart", "message": "qty parsed from query string"},
					},
				},
			},
		},
	}

	t.Run("full issue and event follow sentry sections", func(t *testing.T) {
		body := IssueDescription(fullIssue, event)
		assert.Contains(t, body, "https://your-org.sentry.io/issues/123/")
		assert.Contains(t, body, "## Highlights")
		assert.Contains(t, body, "**Title:** Error #1: This is a test error!")
		assert.Contains(t, body, "**Environment:** production")
		assert.Contains(t, body, "**Status:** unresolved (new)")
		assert.Contains(t, body, "## Stack Trace")
		assert.Contains(t, body, "Error: This is a test error!")
		assert.Contains(t, body, "SentryCustomError (frontend/src/util.ts:44) [in app]")
		assert.Contains(t, body, "## HTTP Request")
		assert.Contains(t, body, "GET https://app.example.com/checkout")
		assert.Contains(t, body, "## Tags")
		assert.Contains(t, body, "## Contexts")
		assert.Contains(t, body, "**browser:** Chrome 151.0")
		assert.Contains(t, body, "## Breadcrumbs")
		assert.Contains(t, body, "info · auth: user logged in")
		assert.Contains(t, body, "## SDK")
		assert.Contains(t, body, "sentry.javascript.browser 8.0.0")
		assert.Contains(t, body, "**Replay ID:** rep-9")
		assert.Contains(t, body, "**Trace ID:** abc123")
		assert.NotContains(t, body, "```json")
		assert.NotContains(t, body, "TODO")
		assert.NotContains(t, body, "Root Cause")
	})

	t.Run("issue without event keeps highlights and omits empty sections", func(t *testing.T) {
		body := IssueDescription(map[string]any{
			"title":     "Broken deploy",
			"permalink": "https://your-org.sentry.io/issues/9/",
		}, nil)
		assert.Equal(
			t,
			"https://your-org.sentry.io/issues/9/\n\n## Highlights\n\n- **Title:** Broken deploy",
			body,
		)
		assert.NotContains(t, body, "## Stack Trace")
		assert.NotContains(t, body, "```json")
	})

	t.Run("missing issue", func(t *testing.T) {
		assert.Empty(t, IssueDescription(nil, nil))
	})

	t.Run("non-map issue", func(t *testing.T) {
		assert.Empty(t, IssueDescription("not a map", nil))
	})

	t.Run("omits stack when the event has no frames", func(t *testing.T) {
		body := IssueDescription(fullIssue, &IssueEventDetail{
			Type:    "default",
			Entries: []IssueEventEntry{{Type: "message", Data: map[string]any{"formatted": "json body"}}},
		})
		assert.NotContains(t, body, "## Stack Trace")
		require.Contains(t, body, "## Highlights")
	})
}

func Test__FetchedIssueDescription(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			sentryMockResponse(http.StatusOK, `{"id":"123","title":"TypeError: boom","permalink":"https://sentry.io/issues/123/","count":"3"}`),
			sentryMockResponse(http.StatusOK, `{"eventID":"latest","entries":[{"type":"exception","data":{"values":[{"type":"TypeError","value":"boom","stacktrace":{"frames":[{"filename":"main.go","function":"Run","lineNo":10,"inApp":true}]}}]}}]}`),
		},
	}

	body := FetchedIssueDescription(testIssueClient(httpCtx), map[string]any{"id": "123", "title": "TypeError: boom"}, nil)
	assert.Contains(t, body, "**Count:** 3")
	assert.Contains(t, body, "Run (main.go:10) [in app]")
	assert.NotContains(t, body, "```json")
}
