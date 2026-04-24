package grafana

import (
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IncidentComponents__Configuration__incidentIsIntegrationResource(t *testing.T) {
	components := []struct {
		name   string
		fields []configuration.Field
	}{
		{name: "get", fields: (&GetIncident{}).Configuration()},
		{name: "update", fields: (&UpdateIncident{}).Configuration()},
		{name: "resolve", fields: (&ResolveIncident{}).Configuration()},
		{name: "activity", fields: (&AddIncidentActivity{}).Configuration()},
	}

	for _, tc := range components {
		t.Run(tc.name, func(t *testing.T) {
			require.NotEmpty(t, tc.fields)
			field := tc.fields[0]
			require.Equal(t, "incident", field.Name)
			require.Equal(t, configuration.FieldTypeIntegrationResource, field.Type)
			require.NotNil(t, field.TypeOptions)
			require.NotNil(t, field.TypeOptions.Resource)
			require.Equal(t, resourceTypeIncident, field.TypeOptions.Resource.Type)
		})
	}
}

func Test__IncidentComponents__Configuration__severityIsIntegrationResource(t *testing.T) {
	declareFields := (&DeclareIncident{}).Configuration()
	require.Equal(t, "severity", declareFields[1].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, declareFields[1].Type)
	require.NotNil(t, declareFields[1].TypeOptions)
	require.NotNil(t, declareFields[1].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncidentSeverity, declareFields[1].TypeOptions.Resource.Type)

	updateFields := (&UpdateIncident{}).Configuration()
	require.Equal(t, "severity", updateFields[2].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, updateFields[2].Type)
	require.NotNil(t, updateFields[2].TypeOptions)
	require.NotNil(t, updateFields[2].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncidentSeverity, updateFields[2].TypeOptions.Resource.Type)
	require.Equal(t, "labels", updateFields[3].Name)
	require.Equal(t, configuration.FieldTypeList, updateFields[3].Type)

	drillFields := (&DeclareDrill{}).Configuration()
	require.Equal(t, "status", drillFields[4].Name)
	require.Equal(t, configuration.FieldTypeSelect, drillFields[4].Type)
	require.Equal(t, "debriefStatus", drillFields[5].Name)
	require.Equal(t, configuration.FieldTypeSelect, drillFields[5].Type)
	require.NotNil(t, drillFields[5].TypeOptions)
	require.NotNil(t, drillFields[5].TypeOptions.Select)
	require.Equal(t, grafanaIncidentDebriefStatusOptions, drillFields[5].TypeOptions.Select.Options)
	require.Equal(t, "startTime", drillFields[6].Name)
	require.Equal(t, configuration.FieldTypeDateTime, drillFields[6].Type)
}

func Test__IncidentComponents__ExampleOutput__DeclareDrillMarksIncidentAsDrill(t *testing.T) {
	declareDrillOutput := (&DeclareDrill{}).ExampleOutput()
	declareDrillData, ok := declareDrillOutput["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, declareDrillData["isDrill"])

	declareIncidentOutput := (&DeclareIncident{}).ExampleOutput()
	declareIncidentData, ok := declareIncidentOutput["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, declareIncidentData["isDrill"])
}

func Test__Grafana__ListResources__IncidentSeverities(t *testing.T) {
	resources, err := (&Grafana{}).ListResources(resourceTypeIncidentSeverity, core.ListResourcesContext{})

	require.NoError(t, err)
	require.Equal(t, []core.IntegrationResource{
		{Type: resourceTypeIncidentSeverity, Name: "Pending", ID: "pending"},
		{Type: resourceTypeIncidentSeverity, Name: "Critical", ID: "critical"},
		{Type: resourceTypeIncidentSeverity, Name: "Major", ID: "major"},
		{Type: resourceTypeIncidentSeverity, Name: "Minor", ID: "minor"},
	}, resources)
}

func Test__DeclareIncident__Setup__RejectsUnknownSeverity(t *testing.T) {
	err := (&DeclareIncident{}).Setup(core.SetupContext{
		Configuration: map[string]any{
			"title":    "API latency",
			"severity": "high",
		},
	})

	require.ErrorContains(t, err, "severity must be one of")
}

func Test__DeclareIncident__Setup__RejectsUnknownDebriefStatus(t *testing.T) {
	err := (&DeclareIncident{}).Setup(core.SetupContext{
		Configuration: map[string]any{
			"title":         "API latency",
			"severity":      "major",
			"debriefStatus": "haha",
		},
	})

	require.ErrorContains(t, err, "debriefStatus must be one of")
}

func Test__DeclareIncident__Execute(t *testing.T) {
	component := &DeclareIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"API latency","severity":"minor","status":"active"}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"fields":[{"uuid":"field-1","slug":"debrief_status","type":"single-select","selectoptions":[{"uuid":"option-1","value":"not_started","label":"Not started"},{"uuid":"option-2","value":"in_progress","label":"In progress"}]}]
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"title":         "API latency",
			"severity":      "minor",
			"description":   "Initial diagnosis",
			"labels":        []any{"api", "prod"},
			"status":        "resolved",
			"debriefStatus": "in_progress",
			"startTime":     "2026-04-20T10:00:00Z",
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Equal(t, "grafana.incident.declared", execCtx.Type)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "Initial diagnosis", payload["initialStatusUpdate"])
	require.Equal(t, "resolved", payload["status"])
	require.Equal(t, "incident", payload["roomPrefix"])

	require.Len(t, execCtx.Payloads, 1)
	emitted := execCtx.Payloads[0].(map[string]any)
	out := emitted["data"].(*Incident)
	require.Equal(t, "incident-123", out.IncidentID)
	require.Equal(t, "API latency", out.Title)
	require.Equal(t, "minor", out.Severity)
	require.Equal(t, "active", out.Status)

	body, err = io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "incidentStart", payload["activityItemKind"])
	require.Equal(t, "incidentStart", payload["eventName"])
	require.Equal(t, "2026-04-20T10:00:00Z", payload["eventTime"])

	body, err = io.ReadAll(httpCtx.Requests[3].Body)
	require.NoError(t, err)
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "field-1", payload["fieldUUID"])
	require.Equal(t, "incident", payload["targetKind"])
	require.Equal(t, "incident-123", payload["targetID"])
	require.Equal(t, "option-2", payload["value"])
}

func Test__DeclareDrill__Execute(t *testing.T) {
	component := &DeclareDrill{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-456","title":"Game day","severity":"minor","status":"active","isDrill":true}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"title":    "Game day",
			"severity": "minor",
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, true, payload["isDrill"])
}

func Test__UpdateIncident__Execute__UpdatesProvidedFields(t *testing.T) {
	component := &UpdateIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"New title","severity":"minor","status":"active"}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"New title","severity":"major","status":"active"}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"title":    "New title",
			"severity": "major",
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Equal(t, "grafana.incident.updated", execCtx.Type)
	require.Len(t, httpCtx.Requests, 2)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateTitle", httpCtx.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateSeverity", httpCtx.Requests[1].URL.Path)

	require.Len(t, execCtx.Payloads, 1)
	emitted := execCtx.Payloads[0].(map[string]any)
	out := emitted["data"].(*Incident)
	require.Equal(t, "incident-123", out.IncidentID)
	require.Equal(t, "New title", out.Title)
	require.Equal(t, "major", out.Severity)
}

func Test__UpdateIncident__Execute__CanSetIsDrillFalse(t *testing.T) {
	component := &UpdateIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"API latency","severity":"minor","status":"active","isDrill":false}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"isDrill":  false,
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Len(t, httpCtx.Requests, 1)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateIncidentIsDrill", httpCtx.Requests[0].URL.Path)

	body, err := io.ReadAll(httpCtx.Requests[0].Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, false, payload["isDrill"])
}

func Test__UpdateIncident__Execute__AddsLabels(t *testing.T) {
	component := &UpdateIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"field":{"slug":"tags"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"API latency","severity":"minor","status":"active","labels":[{"label":"prod"}]}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"labels":   []any{"prod"},
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Equal(t, "grafana.incident.updated", execCtx.Type)
	require.Len(t, httpCtx.Requests, 2)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpCtx.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpCtx.Requests[1].URL.Path)

	body, err := io.ReadAll(httpCtx.Requests[1].Body)
	require.NoError(t, err)
	var payload map[string]any
	require.NoError(t, json.Unmarshal(body, &payload))
	require.Equal(t, "incident-123", payload["incidentID"])
	require.Equal(t, map[string]any{"key": "tags", "label": "prod"}, payload["label"])
}

func Test__UpdateIncident__Setup__RejectsEmptyLabels(t *testing.T) {
	err := (&UpdateIncident{}).Setup(core.SetupContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"labels":   []any{"  "},
		},
	})

	require.ErrorContains(t, err, "labels must include at least one non-empty label")
}

func Test__ResolveIncident__Execute__AddsSummaryThenResolves(t *testing.T) {
	component := &ResolveIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"activityItem":{"activityItemID":"activity-123","incidentID":"incident-123","activityKind":"userNote","body":"Fixed"}
				}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{"incidentID":"incident-123","title":"API latency","severity":"minor","status":"resolved"}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"summary":  "Fixed",
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Equal(t, "grafana.incident.resolved", execCtx.Type)
	require.Len(t, httpCtx.Requests, 2)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/ActivityService.AddActivity", httpCtx.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateStatus", httpCtx.Requests[1].URL.Path)
}

func Test__Grafana__ListResources__Incidents(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incidentPreviews": [
						{"incidentID":"incident-123","title":"API latency","status":"active","severity":"minor"}
					]
				}`)),
			},
		},
	}

	resources, err := (&Grafana{}).ListResources(resourceTypeIncident, core.ListResourcesContext{
		HTTP:        httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
	})

	require.NoError(t, err)
	require.Len(t, resources, 1)
	require.Equal(t, resourceTypeIncident, resources[0].Type)
	require.Equal(t, "incident-123", resources[0].ID)
	require.Equal(t, "API latency [active] (incident-123)", resources[0].Name)
}
