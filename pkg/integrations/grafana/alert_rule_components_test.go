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
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`[{"uid":"folder-1","title":"Infrastructure"}]`)),
				},
			},
		}
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: validCreateAlertRuleConfiguration(),
			HTTP:          httpContext,
			Metadata:      metadataCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "token123",
					"baseURL":  "https://grafana.example.com",
				},
			},
		})

		require.NoError(t, err)
		assert.Equal(t, AlertRuleNodeMetadata{FolderTitle: "Infrastructure"}, metadataCtx.Metadata)
	})
}

func Test__CreateAlertRule__Configuration__UsesIntegrationResources(t *testing.T) {
	component := CreateAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "folderUID", resourceTypeFolder)
	assertIntegrationResourceField(t, fields, "dataSourceUid", resourceTypeDataSource)
}

func Test__CreateAlertRule__Configuration__DoesNotPrefillUserFields(t *testing.T) {
	component := CreateAlertRule{}
	fields := component.Configuration()

	assertFieldHasNoDefault(t, fields, "title")
	assertFieldHasNoDefault(t, fields, "ruleGroup")
	assertFieldHasNoDefault(t, fields, "query")
	assertFieldHasNoDefault(t, fields, "lookbackSeconds")
	assertFieldHasNoDefault(t, fields, "for")
	assertFieldHasNoDefault(t, fields, "noDataState")
	assertFieldHasNoDefault(t, fields, "execErrState")
	assertFieldHasNoDefault(t, fields, "isPaused")

	var pausedField *configuration.Field
	for i := range fields {
		if fields[i].Name == "isPaused" {
			pausedField = &fields[i]
			break
		}
	}

	require.NotNil(t, pausedField)
	assert.False(t, pausedField.Required)
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
	assert.Equal(t, float64(300), relativeTimeRange["from"])
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

func Test__ListAlertRules__Execute(t *testing.T) {
	component := ListAlertRules{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`[
					{"uid":"rule-1","title":"High error rate"},
					{"uid":"rule-2","title":"High latency"}
				]`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
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
	assert.Equal(t, "grafana.alertRules", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)

	emittedPayload, ok := execCtx.Payloads[0].(map[string]any)
	require.True(t, ok)

	response, ok := emittedPayload["data"].(ListAlertRulesOutput)
	if ok {
		require.Len(t, response.AlertRules, 2)
		assert.Equal(t, "rule-1", response.AlertRules[0].UID)
		assert.Equal(t, "High error rate", response.AlertRules[0].Title)
		return
	}

	responseData, ok := emittedPayload["data"].(map[string]any)
	require.True(t, ok)
	alertRules, ok := responseData["alertRules"].([]any)
	require.True(t, ok)
	require.Len(t, alertRules, 2)

	firstRule, ok := alertRules[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "rule-1", firstRule["uid"])
	assert.Equal(t, "High error rate", firstRule["title"])
}

func Test__UpdateAlertRule__Configuration__UsesIntegrationResources(t *testing.T) {
	component := UpdateAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "alertRuleUid", resourceTypeAlertRule)
	assertIntegrationResourceField(t, fields, "folderUID", resourceTypeFolder)
	assertIntegrationResourceField(t, fields, "dataSourceUid", resourceTypeDataSource)
}

func Test__UpdateAlertRule__Configuration__AllowsPartialUpdates(t *testing.T) {
	component := UpdateAlertRule{}
	fields := component.Configuration()

	assertFieldRequired(t, fields, "alertRuleUid", true)
	assertFieldRequired(t, fields, "title", false)
	assertFieldRequired(t, fields, "folderUID", false)
	assertFieldRequired(t, fields, "ruleGroup", false)
	assertFieldRequired(t, fields, "dataSourceUid", false)
	assertFieldRequired(t, fields, "query", false)
	assertFieldRequired(t, fields, "lookbackSeconds", false)
	assertFieldRequired(t, fields, "for", false)
	assertFieldRequired(t, fields, "noDataState", false)
	assertFieldRequired(t, fields, "execErrState", false)
	assertFieldRequired(t, fields, "isPaused", false)
}

func Test__UpdateAlertRule__Setup__StoresAlertRuleTitleMetadata(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"CPU saturation"}`)),
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
			"isPaused":     true,
		},
		HTTP:     httpContext,
		Metadata: metadataCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AlertRuleNodeMetadata{AlertRuleTitle: "CPU saturation"}, metadataCtx.Metadata)
}

func Test__UpdateAlertRule__Execute(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
						"id":12,
						"uid":"rule-1",
						"orgID":1,
						"title":"Old title",
						"folderUID":"folder-old",
						"ruleGroup":"old-group",
						"condition":"A",
					"data":[{
						"refId":"A",
						"queryType":"",
						"datasourceUid":"old-source",
						"relativeTimeRange":{"from":120,"to":0},
						"model":{
							"expr":"sum(rate(errors_total[2m]))",
							"query":"sum(rate(errors_total[2m]))",
							"intervalMs":1000,
							"maxDataPoints":43200,
							"refId":"A"
						}
					}],
					"for":"2m",
					"noDataState":"OK",
						"execErrState":"KeepLast",
						"isPaused":false,
						"labels":{"team":"ops"},
						"annotations":{"summary":"Old summary"},
						"dashboardUid":"dashboard-1",
						"panelId":4,
						"updated":"2026-04-01T10:15:00Z",
						"provenance":"api"
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
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
			"title":        "High error rate",
			"isPaused":     true,
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
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
	assert.Equal(t, "", httpContext.Requests[1].Header.Get("X-Disable-Provenance"))

	body := decodeJSONBody(t, httpContext.Requests[1].Body)
	assert.Equal(t, float64(12), body["id"])
	assert.Equal(t, "rule-1", body["uid"])
	assert.Equal(t, float64(1), body["orgID"])
	assert.Equal(t, float64(1), body["orgId"])
	assert.Equal(t, "High error rate", body["title"])
	assert.Equal(t, "folder-old", body["folderUID"])
	assert.Equal(t, "old-group", body["ruleGroup"])
	assert.Equal(t, "dashboard-1", body["dashboardUid"])
	assert.Equal(t, float64(4), body["panelId"])
	assert.Equal(t, true, body["isPaused"])
	assert.Equal(t, "2m", body["for"])
	assert.Equal(t, "OK", body["noDataState"])
	assert.Equal(t, "KeepLast", body["execErrState"])

	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	queryData, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "old-source", queryData["datasourceUid"])

	model, ok := queryData["model"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "sum(rate(errors_total[2m]))", model["expr"])
	assert.Equal(t, "sum(rate(errors_total[2m]))", model["query"])

	_, hasUpdated := body["updated"]
	assert.False(t, hasUpdated)
	_, hasProvenance := body["provenance"]
	assert.False(t, hasProvenance)
	_, hasID := body["id"]
	assert.True(t, hasID)
}

func Test__UpdateAlertRule__Execute__RejectsFileProvisionedRules(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"uid":"rule-1",
					"title":"Old title",
					"provenance":"file"
				}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
			"title":        "High error rate",
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

	require.ErrorContains(
		t,
		err,
		"file-provisioned Grafana alert rules cannot be updated via the provisioning API",
	)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
}

func Test__UpdateAlertRule__Execute__OmitsDisableProvenanceHeaderForApiProvisionedRules(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"id":1,
					"uid":"rule-1",
					"orgID":1,
					"title":"Old title",
					"folderUID":"folder-old",
					"ruleGroup":"old-group",
					"condition":"A",
					"data":[{"refId":"A","queryType":"","datasourceUid":"ds-1","relativeTimeRange":{"from":300,"to":0},"model":{"expr":"up","refId":"A"}}],
					"for":"5m",
					"noDataState":"NoData",
					"execErrState":"Alerting",
					"isPaused":false,
					"provenance":"api"
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"New title"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
			"title":        "New title",
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
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodPut, httpContext.Requests[1].Method)
	assert.Equal(t, "", httpContext.Requests[1].Header.Get("X-Disable-Provenance"))
}

func Test__UpdateAlertRule__Execute__MergesProvidedQueryFields(t *testing.T) {
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
					"data":[{
						"refId":"A",
						"queryType":"",
						"datasourceUid":"old-source",
						"relativeTimeRange":{"from":120,"to":0},
						"model":{
							"datasource":{"type":"prometheus","uid":"old-source"},
							"expr":"sum(rate(errors_total[2m]))",
							"query":"sum(rate(errors_total[2m]))",
							"intervalMs":1000,
							"maxDataPoints":43200,
							"refId":"A"
						}
					}]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"Old title"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertRuleUid":    "rule-1",
			"dataSourceUid":   "datasource-1",
			"query":           `sum(rate(http_requests_total{status=~"5.."}[5m]))`,
			"lookbackSeconds": 300,
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
	require.Len(t, httpContext.Requests, 2)

	body := decodeJSONBody(t, httpContext.Requests[1].Body)
	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	queryData, ok := data[0].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "datasource-1", queryData["datasourceUid"])

	relativeTimeRange, ok := queryData["relativeTimeRange"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, float64(300), relativeTimeRange["from"])
	assert.Equal(t, float64(0), relativeTimeRange["to"])

	model, ok := queryData["model"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, `sum(rate(http_requests_total{status=~"5.."}[5m]))`, model["expr"])
	assert.Equal(t, `sum(rate(http_requests_total{status=~"5.."}[5m]))`, model["query"])
	assert.Equal(t, float64(1000), model["intervalMs"])
	assert.Equal(t, float64(43200), model["maxDataPoints"])

	datasource, ok := model["datasource"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "datasource-1", datasource["uid"])
}

func Test__UpdateAlertRule__Execute__UpdatesExpressionQueryModels(t *testing.T) {
	component := UpdateAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"uid":"rule-1",
					"orgID":1,
					"title":"Old title",
					"folderUID":"folder-old",
					"ruleGroup":"old-group",
					"condition":"A",
					"data":[{
						"refId":"A",
						"queryType":"",
						"datasourceUid":"__expr__",
						"relativeTimeRange":{"from":0,"to":0},
						"model":{
							"datasource":{"type":"__expr__","uid":"__expr__"},
							"expression":"1 == 1",
							"hide":false,
							"intervalMs":1000,
							"maxDataPoints":43200,
							"refId":"A",
							"type":"math"
						}
					}]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"Old title"}`)),
			},
		},
	}

	execCtx := &contexts.ExecutionStateContext{}
	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
			"query":        "2 == 2",
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
	require.Len(t, httpContext.Requests, 2)

	body := decodeJSONBody(t, httpContext.Requests[1].Body)
	data, ok := body["data"].([]any)
	require.True(t, ok)
	require.Len(t, data, 1)

	queryData, ok := data[0].(map[string]any)
	require.True(t, ok)

	model, ok := queryData["model"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "2 == 2", model["expression"])
}

func Test__DeleteAlertRule__Configuration__UsesIntegrationResource(t *testing.T) {
	component := DeleteAlertRule{}
	fields := component.Configuration()

	assertIntegrationResourceField(t, fields, "alertRuleUid", resourceTypeAlertRule)
}

func Test__DeleteAlertRule__Setup__StoresAlertRuleTitleMetadata(t *testing.T) {
	component := DeleteAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"CPU saturation"}`)),
			},
		},
	}
	metadataCtx := &contexts.MetadataContext{}

	err := component.Setup(core.SetupContext{
		Configuration: map[string]any{
			"alertRuleUid": "rule-1",
		},
		HTTP:     httpContext,
		Metadata: metadataCtx,
		Integration: &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "token123",
				"baseURL":  "https://grafana.example.com",
			},
		},
	})

	require.NoError(t, err)
	assert.Equal(t, AlertRuleNodeMetadata{AlertRuleTitle: "CPU saturation"}, metadataCtx.Metadata)
}

func Test__DeleteAlertRule__Execute(t *testing.T) {
	component := DeleteAlertRule{}

	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"uid":"rule-1","title":"High error rate"}`)),
			},
			{
				StatusCode: http.StatusNoContent,
				Body:       io.NopCloser(strings.NewReader("")),
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
	assert.Equal(t, "grafana.alertRuleDeleted", execCtx.Type)
	require.Len(t, execCtx.Payloads, 1)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)

	emittedPayload, ok := execCtx.Payloads[0].(map[string]any)
	require.True(t, ok)

	response, ok := emittedPayload["data"].(DeleteAlertRuleOutput)
	if ok {
		assert.Equal(t, "rule-1", response.UID)
		assert.Equal(t, "High error rate", response.Title)
		assert.True(t, response.Deleted)
		return
	}

	responseData, ok := emittedPayload["data"].(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "rule-1", responseData["uid"])
	assert.Equal(t, "High error rate", responseData["title"])
	assert.Equal(t, true, responseData["deleted"])
}

func validCreateAlertRuleConfiguration() map[string]any {
	return map[string]any{
		"title":           "High error rate",
		"folderUID":       "folder-1",
		"ruleGroup":       "service-health",
		"dataSourceUid":   "datasource-1",
		"query":           `sum(rate(http_requests_total{status=~"5.."}[5m]))`,
		"lookbackSeconds": 300,
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

func assertFieldHasNoDefault(t *testing.T, fields []configuration.Field, name string) {
	t.Helper()

	var field *configuration.Field
	for i := range fields {
		if fields[i].Name == name {
			field = &fields[i]
			break
		}
	}

	require.NotNil(t, field)
	assert.Nil(t, field.Default)
}

func assertFieldRequired(t *testing.T, fields []configuration.Field, name string, required bool) {
	t.Helper()

	var field *configuration.Field
	for i := range fields {
		if fields[i].Name == name {
			field = &fields[i]
			break
		}
	}

	require.NotNil(t, field)
	assert.Equal(t, required, field.Required)
}
