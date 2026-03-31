package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateAlertRule__Setup(t *testing.T) {
	component := CreateAlertRule{}

	t.Run("title is required", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "title is required")
	})

	t.Run("required fields are validated", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"title": "High error rate",
			},
		})

		require.ErrorContains(t, err, "folderUID is required")
	})

	t.Run("valid configuration passes", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: validCreateAlertRuleConfiguration(),
		})

		require.NoError(t, err)
	})
}

func Test__CreateAlertRule__Configuration__UsesIntegrationResources(t *testing.T) {
	component := CreateAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "folderUID", resourceTypeFolder)
	assertIntegrationResourceField(t, fields, "dataSourceUid", resourceTypeDataSource)
}

func Test__CreateAlertRule__Execute(t *testing.T) {
	component := CreateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusCreated,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"High error rate"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: validCreateAlertRuleConfiguration(),
		HTTP:          httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://grafana.example.com",
			},
		},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	assert.True(t, execCtx.Finished)
	assert.True(t, execCtx.Passed)
	assert.Equal(t, "grafana.alertRule", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "true", httpContext.Requests[0].Header.Get("X-Disable-Provenance"))

	body := decodeJSONBody(t, httpContext.Requests[0].Body)
	assert.Equal(t, "High error rate", body["title"])
	assert.Equal(t, "folder-1", body["folderUID"])
	assert.Equal(t, "service-health", body["ruleGroup"])
	assert.Equal(t, "A", body["condition"])
	assert.Equal(t, "5m", body["for"])
	assert.Equal(t, "NoData", body["noDataState"])
	assert.Equal(t, "Alerting", body["execErrState"])
	assert.Equal(t, false, body["isPaused"])

	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	queryData, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "datasource-1", queryData["datasourceUid"])
	assert.Equal(t, "A", queryData["refId"])

	relativeTimeRange, ok := queryData["relativeTimeRange"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(defaultAlertRuleLookback), relativeTimeRange["from"])
	assert.Equal(t, float64(0), relativeTimeRange["to"])

	model, ok := queryData["model"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, `sum(rate(http_requests_total{status=~"5.."}[5m]))`, model["expr"])
	assert.Equal(t, `sum(rate(http_requests_total{status=~"5.."}[5m]))`, model["query"])

	labels, ok := body["labels"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "api", labels["service"])
	assert.Equal(t, "critical", labels["severity"])

	annotations, ok := body["annotations"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "High error rate detected", annotations["summary"])
}

func Test__GetAlertRule__Configuration__UsesIntegrationResource(t *testing.T) {
	component := GetAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "alertRuleUid", resourceTypeAlertRule)
}

func Test__GetAlertRule__Execute(t *testing.T) {
	component := GetAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"High error rate"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
		},
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://grafana.example.com",
			},
		},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	assert.True(t, execCtx.Finished)
	assert.True(t, execCtx.Passed)
	assert.Equal(t, "grafana.alertRule", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
}

func Test__UpdateAlertRule__Configuration__UsesIntegrationResources(t *testing.T) {
	component := UpdateAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "alertRuleUid", resourceTypeAlertRule)
	assertIntegrationResourceField(t, fields, "folderUID", resourceTypeFolder)
	assertIntegrationResourceField(t, fields, "dataSourceUid", resourceTypeDataSource)
}

func Test__UpdateAlertRule__Execute(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"uid":"rule-1",
					"title":"Old title",
					"folderUID":"folder-old",
					"ruleGroup":"old-group",
					"condition":"A",
					"data":[{"refId":"A","datasourceUid":"old-source"}],
					"labels":{"team":"ops"},
					"annotations":{"summary":"Old summary"}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"High error rate"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: mergeMaps(
			validCreateAlertRuleConfiguration(),
			map[string]any{
				"alertRuleUid": "rule-1",
				"isPaused":     true,
			},
		),
		HTTP: httpContext,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://grafana.example.com",
			},
		},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	assert.True(t, execCtx.Finished)
	assert.True(t, execCtx.Passed)
	assert.Equal(t, "grafana.alertRule", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
	assert.Equal(t, "true", httpContext.Requests[1].Header.Get("X-Disable-Provenance"))

	body := decodeJSONBody(t, httpContext.Requests[1].Body)
	assert.Equal(t, "rule-1", body["uid"])
	assert.Equal(t, "High error rate", body["title"])
	assert.Equal(t, "folder-1", body["folderUID"])
	assert.Equal(t, "service-health", body["ruleGroup"])
	assert.Equal(t, true, body["isPaused"])

	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	queryData, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "datasource-1", queryData["datasourceUid"])
}

func validCreateAlertRuleConfiguration() map[string]any {
	return map[string]any{
		"title":           "High error rate",
		"folderUID":       "folder-1",
		"ruleGroup":       "service-health",
		"dataSourceUid":   "datasource-1",
		"query":           `sum(rate(http_requests_total{status=~"5.."}[5m]))`,
		"lookbackSeconds": defaultAlertRuleLookback,
		"for":             "5m",
		"noDataState":     "NoData",
		"execErrState":    "Alerting",
		"labels": []any{
			map[string]any{
				"key":   "service",
				"value": "api",
			},
			map[string]any{
				"key":   "severity",
				"value": "critical",
			},
		},
		"annotations": []any{
			map[string]any{
				"key":   "summary",
				"value": "High error rate detected",
			},
		},
		"isPaused": false,
	}
}

func assertIntegrationResourceField(
	t *testing.T,
	fields []configuration.Field,
	name string,
	resourceType string,
) {
	t.Helper()

	var field *configuration.Field
	for i := range fields {
		if fields[i].Name == name {
			field = &fields[i]
			break
		}
	}

	require.NotNil(t, field)
	require.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
	require.NotNil(t, field.TypeOptions)
	require.NotNil(t, field.TypeOptions.Resource)
	require.Equal(t, resourceType, field.TypeOptions.Resource.Type)
}

func mergeMaps(maps ...map[string]any) map[string]any {
	merged := map[string]any{}
	for _, current := range maps {
		for key, value := range current {
			merged[key] = value
		}
	}

	return merged
}
