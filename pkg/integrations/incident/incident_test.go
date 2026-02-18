package incident

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__IncidentIO__Sync(t *testing.T) {
	i := &IncidentIO{}

	t.Run("no API key -> error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": ""},
		}

		err := i.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			Integration:   integrationCtx,
		})

		require.ErrorContains(t, err, "API key is required")
	})

	t.Run("successful sync -> ready", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"severities":[{"id":"01FCNDV6P870EA6S7TK1DSYDG0","name":"Minor","description":"Low impact","rank":1}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}

		err := i.Sync(core.SyncContext{
			Configuration: integrationCtx.Configuration,
			HTTP:          httpContext,
			Integration:   integrationCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "ready", integrationCtx.State)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.incident.io/v1/severities", httpContext.Requests[0].URL.String())
	})
}

func Test__IncidentIO__ListResources(t *testing.T) {
	i := &IncidentIO{}

	t.Run("unknown resource type -> empty", func(t *testing.T) {
		resources, err := i.ListResources("unknown", core.ListResourcesContext{
			Integration: &contexts.IntegrationContext{},
		})

		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("severity -> list from API", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"severities":[{"id":"01FCNDV6P870EA6S7TK1DSYDG0","name":"Minor","description":"Low impact","rank":1}]}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiKey": "test-key"},
		}

		resources, err := i.ListResources("severity", core.ListResourcesContext{
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, resources, 1)
		assert.Equal(t, "severity", resources[0].Type)
		assert.Equal(t, "Minor", resources[0].Name)
		assert.Equal(t, "01FCNDV6P870EA6S7TK1DSYDG0", resources[0].ID)
	})
}
