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

func Test__GetDatabase__Setup(t *testing.T) {
	component := &GetDatabase{}

	t.Run("missing required fields return error", func(t *testing.T) {
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
				"database":        "app_db",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"databases": [{"id":"cluster-1","name":"superplane-db"}]
						}`)),
					},
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"dbs": [{"name":"app_db"}]
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

func Test__GetDatabase__Execute(t *testing.T) {
	component := &GetDatabase{}

	t.Run("successful fetch emits enriched database", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"db": {"name":"app_db"}
					}`)),
				},
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"database": {
							"id": "cluster-1",
							"name": "superplane-db",
							"engine": "pg",
							"version": "17",
							"region": "nyc1",
							"status": "online",
							"connection": {
								"host": "db.example.com",
								"port": 25060
							}
						}
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"databaseCluster": "cluster-1",
				"database":        "app_db",
			},
			HTTP:           httpContext,
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.Equal(t, "digitalocean.database.fetched", executionState.Type)
		payload := executionState.Payloads[0].(map[string]any)["data"].(map[string]any)
		assert.Equal(t, "app_db", payload["name"])
		assert.Equal(t, "cluster-1", payload["databaseClusterId"])
		assert.Equal(t, "superplane-db", payload["databaseClusterName"])
		assert.Equal(t, "pg", payload["engine"])
	})
}
