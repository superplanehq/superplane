package datadog

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

const (
	checkoutMonitorQuery = `error-tracking("service:checkout").source("all").new().rollup("count").by("issue.id").last("1d") > 0`
	billingMonitorQuery  = `error-tracking("service:billing").source("all").new().rollup("count").by("issue.id").last("1d") > 0`
	intakeMonitorMessage = "A new error tracking issue was detected.\n\n@webhook-superplane"
)

func Test__OnErrorTrackingAlert__Setup__CreatesMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, monitorJSON(42, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
		},
	}
	metadata := &contexts.MetadataContext{}
	integration := datadogIntegration()

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata:      metadata,
	})

	require.NoError(t, err)
	require.Len(t, integration.Subscriptions, 1)
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "42", saved.MonitorID)
	assert.Equal(t, "checkout", saved.MonitorService)
	assert.NotEmpty(t, saved.SubscriptionID)

	post := monitorRequest(t, httpContext, http.MethodPost)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor", post.URL.String())
	body := readMonitorBody(t, post)
	assert.Equal(t, "SuperPlane checkout", body.Name)
	assert.Equal(t, errorTrackingMonitorType, body.Type)
	assert.Equal(t, checkoutMonitorQuery, body.Query)
	assert.Equal(t, intakeMonitorMessage, body.Message)
	assert.False(t, body.Options.GroupbySimpleMonitor)
	assert.Equal(t, 0, body.Options.NewHostDelay)
	assert.Equal(t, float64(0), body.Options.Thresholds.Critical)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPut))
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
}

func Test__OnErrorTrackingAlert__Setup__SkipsWriteWhenMonitorMatches(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(42, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
		},
	}
	metadata := &contexts.MetadataContext{
		Metadata: map[string]any{
			"subscriptionId": "sub-1",
			"monitorId":      "42",
			"monitorService": "checkout",
		},
	}

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   datadogIntegration(),
		Metadata:      metadata,
	})

	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor/42", httpContext.Requests[0].URL.String())
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPut))
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPost))
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))

	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "42", saved.MonitorID)
	assert.Equal(t, "checkout", saved.MonitorService)
}

func Test__OnErrorTrackingAlert__Setup__ServiceChangeDeletesUnusedMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, monitorJSON(8, "SuperPlane billing", billingMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, `{}`),
		},
	}
	integration := datadogIntegration()
	metadata := &contexts.MetadataContext{
		Metadata: map[string]any{
			"subscriptionId": "sub-1",
			"monitorId":      "7",
			"monitorService": "checkout",
		},
	}

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "billing"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata:      metadata,
	})

	require.NoError(t, err)
	post := monitorRequest(t, httpContext, http.MethodPost)
	assert.Equal(t, billingMonitorQuery, readMonitorBody(t, post).Query)
	deleted := monitorRequest(t, httpContext, http.MethodDelete)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor/7", deleted.URL.String())
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "8", saved.MonitorID)
	assert.Equal(t, "billing", saved.MonitorService)
}

func Test__OnErrorTrackingAlert__Setup__ServiceChangeKeepsSharedMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, monitorJSON(8, "SuperPlane billing", billingMonitorQuery, intakeMonitorMessage, 0, false)),
		},
	}
	integration := datadogIntegration()
	integration.NodeConfigurations = []any{map[string]any{"service": "checkout"}}
	metadata := &contexts.MetadataContext{
		Metadata: map[string]any{
			"subscriptionId": "sub-1",
			"monitorId":      "7",
			"monitorService": "checkout",
		},
	}

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "billing"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata:      metadata,
	})

	require.NoError(t, err)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "8", saved.MonitorID)
	assert.Equal(t, "billing", saved.MonitorService)
}

func Test__OnErrorTrackingAlert__Setup__AdoptsExistingWebhookMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[]}`),
			jsonResponse(http.StatusOK, `{"monitors":[{"id":125,"name":"New issue to review"}]}`),
			jsonResponse(http.StatusOK, monitorJSON(125, "New issue to review", checkoutMonitorQuery, "Alert\n\n@webhook-superplane", 0, false)),
		},
	}
	metadata := &contexts.MetadataContext{}

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   datadogIntegration(),
		Metadata:      metadata,
	})

	require.NoError(t, err)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPost))
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPut))
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "125", saved.MonitorID)
	assert.Equal(t, "checkout", saved.MonitorService)
}

func Test__OnErrorTrackingAlert__Setup__MissingMonitorsWrite(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusForbidden, `{"errors":["Forbidden"]}`),
		},
	}
	metadata := &contexts.MetadataContext{}

	err := (&OnErrorTrackingAlert{}).Setup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   datadogIntegration(),
		Metadata:      metadata,
	})

	require.ErrorContains(t, err, "monitors_write")
	assert.Nil(t, metadata.Metadata)
}

func Test__OnErrorTrackingAlert__Cleanup__DeletesLastMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{jsonResponse(http.StatusOK, `{}`)},
	}
	integration := datadogIntegration()
	integration.NodeConfigurations = []any{map[string]any{"service": "checkout"}}

	err := (&OnErrorTrackingAlert{}).Cleanup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata: &contexts.MetadataContext{Metadata: map[string]any{
			"monitorId":      "9",
			"monitorService": "checkout",
		}},
	})

	require.NoError(t, err)
	deleted := monitorRequest(t, httpContext, http.MethodDelete)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor/9", deleted.URL.String())
}

func Test__OnErrorTrackingAlert__Cleanup__KeepsSharedMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	integration := datadogIntegration()
	integration.NodeConfigurations = []any{
		map[string]any{"service": "checkout"},
		map[string]any{"service": "checkout"},
	}

	err := (&OnErrorTrackingAlert{}).Cleanup(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata: &contexts.MetadataContext{Metadata: map[string]any{
			"monitorId":      "9",
			"monitorService": "checkout",
		}},
	})

	require.NoError(t, err)
	assert.Empty(t, httpContext.Requests)
}

func Test__Datadog__Cleanup__DeletesIntakeMonitors(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, `[
				{"id":1,"name":"SuperPlane checkout","type":"error-tracking alert","query":"q","message":"owned","options":{"groupby_simple_monitor":false,"new_host_delay":0,"thresholds":{"critical":0}}},
				{"id":2,"name":"Disk usage","type":"query alert","query":"q","message":"disk","options":{"groupby_simple_monitor":false,"new_host_delay":0,"thresholds":{"critical":0}}},
				{"id":3,"name":"New issue to review","type":"error-tracking alert","query":"q","message":"Alert @webhook-superplane","options":{"groupby_simple_monitor":false,"new_host_delay":0,"thresholds":{"critical":0}}}
			]`),
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, `{}`),
		},
	}

	err := (&Datadog{}).Cleanup(core.IntegrationCleanupContext{
		HTTP:        httpContext,
		Integration: datadogIntegration(),
	})

	require.NoError(t, err)
	var deleted []string
	for _, request := range httpContext.Requests {
		if request.Method == http.MethodDelete && strings.Contains(request.URL.Path, "/api/v1/monitor/") {
			deleted = append(deleted, request.URL.Path)
		}
	}
	assert.Equal(t, []string{"/api/v1/monitor/1", "/api/v1/monitor/3"}, deleted)
	assert.NotContains(t, requestPaths(httpContext), "/api/v1/monitor/2")
}

func datadogIntegration() *contexts.IntegrationContext {
	return &contexts.IntegrationContext{
		Configuration: map[string]any{
			"site":   "datadoghq.eu",
			"apiKey": "test-api-key",
			"appKey": "test-app-key",
		},
	}
}

func jsonResponse(status int, body string) *http.Response {
	return &http.Response{
		StatusCode: status,
		Body:       io.NopCloser(strings.NewReader(body)),
	}
}

func monitorJSON(id int64, name, query, message string, delay int, simple bool) string {
	return fmt.Sprintf(
		`{"id":%d,"name":%q,"type":"error-tracking alert","query":%q,"message":%q,"options":{"groupby_simple_monitor":%t,"new_host_delay":%d,"thresholds":{"critical":0}}}`,
		id, name, query, message, simple, delay,
	)
}

func monitorRequest(t *testing.T, httpContext *contexts.HTTPContext, method string) *http.Request {
	t.Helper()
	matches := requestsWithMethod(httpContext, method)
	require.Len(t, matches, 1)
	return matches[0]
}

func requestsWithMethod(httpContext *contexts.HTTPContext, method string) []*http.Request {
	var matches []*http.Request
	for _, request := range httpContext.Requests {
		if request.Method == method {
			matches = append(matches, request)
		}
	}
	return matches
}

func requestPaths(httpContext *contexts.HTTPContext) []string {
	paths := make([]string, 0, len(httpContext.Requests))
	for _, request := range httpContext.Requests {
		paths = append(paths, request.URL.Path)
	}
	return paths
}

func readMonitorBody(t *testing.T, request *http.Request) Monitor {
	t.Helper()
	raw, err := io.ReadAll(request.Body)
	require.NoError(t, err)
	var monitor Monitor
	require.NoError(t, json.Unmarshal(raw, &monitor))
	return monitor
}
