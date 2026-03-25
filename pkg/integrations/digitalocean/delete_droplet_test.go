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

func Test__DeleteDroplet__Setup(t *testing.T) {
	component := &DeleteDroplet{}

	t.Run("missing droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
			Metadata:      &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "droplet is required")
	})

	t.Run("empty droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "droplet is required")
	})

	t.Run("expression droplet is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "{{ $.trigger.data.dropletId }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid droplet -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "98765432",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"droplet": {
								"id": 98765432,
								"name": "test-droplet"
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{
					"apiToken": "test-token",
				},
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__DeleteDroplet__Execute(t *testing.T) {
	component := &DeleteDroplet{}

	t.Run("successful deletion -> emits deletion confirmation", func(t *testing.T) {
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
				"droplet": "98765432",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.droplet.deleted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("droplet ID in scientific notation string -> succeeds", func(t *testing.T) {
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
				// This is the exact shape produced by expression resolution for numeric values:
				// `fmt.Sprintf("%v", float64(557784760))` -> "5.5778476e+08"
				"droplet": "5.5778476e+08",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.droplet.deleted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("droplet ID out of int64 range (2^63) in scientific notation string -> returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
			},
		}
		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		// 2^63, which is out of range for signed int64 and would overflow int on 64-bit platforms.
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet": "9.223372036854776e+18",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "too large")
		assert.False(t, executionState.Passed)
	})

	t.Run("invalid droplet ID format -> returns error", func(t *testing.T) {
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
				"droplet": "not-a-number",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid droplet")
		assert.False(t, executionState.Passed)
	})

	t.Run("droplet not found (404) -> emits success (idempotent)", func(t *testing.T) {
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
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.droplet.deleted", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("API error (not 404) -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"id":"server_error","message":"Internal server error"}`)),
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
				"droplet": "98765432",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete droplet")
		assert.False(t, executionState.Passed)
	})

	t.Run("unauthorized (401) -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "invalid-token",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"droplet": "98765432",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete droplet")
		assert.False(t, executionState.Passed)
	})
}
