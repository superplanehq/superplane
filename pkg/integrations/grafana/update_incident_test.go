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

func Test__UpdateIncident__Configuration__UsesExpectedFieldTypes(t *testing.T) {
	fields := (&UpdateIncident{}).Configuration()

	require.Equal(t, "incident", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncident, fields[0].TypeOptions.Resource.Type)

	require.Equal(t, "severity", fields[2].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[2].Type)
	require.NotNil(t, fields[2].TypeOptions)
	require.NotNil(t, fields[2].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncidentSeverity, fields[2].TypeOptions.Resource.Type)

	require.Equal(t, "labels", fields[3].Name)
	require.Equal(t, configuration.FieldTypeList, fields[3].Type)
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

func Test__UpdateIncident__Execute__EmitsFullIncidentWhenLabelsAlreadyExist(t *testing.T) {
	component := &UpdateIncident{}
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: http.StatusBadRequest,
				Body:       io.NopCloser(strings.NewReader(`{"error":"duplicate field option"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"error":"label already exists"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{
						"incidentID":"incident-123",
						"title":"API latency",
						"severity":"minor",
						"status":"active",
						"labels":[{"label":"prod"}],
						"isDrill":true
					}
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
	require.Len(t, httpCtx.Requests, 3)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.GetIncident", httpCtx.Requests[2].URL.Path)

	require.Len(t, execCtx.Payloads, 1)
	emitted := execCtx.Payloads[0].(map[string]any)
	out := emitted["data"].(*Incident)
	require.Equal(t, "incident-123", out.IncidentID)
	require.Equal(t, "API latency", out.Title)
	require.Equal(t, "minor", out.Severity)
	require.Equal(t, "active", out.Status)
	require.True(t, out.IsDrill)
	require.Len(t, out.Labels, 1)
	require.Equal(t, "prod", out.Labels[0].Label)
	require.Equal(t, "https://grafana.example.com/a/grafana-irm-app/incidents/incident-123", out.IncidentURL)
}

func Test__UpdateIncident__Execute__RefreshesIncidentWhenSomeLabelsDuplicate(t *testing.T) {
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
				Body:       io.NopCloser(strings.NewReader(`{"field":{"slug":"tags"}}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body:       io.NopCloser(strings.NewReader(`{"error":"label already exists"}`)),
			},
			{
				StatusCode: http.StatusOK,
				Body: io.NopCloser(strings.NewReader(`{
					"incident":{
						"incidentID":"incident-123",
						"title":"New title",
						"severity":"minor",
						"status":"active",
						"labels":[{"label":"prod"}]
					}
				}`)),
			},
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"incident": "incident-123",
			"title":    "New title",
			"labels":   []any{"prod"},
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"baseURL": "https://grafana.example.com", "apiToken": "token"}},
		ExecutionState: execCtx,
	})

	require.NoError(t, err)
	require.True(t, execCtx.Passed)
	require.Len(t, httpCtx.Requests, 4)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.UpdateTitle", httpCtx.Requests[0].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/FieldsService.AddLabelValue", httpCtx.Requests[1].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.AddLabel", httpCtx.Requests[2].URL.Path)
	require.Equal(t, "/api/plugins/grafana-irm-app/resources/api/v1/IncidentsService.GetIncident", httpCtx.Requests[3].URL.Path)

	require.Len(t, execCtx.Payloads, 1)
	emitted := execCtx.Payloads[0].(map[string]any)
	out := emitted["data"].(*Incident)
	require.Equal(t, "incident-123", out.IncidentID)
	require.Equal(t, "New title", out.Title)
	require.Len(t, out.Labels, 1)
	require.Equal(t, "prod", out.Labels[0].Label)
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
