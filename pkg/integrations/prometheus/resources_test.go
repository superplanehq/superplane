package prometheus

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

func Test__Prometheus__ListResources(t *testing.T) {
	integration := &Prometheus{}

	t.Run("unknown resource type returns empty list", func(t *testing.T) {
		resources, err := integration.ListResources("unknown", core.ListResourcesContext{})
		require.NoError(t, err)
		assert.Empty(t, resources)
	})

	t.Run("lists silence resources", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"id":"abc123","status":{"state":"active"},"comment":"Maintenance window"},
						{"id":"xyz789","status":{"state":"expired"},"comment":""},
						{"id":"   ","status":{"state":"active"},"comment":"ignored"}
					]`)),
				},
			},
		}

		resources, err := integration.ListResources(ResourceTypeSilence, core.ListResourcesContext{
			HTTP: httpCtx,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"baseURL":  "https://prometheus.example.com",
					"authType": AuthTypeNone,
				},
			},
		})
		require.NoError(t, err)
		require.Len(t, resources, 2)

		assert.Equal(t, ResourceTypeSilence, resources[0].Type)
		assert.Equal(t, "abc123", resources[0].ID)
		assert.Equal(t, "Maintenance window (active)", resources[0].Name)

		assert.Equal(t, ResourceTypeSilence, resources[1].Type)
		assert.Equal(t, "xyz789", resources[1].ID)
		assert.Equal(t, "xyz789 (expired)", resources[1].Name)
	})
}
