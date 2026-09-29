package datadog

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

func Test__OnErrorTrackingAlert__OnIntegrationMessage(t *testing.T) {
	trigger := &OnErrorTrackingAlert{}

	t.Run("emits triggered error tracking alerts", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"title":            "[Triggered] checkout new issues",
				"body":             "InventoryTimeout: checkout failed",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		assert.Equal(t, ErrorTrackingAlertPayloadType, events.Payloads[0].Type)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Contains(t, payload.Description, "InventoryTimeout: checkout failed")
		assert.Equal(t, payload.Description, payload.Body)
	})

	t.Run("loads the error tracking issue named by the alert", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				issueDetailsResponse(issueID),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{
						"id": "` + issueID + `",
						"attributes": {"total_count": 12, "impacted_users": 3, "impacted_sessions": 4},
						"relationships": {"issue": {"data": {"id": "` + issueID + `"}}}
					}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{"attributes": {"custom": {"error": {"stack": "goroutine 1 [running]:\nmain.Charge(checkout/pay.go:22)"}}}}]
				}`))},
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"title":            "[Triggered] myt-home dogfood error 783228ea-d3e4-4a21-9ccd-477b80b24c19",
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
				"tags":             "service:myt-home,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "TimeoutError: checkout timed out", payload.Title)
		assert.Equal(t, payload.Description, payload.Body)
		assert.Contains(t, payload.Description, "checkout/pay.go")
		assert.Contains(t, payload.Description, "Charge")
		assert.Contains(t, payload.Description, "myt-home")
		assert.Contains(t, payload.Description, "Occurrences:** 12")
		assert.Contains(t, payload.Description, "main.Charge(checkout/pay.go:22)")
		assert.Contains(t, payload.Description, "myt-home dogfood error")
		require.NotEmpty(t, httpContext.Requests)
		assert.Contains(t, httpContext.Requests[0].URL.Path, issueID)
		assert.NotContains(t, httpContext.Requests[0].URL.Path, "783228ea-d3e4-4a21-9ccd-477b80b24c19")
	})

	t.Run("adds error sample context and related logs", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				issueDetailsResponse(issueID),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{
						"id": "` + issueID + `",
						"attributes": {"total_count": 1},
						"relationships": {"issue": {"data": {"id": "` + issueID + `"}}}
					}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{
						"attributes": {
							"custom": {
								"otel": {"trace_id": "cf2c57cfc127be5a1f156480875acc0a", "span_id": "79c41ba11e216970"},
								"env": "development",
								"resource_name": "POST /api/debug/datadog-error",
								"http": {"method": "POST", "path": "/api/debug/datadog-error", "status_code": 500},
								"error": {"stack": "goroutine 121 [running]:\nruntime/debug.Stack()"}
							},
							"start_timestamp": "2026-09-27T19:51:32.010Z"
						}
					}]
				}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{
						"attributes": {
							"timestamp": "2026-09-27T19:51:32.010Z",
							"status": "error",
							"service": "myt-home",
							"message": "POST /api/debug/datadog-error -> 500 (0.2ms)"
						}
					}]
				}`))},
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"title":            "[Triggered] dogfood",
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, payload.Description, payload.Body)
		assert.Contains(t, payload.Description, "## Error sample")
		assert.Contains(t, payload.Description, "POST /api/debug/datadog-error -> 500")
		assert.Contains(t, payload.Description, "## Related logs")
		assert.Contains(t, payload.Description, "POST /api/debug/datadog-error -> 500 (0.2ms)")
		assert.Contains(t, payload.Description, "View the trace")
		assert.Contains(t, payload.Description, "View logs in Datadog")
	})

	t.Run("keeps the issue when sample lookup is forbidden", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				issueDetailsResponse(issueID),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"errors":["Forbidden"]}`))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"errors":["Forbidden"]}`))},
				{StatusCode: http.StatusForbidden, Body: io.NopCloser(strings.NewReader(`{"errors":["Forbidden"]}`))},
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"title":            "[Triggered] dogfood",
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "TimeoutError: checkout timed out", payload.Title)
		assert.Contains(t, payload.Description, "checkout timed out")
		assert.NotContains(t, payload.Description, "## Error sample")
		assert.NotContains(t, payload.Description, "## Related logs")
	})

	t.Run("emits when the monitor query names the configured service", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"alert_query":      `error-tracking("service:checkout").source("backend").new().rollup("count").by("@issue.id").last("1d") > 0`,
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("emits when alert scope names the configured service", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"alert_scope":      "service:checkout",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("emits when monitor tags name the configured service", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("ignores a longer service name that only shares a prefix", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout-api,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("ignores a monitor that covers every service", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"alert_query":      `error-tracking("*").source("backend").new().rollup("count").by("@issue.id").last("1d") > 0`,
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("ignores an alert for a different service", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{"service": "billing"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("ignores recovered alerts", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": "Recovered",
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("ignores re-triggered alerts when the field is empty", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionRetriggered,
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("emits re-triggered alerts when that transition is selected", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Configuration: map[string]any{
				"alertTransitions": []any{AlertTransitionRetriggered},
			},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionRetriggered,
				"title":            "[Re-Triggered] checkout new issues",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
	})

	t.Run("reads the environment from alert scope then tags", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"alert_scope":      "service:checkout,env:staging",
				"tags":             "service:checkout,env:prod,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "staging", payload.Environment)
	})

	t.Run("reads the environment from tags when scope has none", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout,env:prod,monitor",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "prod", payload.Environment)
	})

	t.Run("falls back to the sample environment when tags have none", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				issueDetailsResponse(issueID),
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{
					"data": [{"attributes": {"custom": {"env": "development", "resource_name": "POST /checkout"}}}]
				}`))},
			},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/error-tracking/issue/" + issueID,
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "development", payload.Environment)
	})

	t.Run("drops an alert when the loaded issue belongs to another service", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{issueDetailsResponse(issueID)},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout,monitor",
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/monitors/98765",
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})

	t.Run("uses the issue page when the loaded issue matches the service", func(t *testing.T) {
		const issueID = "da226b38-baac-11f1-bad1-da7ad0900005"
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{issueDetailsResponseFor(issueID, "checkout")},
		}
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"site":   "datadoghq.eu",
					"apiKey": "test-api-key",
					"appKey": "test-app-key",
				},
			},
			Configuration: map[string]any{"service": "checkout"},
			Message: map[string]any{
				"event_type":       ErrorTrackingAlertEventType,
				"alert_transition": AlertTransitionTriggered,
				"tags":             "service:checkout,monitor",
				"body":             issueID,
				"link":             "https://app.datadoghq.eu/monitors/98765",
			},
			Events: events,
		})
		require.NoError(t, err)
		require.Len(t, events.Payloads, 1)
		payload := events.Payloads[0].Data.(ErrorTrackingAlertPayload)
		assert.Equal(t, "https://app.datadoghq.eu/error-tracking/issue/"+issueID, payload.Link)
	})

	t.Run("ignores non error tracking events", func(t *testing.T) {
		events := &contexts.EventContext{}
		err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
			Message: map[string]any{
				"event_type":       "query_alert_monitor",
				"alert_transition": AlertTransitionTriggered,
			},
			Events: events,
		})
		require.NoError(t, err)
		assert.Empty(t, events.Payloads)
	})
}

func Test__OnErrorTrackingAlert__ExampleDataMatchesTrigger(t *testing.T) {
	trigger := &OnErrorTrackingAlert{}
	example := trigger.ExampleData()
	message, _ := example["data"].(map[string]any)
	require.NotNil(t, message)

	events := &contexts.EventContext{}
	err := trigger.OnIntegrationMessage(core.IntegrationMessageContext{
		Configuration: map[string]any{"service": "checkout"},
		Message:       message,
		Events:        events,
	})
	require.NoError(t, err)
	require.Len(t, events.Payloads, 1)
}

func issueDetailsResponse(issueID string) *http.Response {
	return issueDetailsResponseFor(issueID, "myt-home")
}

func issueDetailsResponseFor(issueID, service string) *http.Response {
	body := `{
		"data": {
			"id": "` + issueID + `",
			"type": "issue",
			"attributes": {
				"error_type": "TimeoutError",
				"error_message": "checkout timed out",
				"service": "` + service + `",
				"file_path": "checkout/pay.go",
				"function_name": "Charge",
				"platform": "BACKEND",
				"state": "OPEN"
			}
		}
	}`
	return &http.Response{
		StatusCode: http.StatusOK,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func TestIssueIDCandidates_PrefersExplicitIssueID(t *testing.T) {
	ids := issueIDCandidates(AlertDetails{
		Link:       "https://app.datadoghq.eu/event/event?id=da226b38-baac-11f1-bad1-da7ad0900005",
		AlertScope: "@issue.id:c1726a66-1f64-11ee-b338-da7ad0900002",
		Body:       "myt-home dogfood error 783228ea-d3e4-4a21-9ccd-477b80b24c19",
	})

	assert.Equal(t, []string{
		"c1726a66-1f64-11ee-b338-da7ad0900002",
		"da226b38-baac-11f1-bad1-da7ad0900005",
		"783228ea-d3e4-4a21-9ccd-477b80b24c19",
	}, ids)
}

func TestContainsServiceToken(t *testing.T) {
	assert.True(t, containsServiceToken(`error-tracking("service:checkout")`, "checkout"))
	assert.True(t, containsServiceToken("service:checkout,monitor", "checkout"))
	assert.True(t, containsServiceToken("SERVICE:Checkout", "checkout"))
	assert.False(t, containsServiceToken("service:checkout-api", "checkout"))
	assert.False(t, containsServiceToken("myservice:checkout", "checkout"))
	assert.False(t, containsServiceToken(`error-tracking("*")`, "checkout"))
	assert.False(t, containsServiceToken("", "checkout"))
}

func TestFirstTagValue(t *testing.T) {
	assert.Equal(t, "staging", firstTagValue("service:checkout,env:staging", "env"))
	assert.Equal(t, "Prod", firstTagValue("ENV:Prod,monitor", "env"))
	assert.Equal(t, "checkout", firstTagValue(`error-tracking("service:checkout")`, "service"))
	assert.Equal(t, "", firstTagValue("service:checkout", "env"))
	assert.Equal(t, "", firstTagValue("environment:prod", "env"))
}
