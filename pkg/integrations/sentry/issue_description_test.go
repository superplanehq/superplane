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
		assert.Contains(t, body, "[View in Sentry](https://your-org.sentry.io/issues/123/events/evt-1/)")
		assert.Contains(t, body, "## Highlights")
		assert.Contains(t, body, "**Title:** Error #1: This is a test error!")
		assert.Contains(t, body, "**Short ID:** IPE-1")
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
		assert.NotContains(t, body, "## Message")
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
			"[View in Sentry](https://your-org.sentry.io/issues/9/)\n\n## Highlights\n\n- **Title:** Broken deploy",
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
		assert.Contains(t, body, "**Event type:** default")
		assert.Contains(t, body, "## Message")
		assert.Contains(t, body, "json body")
	})

	t.Run("prefers the event web URL", func(t *testing.T) {
		body := IssueDescription(fullIssue, &IssueEventDetail{
			WebURL: "https://your-org.sentry.io/issues/123/events/abc/",
		})
		assert.Contains(t, body, "[View in Sentry](https://your-org.sentry.io/issues/123/events/abc/)")
		assert.NotContains(t, body, "events/evt-1")
	})

	t.Run("adds extra highlights and a message-only body", func(t *testing.T) {
		issue := map[string]any{
			"title":     "HTTP 500 /api/v1/runner/planning-sessions/specs",
			"shortId":   "PRODUCTION-98",
			"priority":  "high",
			"userCount": 3,
			"permalink": "https://your-org.sentry.io/issues/7738363580/",
			"tags": []any{
				map[string]any{"key": "release", "value": "from-tag"},
			},
		}
		body := IssueDescription(issue, &IssueEventDetail{
			Type:    "default",
			Message: "HTTP 500 POST /api/v1/runner/planning-sessions/specs",
			Release: IssueEventRelease{Version: "2026.09.17"},
			User: map[string]any{
				"username": "runner",
				"email":    "runner@example.com",
				"id":       "42",
			},
			Entries: []IssueEventEntry{
				{Type: "message", Data: map[string]any{"formatted": "HTTP 500 POST /api/v1/runner/planning-sessions/specs"}},
			},
		})
		assert.Contains(t, body, "**Short ID:** PRODUCTION-98")
		assert.Contains(t, body, "**Priority:** high")
		assert.Contains(t, body, "**Users affected:** 3")
		assert.Contains(t, body, "**Event type:** default")
		assert.Contains(t, body, "**Message:** HTTP 500 POST /api/v1/runner/planning-sessions/specs")
		assert.Contains(t, body, "**Release:** 2026.09.17")
		assert.NotContains(t, body, "**Release:** from-tag")
		assert.Contains(t, body, "## Message")
		assert.Contains(t, body, "## User")
		assert.Contains(t, body, "**username:** runner")
		assert.Contains(t, body, "**email:** runner@example.com")
		assert.Contains(t, body, "**id:** 42")
	})

	t.Run("uses the release tag when the event has no release field", func(t *testing.T) {
		body := IssueDescription(map[string]any{
			"title": "Broken deploy",
			"tags": []any{
				map[string]any{"key": "release", "value": "1.2.3"},
			},
		}, nil)
		assert.Contains(t, body, "**Release:** 1.2.3")
	})

	t.Run("writes safe request headers and body and omits secrets", func(t *testing.T) {
		body := IssueDescription(fullIssue, &IssueEventDetail{
			Entries: []IssueEventEntry{
				{
					Type: "request",
					Data: map[string]any{
						"method": "POST",
						"url":    "https://app.example.com/checkout",
						"headers": []any{
							[]any{"Content-Type", "application/json"},
							[]any{"Authorization", "Bearer secret-token"},
							[]any{"Cookie", "session=abc"},
							[]any{"X-Api-Key", "super-secret"},
							[]any{"User-Agent", "Go-http-client/1.1"},
						},
						"data": map[string]any{"spec": "draft"},
					},
				},
			},
		})
		assert.Contains(t, body, "POST https://app.example.com/checkout")
		assert.Contains(t, body, "**Content-Type:** application/json")
		assert.Contains(t, body, "**User-Agent:** Go-http-client/1.1")
		assert.Contains(t, body, `"spec":"draft"`)
		assert.NotContains(t, body, "secret-token")
		assert.NotContains(t, body, "session=abc")
		assert.NotContains(t, body, "super-secret")
		assert.NotContains(t, body, "Authorization")
		assert.NotContains(t, body, "Cookie")
		assert.NotContains(t, body, "X-Api-Key")
	})

	t.Run("keeps angle brackets in markdown highlights", func(t *testing.T) {
		body := IssueDescription(map[string]any{
			"title":     "fmt.wrapError: proxy listen <ip>:80: bind: permission denied",
			"permalink": "https://your-org.sentry.io/issues/9/",
		}, nil)
		assert.Contains(t, body, "**Title:** fmt.wrapError: proxy listen &lt;ip&gt;:80: bind: permission denied")
		assert.NotContains(t, body, "**Title:** fmt.wrapError: proxy listen :80")
	})

	t.Run("keeps angle brackets in http request details", func(t *testing.T) {
		body := IssueDescription(fullIssue, &IssueEventDetail{
			Entries: []IssueEventEntry{
				{
					Type: "request",
					Data: map[string]any{
						"method":       "GET",
						"url":          "https://<ip>:8080/checkout",
						"query_string": "host=<ip>",
						"headers": []any{
							[]any{"X-Forwarded-For", "<ip>"},
						},
					},
				},
			},
		})
		assert.Contains(t, body, "GET https://&lt;ip&gt;:8080/checkout")
		assert.Contains(t, body, "Query: host=&lt;ip&gt;")
		assert.Contains(t, body, "**X-Forwarded-For:** &lt;ip&gt;")
	})

	t.Run("omits tags when the issue lists keys without values", func(t *testing.T) {
		body := IssueDescription(map[string]any{
			"title": "Broken deploy",
			"tags": []any{
				map[string]any{"key": "command", "name": "Command", "totalValues": 3.0},
			},
		}, nil)
		assert.NotContains(t, body, "## Tags")
	})
}

func Test__FetchedIssueDescription(t *testing.T) {
	t.Run("uses a string release and stack from the latest event", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, `{"id":"123","title":"TypeError: boom","permalink":"https://sentry.io/issues/123/","count":"3"}`),
				sentryMockResponse(http.StatusOK, `{"eventID":"evt-latest","release":"1.4.2","web_url":"https://sentry.io/issues/123/events/evt-latest/","entries":[{"type":"exception","data":{"values":[{"type":"TypeError","value":"boom","stacktrace":{"frames":[{"filename":"main.go","function":"Run","lineNo":10,"inApp":true}]}}]}}]}`),
			},
		}

		body := FetchedIssueDescription(testIssueClient(httpCtx), map[string]any{"id": "123", "title": "TypeError: boom"}, nil)
		assert.Contains(t, body, "**Count:** 3")
		assert.Contains(t, body, "**Release:** 1.4.2")
		assert.Contains(t, body, "[View in Sentry](https://sentry.io/issues/123/events/evt-latest/)")
		assert.Contains(t, body, "Run (main.go:10) [in app]")
		assert.NotContains(t, body, "```json")
	})

	t.Run("keeps the event when Sentry returns a release object", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				sentryMockResponse(http.StatusOK, latestIssueAPIBody),
				sentryMockResponse(http.StatusOK, latestEventAPIBody),
			},
		}

		body := FetchedIssueDescription(
			testIssueClient(httpCtx),
			map[string]any{"id": "148481072", "title": "fmt.wrapError: proxy listen <ip>:80"},
			nil,
		)
		assert.Contains(t, body, "**Status:** unresolved (escalating)")
		assert.Contains(t, body, "**Release:** 4dd58c31ad607b30529d09686c0c8cc8381b2392")
		assert.Contains(t, body, "**Event type:** error")
		assert.Contains(t, body, "## Stack Trace")
		assert.Contains(t, body, "newErrorEvent (/home/runner/work/wrk3/wrk3/internal/telemetry/telemetry.go:89) [in app]")
		assert.Contains(t, body, "**command:** run")
		assert.Contains(t, body, "**runtime:** go go1.26.8")
		assert.Contains(t, body, "## Contexts")
		assert.Contains(t, body, "## Additional Data")
		assert.Contains(t, body, "## SDK")
		assert.Contains(t, body, "sentry.go 0.31.1")
		assert.Contains(t, body, "**Trace ID:** a96440db4d90323b020fc871a03807ff")
		assert.Contains(t, body, "proxy listen &lt;ip&gt;:80")
		assert.NotContains(t, body, "```json")
	})
}

const latestIssueAPIBody = `{
	"id":"148481072",
	"shortId":"WRK3-2",
	"title":"fmt.wrapError: proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied",
	"culprit":"github.com/mytmlt/wrk3/internal/telemetry in newErrorEvent",
	"permalink":"https://miyat-test-org-1.sentry.io/issues/148481072/",
	"level":"error",
	"status":"unresolved",
	"substatus":"escalating",
	"priority":"high",
	"count":"3",
	"userCount":0,
	"project":{"id":"4512124896608336","name":"go","slug":"wrk3"},
	"tags":[{"key":"command","name":"Command","totalValues":3},{"key":"runtime","name":"Runtime","totalValues":3}]
}`

const latestEventAPIBody = `{
	"id":"c5764589c1fc4b96aa6bcf9e57cb291b",
	"eventID":"c5764589c1fc4b96aa6bcf9e57cb291b",
	"title":"fmt.wrapError: proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied",
	"message":"proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied",
	"type":"error",
	"platform":"go",
	"culprit":"github.com/mytmlt/wrk3/internal/telemetry in newErrorEvent",
	"web_url":"https://miyat-test-org-1.sentry.io/issues/148481072/events/c5764589c1fc4b96aa6bcf9e57cb291b/",
	"release":{
		"id":137233750,
		"version":"4dd58c31ad607b30529d09686c0c8cc8381b2392",
		"shortVersion":"4dd58c31ad607b30529d09686c0c8cc8381b2392"
	},
	"sdk":{"name":"sentry.go","version":"0.31.1"},
	"context":{
		"command":"run",
		"error_type":"fmt.wrapError",
		"goos":"linux",
		"message":"proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied"
	},
	"contexts":{
		"os":{"name":"linux"},
		"runtime":{"name":"go","version":"go1.26.8"},
		"trace":{"trace_id":"a96440db4d90323b020fc871a03807ff"}
	},
	"tags":[
		{"key":"command","value":"run"},
		{"key":"runtime","value":"go go1.26.8"}
	],
	"entries":[
		{"type":"message","data":{"formatted":"proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied"}},
		{"type":"exception","data":{"values":[{
			"type":"fmt.wrapError",
			"value":"proxy listen <ip>:80: listen tcp <ip>:80: bind: permission denied",
			"stacktrace":{"frames":[
				{"filename":"/home/runner/work/wrk3/wrk3/main.go","function":"main","lineNo":32,"inApp":true},
				{"filename":"/home/runner/work/wrk3/wrk3/internal/telemetry/telemetry.go","function":"ReportIfEnabled","lineNo":65,"inApp":true},
				{"filename":"/home/runner/work/wrk3/wrk3/internal/telemetry/telemetry.go","function":"newErrorEvent","lineNo":89,"inApp":true}
			]}
		}]}},
		{"type":"threads","data":{"values":[{"id":"0","current":true,"name":"main"}]}}
	]
}`
