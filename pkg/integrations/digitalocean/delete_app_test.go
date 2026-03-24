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

func Test__DeleteApp__Setup(t *testing.T) {
	component := &DeleteApp{}

	t.Run("missing app returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "app is required")
	})

	t.Run("expression app is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"app": "{{ $.trigger.data.appId }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid app -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"app": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"app": {
								"id": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48",
								"spec": {"name": "test-app"}
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteApp__Execute(t *testing.T) {
	component := &DeleteApp{}

	t.Run("successful deletion -> emits deletion confirmation", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader("")),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"app": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48"},
			HTTP:          httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.app.deleted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("app not found (404) -> emits success (idempotent)", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "not found"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"app": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48"},
			HTTP:          httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.app.deleted", executionState.Type)
	})

	t.Run("API error (not 404) -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"message": "internal error"}`)),
				},
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{"app": "b6bdf840-2854-4f87-a9f6-6a0c4dbf3a48"},
			HTTP:          httpContext,
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete app")
	})
}
