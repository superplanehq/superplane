package digitalocean

import (
	"fmt"
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

func Test__UpdateGPUDroplet__Setup(t *testing.T) {
	component := &UpdateGPUDroplet{}

	t.Run("missing GPU droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name": "new-name",
			},
		})
		require.ErrorContains(t, err, "GPU droplet is required")
	})

	t.Run("neither name nor size provided returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
			},
		})
		require.ErrorContains(t, err, "at least one of name or size must be provided")
	})

	t.Run("name only -> no error", func(t *testing.T) {
		name := "new-gpu-droplet-name"
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"name":       name,
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

	t.Run("size only -> no error", func(t *testing.T) {
		size := "gpu-h100x8-640gb"
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"gpuSize":    size,
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

	t.Run("name and size -> no error", func(t *testing.T) {
		name := "renamed-gpu"
		size := "gpu-h100x8-640gb"
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"name":       name,
				"gpuSize":    size,
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
}

func Test__UpdateGPUDroplet__Execute(t *testing.T) {
	component := &UpdateGPUDroplet{}

	renameActionJSON := `{"action": {"id": 111, "status": "in-progress", "type": "rename"}}`
	powerOffActionJSON := `{"action": {"id": 222, "status": "in-progress", "type": "power_off"}}`

	t.Run("rename only -> initiates rename and schedules poll with renaming_only state", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(renameActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		name := "new-gpu-name"
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"name":       name,
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
		assert.Equal(t, 111, metadata["actionID"])
		assert.Equal(t, 123456789, metadata["dropletID"])
		assert.Equal(t, "renaming_only", metadata["state"])
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 10*time.Second, requestCtx.Duration)
	})

	t.Run("rename + resize -> initiates rename and schedules poll with renaming state", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(renameActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		name := "new-gpu-name"
		size := "gpu-h100x8-640gb"
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"name":       name,
				"gpuSize":    size,
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
		assert.Equal(t, 111, metadata["actionID"])
		assert.Equal(t, "renaming", metadata["state"])
		assert.Equal(t, "gpu-h100x8-640gb", metadata["newSize"])
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("resize only -> powers off first and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(powerOffActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		size := "gpu-h100x8-640gb"
		metadataCtx := &contexts.MetadataContext{}
		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "123456789",
				"gpuSize":    size,
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
		assert.Equal(t, 222, metadata["actionID"])
		assert.Equal(t, "powering_off_for_resize", metadata["state"])
		assert.Equal(t, "gpu-h100x8-640gb", metadata["newSize"])
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("invalid droplet ID -> returns error", func(t *testing.T) {
		size := "gpu-h100x8-640gb"
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"gpuDroplet": "not-a-number",
				"gpuSize":    size,
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid GPU droplet ID")
	})
}

func Test__UpdateGPUDroplet__HandleHook(t *testing.T) {
	component := &UpdateGPUDroplet{}

	activeDropletJSON := `{
		"droplet": {
			"id": 123456789,
			"name": "renamed-gpu",
			"memory": 245760,
			"vcpus": 20,
			"disk": 480,
			"status": "active",
			"region": {"name": "New York 3", "slug": "nyc3"},
			"image": {"id": 12345, "name": "Ubuntu 22.04 (LTS) x64", "slug": "ubuntu-22-04-x64"},
			"size_slug": "gpu-h100x8-640gb",
			"networks": {"v4": [{"ip_address": "192.0.2.1", "type": "public"}]},
			"tags": []
		}
	}`

	powerOffActionJSON := `{"action": {"id": 222, "status": "in-progress", "type": "power_off"}}`
	resizeActionJSON := `{"action": {"id": 333, "status": "in-progress", "type": "resize"}}`
	powerOnActionJSON := `{"action": {"id": 444, "status": "in-progress", "type": "power_on"}}`

	completedActionJSON := func(id int) string {
		return `{"action": {"id": ` + fmt.Sprintf("%d", id) + `, "status": "completed"}}`
	}

	t.Run("action in-progress -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"action": {"id": 111, "status": "in-progress"}}`))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  111,
				"dropletID": 123456789,
				"state":     "renaming_only",
				"newSize":   "",
			},
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

	t.Run("renaming_only completed -> emits updated droplet", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(completedActionJSON(111)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(activeDropletJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  111,
				"dropletID": 123456789,
				"state":     "renaming_only",
				"newSize":   "",
			},
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
		assert.Equal(t, "digitalocean.gpuDroplet.updated", executionState.Type)
	})

	t.Run("renaming completed (with resize) -> powers off droplet", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(completedActionJSON(111)))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(powerOffActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  111,
				"dropletID": 123456789,
				"state":     "renaming",
				"newSize":   "gpu-h100x8-640gb",
			},
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

		updatedMetadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "powering_off_for_resize", updatedMetadata["state"])
		assert.Equal(t, 222, updatedMetadata["actionID"])
	})

	t.Run("powering_off_for_resize completed -> initiates resize", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(completedActionJSON(222)))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(resizeActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  222,
				"dropletID": 123456789,
				"state":     "powering_off_for_resize",
				"newSize":   "gpu-h100x8-640gb",
			},
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

		updatedMetadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "resizing", updatedMetadata["state"])
		assert.Equal(t, 333, updatedMetadata["actionID"])
	})

	t.Run("resizing completed -> powers droplet back on", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(completedActionJSON(333)))},
				{StatusCode: http.StatusCreated, Body: io.NopCloser(strings.NewReader(powerOnActionJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  333,
				"dropletID": 123456789,
				"state":     "resizing",
				"newSize":   "gpu-h100x8-640gb",
			},
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

		updatedMetadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "powering_on_after_resize", updatedMetadata["state"])
		assert.Equal(t, 444, updatedMetadata["actionID"])
	})

	t.Run("powering_on_after_resize completed -> emits updated droplet", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(completedActionJSON(444)))},
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(activeDropletJSON))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  444,
				"dropletID": 123456789,
				"state":     "powering_on_after_resize",
				"newSize":   "gpu-h100x8-640gb",
			},
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
		assert.Equal(t, "digitalocean.gpuDroplet.updated", executionState.Type)
	})

	t.Run("action errored -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"action": {"id": 111, "status": "errored"}}`))},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: map[string]any{
				"actionID":  111,
				"dropletID": 123456789,
				"state":     "renaming_only",
				"newSize":   "",
			},
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
		assert.Contains(t, err.Error(), "errored")
	})

	t.Run("unknown hook name -> returns error", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           "unknown",
			HTTP:           &contexts.HTTPContext{},
			Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiToken": "test-token"}},
			Metadata:       &contexts.MetadataContext{},
			ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown hook")
	})
}
