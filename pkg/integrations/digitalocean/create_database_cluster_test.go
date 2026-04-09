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

func Test__CreateDatabaseCluster__Setup(t *testing.T) {
	component := &CreateDatabaseCluster{}

	t.Run("missing required fields return errors", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid config returns no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":     "superplane-db",
				"engine":   "pg",
				"version":  "18",
				"region":   "nyc1",
				"size":     "db-s-1vcpu-1gb",
				"numNodes": "1",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__CreateDatabaseCluster__Execute(t *testing.T) {
	component := &CreateDatabaseCluster{}

	t.Run("successful creation stores cluster ID and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"database": {
							"id": "cluster-1",
							"name": "superplane-db",
							"engine": "pg",
							"version": "18.0",
							"region": "nyc1",
							"size": "db-s-1vcpu-1gb",
							"num_nodes": 1,
							"status": "creating"
						}
					}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "superplane-db",
				"engine":   "pg",
				"version":  "18",
				"region":   "nyc1",
				"size":     "db-s-1vcpu-1gb",
				"numNodes": "1",
			},
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "cluster-1", metadata["databaseClusterID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, databaseClusterPollInterval, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("invalid numNodes returns error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":     "superplane-db",
				"engine":   "pg",
				"version":  "18",
				"region":   "nyc1",
				"size":     "db-s-1vcpu-1gb",
				"numNodes": "one",
			},
			Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
		})

		require.ErrorContains(t, err, "invalid numNodes")
	})
}

func Test__CreateDatabaseCluster__HandleAction(t *testing.T) {
	component := &CreateDatabaseCluster{}

	t.Run("online cluster emits result", func(t *testing.T) {
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
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata:       &contexts.MetadataContext{Metadata: map[string]any{"databaseClusterID": "cluster-1"}},
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.Equal(t, "digitalocean.database.cluster.created", executionState.Type)
	})

	t.Run("creating cluster reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"database": {
							"id": "cluster-1",
							"name": "superplane-db",
							"status": "creating"
						}
					}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requestCtx := &contexts.RequestContext{}
		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata:       &contexts.MetadataContext{Metadata: map[string]any{"databaseClusterID": "cluster-1"}},
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, databaseClusterPollInterval, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("failed cluster returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"database": {
							"id": "cluster-1",
							"name": "superplane-db",
							"status": "failed"
						}
					}`)),
				},
			},
		}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata:       &contexts.MetadataContext{Metadata: map[string]any{"databaseClusterID": "cluster-1"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       &contexts.RequestContext{},
		})

		require.ErrorContains(t, err, "database cluster reached failed status")
	})
}
