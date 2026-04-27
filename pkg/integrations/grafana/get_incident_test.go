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

func Test__GetIncident__Configuration__incidentIsIntegrationResource(t *testing.T) {
	fields := (&GetIncident{}).Configuration()

	require.NotEmpty(t, fields)
	require.Equal(t, "incident", fields[0].Name)
	require.Equal(t, configuration.FieldTypeIntegrationResource, fields[0].Type)
	require.NotNil(t, fields[0].TypeOptions)
	require.NotNil(t, fields[0].TypeOptions.Resource)
	require.Equal(t, resourceTypeIncident, fields[0].TypeOptions.Resource.Type)
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
