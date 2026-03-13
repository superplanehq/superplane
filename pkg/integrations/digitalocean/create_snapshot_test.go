package digitalocean

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__CreateSnapshot__Setup(t *testing.T) {
	component := &CreateSnapshot{}

	t.Run("missing droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "my-snapshot",
			},
		})

		require.ErrorContains(t, err, "droplet is required")
	})

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "98765432",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplet": {
							"id": 98765432,
							"name": "my-droplet",
							"status": "active"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "98765432",
				"name":    "my-snapshot",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__CreateSnapshot__Execute(t *testing.T) {
	component := &CreateSnapshot{}

	t.Run("successful snapshot request -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 11223344,
							"status": "in-progress",
							"type": "snapshot",
							"started_at": "2024-06-15T10:30:00Z",
							"completed_at": "",
							"resource_id": 98765432,
							"resource_type": "droplet",
							"region_slug": "nyc3"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet": "98765432",
				"name":    "my-snapshot",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store action ID and droplet ID in metadata
		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 11223344, metadata["actionID"])
		assert.Equal(t, 98765432, metadata["droplet"])

		// Should schedule a poll action
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 5*time.Second, requestCtx.Duration)

		// Should NOT emit yet (waiting for action to complete)
		assert.False(t, executionState.Passed)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"The resource you requested could not be found."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet": "99999999",
				"name":    "my-snapshot",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create snapshot")
	})
}

func Test__CreateSnapshot__HandleAction(t *testing.T) {
	component := &CreateSnapshot{}

	t.Run("action completed -> fetches snapshot and emits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				// First call: GetAction
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 11223344,
							"status": "completed",
							"type": "snapshot",
							"started_at": "2024-06-15T10:30:00Z",
							"completed_at": "2024-06-15T10:32:00Z",
							"resource_id": 98765432,
							"resource_type": "droplet",
							"region_slug": "nyc3"
						}
					}`)),
				},
				// Second call: GetDropletSnapshots
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"snapshots": [
							{
								"id": "12345678",
								"name": "my-snapshot",
								"created_at": "2024-06-15T10:32:00Z",
								"resource_id": "98765432",
								"resource_type": "droplet",
								"regions": ["nyc3"],
								"min_disk_size": 25,
								"size_gigabytes": 2.36
							}
						]
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID": 11223344,
				"droplet":  98765432,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.snapshot.created", executionState.Type)
	})

	t.Run("action still in-progress -> schedules another poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 11223344,
							"status": "in-progress",
							"type": "snapshot",
							"started_at": "2024-06-15T10:30:00Z",
							"completed_at": "",
							"resource_id": 98765432,
							"resource_type": "droplet",
							"region_slug": "nyc3"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID": 11223344,
				"droplet":  98765432,
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.False(t, executionState.Passed)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 5*time.Second, requestCtx.Duration)
	})

	t.Run("action errored -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 11223344,
							"status": "errored",
							"type": "snapshot",
							"started_at": "2024-06-15T10:30:00Z",
							"completed_at": "",
							"resource_id": 98765432,
							"resource_type": "droplet",
							"region_slug": "nyc3"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID": 11223344,
				"droplet":  98765432,
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "snapshot action failed")
		assert.False(t, executionState.Passed)
	})

	t.Run("unknown action name -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.HandleAction(core.ActionContext{
			Name:           "unknown",
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}
