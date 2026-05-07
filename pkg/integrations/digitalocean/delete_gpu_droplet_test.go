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

func Test__DeleteGPUDroplet__Setup(t *testing.T) {
	component := &DeleteGPUDroplet{}

	t.Run("missing GPU droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{},
		})
		require.ErrorContains(t, err, "GPU droplet is required")
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"droplet": {"id": 123456789, "name": "gpu-node-1", "status": "active"}}`)),
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

	t.Run("expression value is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "{{ $.trigger.data.dropletId }}",
			},
			Metadata: &contexts.MetadataContext{},
		})
		require.NoError(t, err)
	})
}

func Test__DeleteGPUDroplet__Execute(t *testing.T) {
	component := &DeleteGPUDroplet{}

	t.Run("successful deletion -> emits deleted event", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNoContent,
					Body:       io.NopCloser(strings.NewReader(``)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.gpuDroplet.deleted", executionState.Type)
	})

	t.Run("droplet already deleted (404) -> emits success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"id":"not_found","message":"The resource you were accessing could not be found."}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.gpuDroplet.deleted", executionState.Type)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnauthorized,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unauthorized","message":"Unable to authenticate you"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete GPU droplet")
	})

	t.Run("invalid droplet ID format -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "not-a-number",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid droplet ID")
	})
}
