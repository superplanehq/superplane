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

func Test__DeleteDatabase__Setup(t *testing.T) {
	component := &DeleteDatabase{}

	t.Run("missing database cluster returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"database": "app_db",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "databaseCluster is required")
	})

	t.Run("missing database returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "database is required")
	})

	t.Run("valid configuration resolves database metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"database":        "app_db",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [
								{"id": "cluster-1", "name": "primary-postgres", "engine": "pg"}
							]
						}`)),
					},
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"dbs": [
								{"name": "defaultdb"},
								{"name": "app_db"}
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
		assert.Equal(t, DatabaseNodeMetadata{
			DatabaseClusterID:   "cluster-1",
			DatabaseClusterName: "primary-postgres",
			DatabaseName:        "app_db",
		}, metadataCtx.Metadata)
	})
}

func Test__DeleteDatabase__Execute(t *testing.T) {
	component := &DeleteDatabase{}

	t.Run("successful deletion emits deleted payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"database":        "app_db",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: DatabaseNodeMetadata{
					DatabaseClusterID:   "cluster-1",
					DatabaseClusterName: "primary-postgres",
					DatabaseName:        "app_db",
				},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.database.deleted", executionState.Type)
		payload, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "digitalocean.database.deleted", payload["type"])
		assert.Equal(t, map[string]any{
			"name":                "app_db",
			"databaseClusterId":   "cluster-1",
			"databaseClusterName": "primary-postgres",
			"deleted":             true,
		}, payload["data"])
	})

	t.Run("database not found emits success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message":"not found"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"database":        "app_db",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			NodeMetadata: &contexts.MetadataContext{
				Metadata: DatabaseNodeMetadata{
					DatabaseClusterID:   "cluster-1",
					DatabaseClusterName: "primary-postgres",
				},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.database.deleted", executionState.Type)
	})
}
