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

func Test__DeclareDrill__Configuration__IncludesStatusAndStartTime(t *testing.T) {
	fields := (&DeclareDrill{}).Configuration()

	require.Equal(t, "status", fields[4].Name)
	require.Equal(t, configuration.FieldTypeSelect, fields[4].Type)
	require.Equal(t, "startTime", fields[5].Name)
	require.Equal(t, configuration.FieldTypeDateTime, fields[5].Type)
}

func Test__DeclareDrill__ExampleOutput__MarksIncidentAsDrill(t *testing.T) {
	output := (&DeclareDrill{}).ExampleOutput()
	data, ok := output["data"].(map[string]any)
	require.True(t, ok)
	require.Equal(t, true, data["isDrill"])
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
