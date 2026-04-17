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

func Test__CreateDatabase__Setup(t *testing.T) {
	component := &CreateDatabase{}

	t.Run("missing database cluster returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "app_db",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "databaseCluster is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration resolves cluster metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"name":            "app_db",
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

func Test__CreateDatabase__Execute(t *testing.T) {
	component := &CreateDatabase{}

	t.Run("successful creation emits created database payload", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"db": {
							"name": "app_db"
						}
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"name":            "app_db",
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
		assert.Equal(t, "digitalocean.database.created", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
		payload, ok := executionState.Payloads[0].(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "digitalocean.database.created", payload["type"])
		assert.Equal(t, map[string]any{
			"name":                "app_db",
			"databaseClusterId":   "cluster-1",
			"databaseClusterName": "primary-postgres",
		}, payload["data"])
	})

	t.Run("API error returns failure", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Database management is not supported for Caching clusters."}`)),
				},
			},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"name":            "app_db",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			NodeMetadata: &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{
				KVs: map[string]string{},
			},
		})

		require.ErrorContains(t, err, "failed to create database")
	})
}
