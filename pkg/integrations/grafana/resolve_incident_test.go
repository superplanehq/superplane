package grafana

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__ResolveIncident__Configuration__incidentIsIntegrationResource(t *testing.T) {
	fields := (&ResolveIncident{}).Configuration()

	require.NotEmpty(t, fields)
	require.Equal(t, "incident", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncident, fields[0].TypeOptions.Resource.Type)
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
