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

func Test__GetGPUDroplet__Setup(t *testing.T) {
	component := &GetGPUDroplet{}

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

func Test__GetGPUDroplet__Execute(t *testing.T) {
	component := &GetGPUDroplet{}

	gpuDropletJSON := `{
		"droplet": {
			"id": 123456789,
			"name": "my-gpu-droplet",
			"memory": 245760,
			"vcpus": 20,
			"disk": 480,
			"status": "active",
			"region": {"name": "New York 3", "slug": "nyc3"},
			"image": {"id": 12345, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64"},
			"size_slug": "gpu-h100x1-80gb",
			"networks": {"v4": [{"ip_address": "192.0.2.1", "type": "public"}]},
			"tags": ["ml", "gpu"]
		}
	}`

	t.Run("successful fetch -> emits droplet data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(gpuDropletJSON))},
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
		assert.Equal(t, "digitalocean.gpuDroplet.fetched", executionState.Type)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
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
				"gpuDroplet": "999999999",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to get GPU droplet")
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
		assert.Contains(t, err.Error(), "invalid GPU droplet ID")
	})
}
