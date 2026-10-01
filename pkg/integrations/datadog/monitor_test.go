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
	assert.Equal(t, []string{intakeOwnerTag(integration.ID().String())}, body.Tags)
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
			jsonResponse(http.StatusOK, monitorJSON(8, "SuperPlane billing", billingMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, monitorJSON(7, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
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
			jsonResponse(http.StatusOK, monitorJSON(8, "SuperPlane billing", billingMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, monitorJSON(7, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
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

func Test__OnErrorTrackingAlert__Setup__LeavesCustomWebhookMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[{"id":125,"name":"SuperPlane checkout"}]}`),
			jsonResponse(http.StatusOK, monitorJSON(125, "SuperPlane checkout", checkoutMonitorQuery, "Alert\n\n@webhook-superplane", 0, false)),
			jsonResponse(http.StatusOK, monitorJSON(42, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
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
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodPut))
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
	post := monitorRequest(t, httpContext, http.MethodPost)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor", post.URL.String())
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "42", saved.MonitorID)
	assert.Equal(t, "checkout", saved.MonitorService)
}

func Test__OnErrorTrackingAlert__Setup__LeavesAnotherConnectionsMonitor(t *testing.T) {
	otherTag := intakeOwnerTag("11111111-1111-1111-1111-111111111111")
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{"monitors":[{"id":7,"name":"SuperPlane checkout"}]}`),
			jsonResponse(http.StatusOK, monitorJSON(7, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false, otherTag)),
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
	saved := metadata.Metadata.(OnErrorTrackingAlertMetadata)
	assert.Equal(t, "42", saved.MonitorID)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
	post := monitorRequest(t, httpContext, http.MethodPost)
	assert.Equal(t, []string{intakeOwnerTag(integration.ID().String())}, readMonitorBody(t, post).Tags)
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
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(9, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, `{}`),
		},
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
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(9, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
		},
	}
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
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
	assert.Len(t, requestsWithMethod(httpContext, http.MethodGet), 1)
}

func Test__Datadog__Cleanup__DeletesOnlyOwnedMonitors(t *testing.T) {
	integration := datadogIntegration()
	tag := intakeOwnerTag(integration.ID().String())
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, `{"monitors":[{"id":1,"name":"SuperPlane checkout"},{"id":3,"name":"New issue to review"}]}`),
			jsonResponse(http.StatusOK, monitorJSON(1, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false, tag)),
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, monitorJSON(3, "New issue to review", checkoutMonitorQuery, "Alert\n\n@webhook-superplane", 0, false)),
			jsonResponse(http.StatusOK, `{"monitors":[
				{"id":4,"name":"SuperPlane billing"},
				{"id":6,"name":"SuperPlane checkout"},
				{"id":2,"name":"Disk usage"},
				{"id":5,"name":"SuperPlane dashboard"}
			]}`),
			jsonResponse(http.StatusOK, monitorJSON(4, "SuperPlane billing", billingMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, monitorJSON(6, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false, "superplane_integration:11111111-1111-1111-1111-111111111111")),
			jsonResponse(http.StatusOK, monitorJSON(2, "Disk usage", "avg(last_5m):avg:system.disk.in_use{*} > 0.9", "disk", 0, false)),
			jsonResponse(http.StatusOK, monitorJSON(5, "SuperPlane dashboard", checkoutMonitorQuery, "Alert\n\n@webhook-superplane", 0, false)),
		},
	}

	err := (&Datadog{}).Cleanup(core.IntegrationCleanupContext{
		HTTP:        httpContext,
		Integration: integration,
	})

	require.NoError(t, err)
	var deleted []string
	for _, request := range httpContext.Requests {
		if request.Method == http.MethodDelete && strings.Contains(request.URL.Path, "/api/v1/monitor/") {
			deleted = append(deleted, request.URL.Path)
		}
	}
	assert.Equal(t, []string{"/api/v1/monitor/1", "/api/v1/monitor/4"}, deleted)
	assert.NotContains(t, deleted, "/api/v1/monitor/2")
	assert.NotContains(t, deleted, "/api/v1/monitor/3")
	assert.NotContains(t, deleted, "/api/v1/monitor/5")
	assert.NotContains(t, deleted, "/api/v1/monitor/6")
}

func Test__Datadog__Cleanup__ReturnsMonitorDeleteError(t *testing.T) {
	integration := datadogIntegration()
	tag := intakeOwnerTag(integration.ID().String())
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, `{}`),
			jsonResponse(http.StatusOK, `{"monitors":[{"id":1,"name":"SuperPlane checkout"}]}`),
			jsonResponse(http.StatusOK, monitorJSON(1, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false, tag)),
			jsonResponse(http.StatusInternalServerError, `{"errors":["unavailable"]}`),
		},
	}

	err := (&Datadog{}).Cleanup(core.IntegrationCleanupContext{
		HTTP:        httpContext,
		Integration: integration,
	})

	require.Error(t, err)
	assert.NotContains(t, err.Error(), "monitors_write")
}

func Test__ReleaseRetiredIntakeMonitor__DeletesOwnedMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(9, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
			jsonResponse(http.StatusOK, `{}`),
		},
	}

	err := ReleaseRetiredIntakeMonitor(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   datadogIntegration(),
		Metadata: &contexts.MetadataContext{Metadata: map[string]any{
			"monitorId":      "9",
			"monitorService": "checkout",
		}},
	})

	require.NoError(t, err)
	deleted := monitorRequest(t, httpContext, http.MethodDelete)
	assert.Equal(t, "https://api.datadoghq.eu/api/v1/monitor/9", deleted.URL.String())
}

func Test__ReleaseRetiredIntakeMonitor__KeepsSharedMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(9, "SuperPlane checkout", checkoutMonitorQuery, intakeMonitorMessage, 0, false)),
		},
	}
	integration := datadogIntegration()
	integration.NodeConfigurations = []any{map[string]any{"service": "checkout"}}

	err := ReleaseRetiredIntakeMonitor(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   integration,
		Metadata: &contexts.MetadataContext{Metadata: map[string]any{
			"monitorId":      "9",
			"monitorService": "checkout",
		}},
	})

	require.NoError(t, err)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
}

func Test__ReleaseRetiredIntakeMonitor__LeavesCustomMonitor(t *testing.T) {
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, monitorJSON(9, "New issue to review", checkoutMonitorQuery, "Alert\n\n@webhook-superplane", 0, false)),
		},
	}

	err := ReleaseRetiredIntakeMonitor(core.TriggerContext{
		Configuration: map[string]any{"service": "checkout"},
		HTTP:          httpContext,
		Integration:   datadogIntegration(),
		Metadata: &contexts.MetadataContext{Metadata: map[string]any{
			"monitorId":      "9",
			"monitorService": "checkout",
		}},
	})

	require.NoError(t, err)
	assert.Empty(t, requestsWithMethod(httpContext, http.MethodDelete))
}

func Test__Client__SearchMonitors__ReadsEveryPage(t *testing.T) {
	const fullPages = 21
	responses := make([]*http.Response, 0, fullPages+1)
	for page := 0; page < fullPages; page++ {
		responses = append(responses, jsonResponse(http.StatusOK, monitorSearchPage(monitorPageSize, page*monitorPageSize)))
	}
	responses = append(responses, jsonResponse(http.StatusOK, monitorSearchPage(1, fullPages*monitorPageSize)))
	httpContext := &contexts.HTTPContext{Responses: responses}
	client, err := NewClient(httpContext, datadogIntegration())
	require.NoError(t, err)

	monitors, err := client.SearchMonitors("tag:superplane_integration:test")

	require.NoError(t, err)
	assert.Len(t, monitors, fullPages*monitorPageSize+1)
	assert.Len(t, httpContext.Requests, fullPages+1)
	assert.Equal(t, int64(0), monitors[0].ID)
	assert.Equal(t, int64(fullPages*monitorPageSize), monitors[len(monitors)-1].ID)
}

func Test__Client__SearchMonitors__StopsWhenPageRepeats(t *testing.T) {
	page := monitorSearchPage(monitorPageSize, 0)
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			jsonResponse(http.StatusOK, page),
			jsonResponse(http.StatusOK, page),
		},
	}
	client, err := NewClient(httpContext, datadogIntegration())
	require.NoError(t, err)

	_, err = client.SearchMonitors("title:SuperPlane")

	require.ErrorContains(t, err, "repeated page")
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

func monitorJSON(id int64, name, query, message string, delay int, simple bool, tags ...string) string {
	if tags == nil {
		tags = []string{}
	}
	encodedTags, err := json.Marshal(tags)
	if err != nil {
		encodedTags = []byte("[]")
	}
	return fmt.Sprintf(
		`{"id":%d,"name":%q,"type":"error-tracking alert","query":%q,"message":%q,"tags":%s,"options":{"groupby_simple_monitor":%t,"new_host_delay":%d,"thresholds":{"critical":0}}}`,
		id, name, query, message, encodedTags, simple, delay,
	)
}

func monitorSearchPage(count, startID int) string {
	monitors := make([]Monitor, 0, count)
	for i := 0; i < count; i++ {
		monitors = append(monitors, Monitor{ID: int64(startID + i), Name: fmt.Sprintf("SuperPlane %d", startID+i)})
	}
	body, err := json.Marshal(struct {
		Monitors []Monitor `json:"monitors"`
	}{Monitors: monitors})
	if err != nil {
		return `{"monitors":[]}`
	}
	return string(body)
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
