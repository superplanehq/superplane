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

func Test__DeleteSnapshot__Setup(t *testing.T) {
	component := &DeleteSnapshot{}

	t.Run("missing snapshot returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})

		require.ErrorContains(t, err, "snapshot is required")
	})

	t.Run("valid configuration -> resolves metadata", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"snapshot": {
							"id": "12345678",
							"name": "my-snapshot",
							"created_at": "2024-06-15T10:30:00Z",
							"resource_id": "98765432",
							"resource_type": "droplet",
							"regions": ["nyc3"],
							"min_disk_size": 25,
							"size_gigabytes": 2.36
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

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"snapshot": "12345678",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)

		metadata := metadataCtx.Metadata.(SnapshotNodeMetadata)
		assert.Equal(t, "12345678", metadata.SnapshotID)
		assert.Equal(t, "my-snapshot", metadata.SnapshotName)
	})

	t.Run("expression placeholder -> stores raw expression in metadata", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"snapshot": "{{ steps.prev.snapshotId }}",
			},
			Metadata: metadataCtx,
		})

		require.NoError(t, err)

		metadata := metadataCtx.Metadata.(SnapshotNodeMetadata)
		assert.Equal(t, "{{ steps.prev.snapshotId }}", metadata.SnapshotName)
	})
}

func Test__DeleteSnapshot__Execute(t *testing.T) {
	component := &DeleteSnapshot{}

	t.Run("successful deletion -> emits output", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
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
				"snapshot": "12345678",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.snapshot.deleted", executionState.Type)

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		assert.Equal(t, "digitalocean.snapshot.deleted", payload["type"])
		data := payload["data"].(map[string]any)
		assert.Equal(t, "12345678", data["snapshotId"])
		assert.Equal(t, true, data["deleted"])
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
				"snapshot": "99999999",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete snapshot")
		assert.False(t, executionState.Passed)
	})
}
