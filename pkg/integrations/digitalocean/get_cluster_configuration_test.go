package digitalocean

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

func Test__GetClusterConfiguration__Setup(t *testing.T) {
	component := &GetClusterConfiguration{}

	t.Run("missing cluster returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "databaseCluster is required")
	})

	t.Run("valid config resolves metadata", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [{"id":"cluster-1","name":"superplane-db"}]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Metadata:    &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__GetClusterConfiguration__Execute(t *testing.T) {
	component := &GetClusterConfiguration{}

	t.Run("successful fetch emits configuration", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"config": {
							"autovacuum_naptime": 60,
							"jit": true
						}
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
			},
			HTTP:        httpContext,
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			NodeMetadata: &contexts.MetadataContext{Metadata: map[string]any{
				"databaseClusterName": "superplane-db",
			}},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, "digitalocean.database.cluster.config.fetched", executionState.Type)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "cluster-1", payload["databaseClusterId"])
		assert.Equal(t, "superplane-db", payload["databaseClusterName"])
	})
}
