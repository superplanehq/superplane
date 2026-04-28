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

func Test__DeclareIncident__Configuration__severityIsIntegrationResource(t *testing.T) {
	fields := (&DeclareIncident{}).Configuration()

	require.Equal(t, "severity", fields[1].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[1].Type)
	require.NotNil(t, fields[1].TypeOptions)
	require.NotNil(t, fields[1].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncidentSeverity, fields[1].TypeOptions.Resource.Type)
}

func Test__DeclareIncident__ExampleOutput__MarksIncidentAsNotDrill(t *testing.T) {
	output := (&DeclareIncident{}).ExampleOutput()
	data, ok := output["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, false, data["isDrill"])
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
		},
	}
	execCtx := &contexts.ExecutionStateContext{}

	err := component.Execute(core.ExecutionContext{
		Configuration: map[string]any{
			"title":       "API latency",
			"severity":    "minor",
			"description": "Initial diagnosis",
			"labels":      []any{"api", "prod"},
			"status":      "resolved",
			"startTime":   "2026-04-20T10:00:00Z",
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
}
