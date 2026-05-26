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

func Test__CreateGPUDroplet__Setup(t *testing.T) {
	component := &CreateGPUDroplet{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
		})
		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing region returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"regions": [{"name": "New York 3", "slug": "nyc3", "available": true, "sizes": ["gpu-h100x1-80gb"]}], "links": {}, "meta": {"total": 1}}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
		})
		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing GPU size returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuRegion":    "nyc3",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"sizes": [{"slug": "gpu-h100x1-80gb", "memory": 245760, "vcpus": 20, "disk": 480, "transfer": 10.0, "price_monthly": 4896.00, "available": true}], "links": {}, "meta": {"total": 1}}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
		})
		require.ErrorContains(t, err, "GPU size is required")
	})

	t.Run("missing image type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
		})
		require.ErrorContains(t, err, "image type is required")
	})

	t.Run("invalid image type returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":      "my-gpu-droplet",
				"gpuRegion": "nyc3",
				"gpuSize":   "gpu-h100x1-80gb",
				"imageType": "invalid-type",
			},
		})
		require.ErrorContains(t, err, "invalid image type")
	})

	t.Run("one-click type with missing image returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":      "my-gpu-droplet",
				"gpuRegion": "nyc3",
				"gpuSize":   "gpu-h100x1-80gb",
				"imageType": "one-click",
			},
		})
		require.ErrorContains(t, err, "one-click application image is required")
	})

	t.Run("base-os type with missing image returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":      "my-gpu-droplet",
				"gpuRegion": "nyc3",
				"gpuSize":   "gpu-h100x1-80gb",
				"imageType": "base-os",
			},
		})
		require.ErrorContains(t, err, "base OS image is required")
	})

	t.Run("valid base-os configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
		})
		require.NoError(t, err)
	})

	t.Run("valid one-click configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":             "my-gpu-droplet",
				"gpuRegion":        "nyc3",
				"gpuSize":          "gpu-h100x1-80gb",
				"imageType":        "one-click",
				"oneClickGPUImage": "ml-in-a-box",
			},
		})
		require.NoError(t, err)
	})

	t.Run("expression name is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":         "{{ $.trigger.data.hostname }}",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
		})
		require.NoError(t, err)
	})
}

func Test__CreateGPUDroplet__Execute(t *testing.T) {
	component := &CreateGPUDroplet{}

	gpuDropletJSON := `{
		"droplet": {
			"id": 123456789,
			"name": "my-gpu-droplet",
			"memory": 245760,
			"vcpus": 20,
			"disk": 480,
			"status": "new",
			"region": {"name": "New York 3", "slug": "nyc3"},
			"image": {"id": 12345, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64"},
			"size_slug": "gpu-h100x1-80gb",
			"networks": {"v4": []},
			"tags": []
		}
	}`

	t.Run("successful creation with base-os image -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(gpuDropletJSON)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 123456789, metadata["dropletID"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 10*time.Second, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("successful creation with one-click image -> stores metadata and schedules poll", func(t *testing.T) {
		oneClickJSON := `{
			"droplet": {
				"id": 987654321,
				"name": "my-gpu-droplet",
				"status": "new",
				"region": {"name": "New York 3", "slug": "nyc3"},
				"image": {"id": 99999, "name": "ML-in-a-Box", "slug": "ml-in-a-box"},
				"size_slug": "gpu-h100x1-80gb",
				"networks": {"v4": []}
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body:       io.NopCloser(strings.NewReader(oneClickJSON)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":             "my-gpu-droplet",
				"gpuRegion":        "nyc3",
				"gpuSize":          "gpu-h100x1-80gb",
				"imageType":        "one-click",
				"oneClickGPUImage": "ml-in-a-box",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 987654321, metadata["dropletID"])
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"GPU droplets are not available in this region"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":         "my-gpu-droplet",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create GPU droplet")
	})

	t.Run("invalid droplet name -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"name":         "invalid name with spaces",
				"gpuRegion":    "nyc3",
				"gpuSize":      "gpu-h100x1-80gb",
				"imageType":    "base-os",
				"baseGPUImage": "ubuntu-22-04-x64",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: executionState,
		})

		require.Error(t, err)
	})
}

func Test__CreateGPUDroplet__HandleHook(t *testing.T) {
	component := &CreateGPUDroplet{}

	activeDropletJSON := `{
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
			"tags": []
		}
	}`

	t.Run("droplet active -> emits with droplet data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(activeDropletJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{"dropletID": 123456789},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleHook(core.ActionHookContext{
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
		assert.Equal(t, "digitalocean.gpuDroplet.created", executionState.Type)
	})

	t.Run("droplet still new -> schedules another poll", func(t *testing.T) {
		newDropletJSON := `{
			"droplet": {
				"id": 123456789,
				"name": "my-gpu-droplet",
				"status": "new",
				"region": {"name": "New York 3", "slug": "nyc3"},
				"image": {"id": 12345, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64"},
				"size_slug": "gpu-h100x1-80gb",
				"networks": {"v4": []}
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(newDropletJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{"dropletID": 123456789},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleHook(core.ActionHookContext{
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
		assert.Equal(t, 10*time.Second, requestCtx.Duration)
	})

	t.Run("droplet in unexpected status -> returns error", func(t *testing.T) {
		offDropletJSON := `{
			"droplet": {
				"id": 123456789,
				"name": "my-gpu-droplet",
				"status": "off",
				"region": {"name": "New York 3", "slug": "nyc3"},
				"image": {"id": 12345, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64"},
				"size_slug": "gpu-h100x1-80gb",
				"networks": {"v4": []}
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(offDropletJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{"dropletID": 123456789},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unexpected status")
		assert.False(t, executionState.Passed)
	})

	t.Run("unknown hook name -> returns error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleHook(core.ActionHookContext{
			Name:           "unknown",
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown hook")
	})
}
