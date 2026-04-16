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

func Test__GetDatabaseCluster__Setup(t *testing.T) {
	component := &GetDatabaseCluster{}

	t.Run("missing cluster returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "databaseCluster is required")
	})

	t.Run("valid cluster resolves metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [
								{"id": "cluster-1", "name": "superplane-db"}
							]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, DatabaseClusterNodeMetadata{
			DatabaseClusterID:   "cluster-1",
			DatabaseClusterName: "superplane-db",
		}, metadataCtx.Metadata)
	})
}

func Test__GetDatabaseCluster__Execute(t *testing.T) {
	component := &GetDatabaseCluster{}

	t.Run("successful retrieval emits cluster", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"database": {
							"id": "cluster-1",
							"name": "superplane-db",
							"engine": "pg",
							"version": "18.0",
							"region": "nyc1",
							"size": "db-s-1vcpu-1gb",
							"num_nodes": 1,
							"status": "online"
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
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, "digitalocean.database.cluster.fetched", executionState.Type)
	})
}
