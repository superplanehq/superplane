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

func Test__CreateDroplet__Setup(t *testing.T) {
	component := &CreateDroplet{}

	t.Run("missing name returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
			},
		})

		require.ErrorContains(t, err, "name is required")
	})

	t.Run("missing region returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":  "my-droplet",
				"size":  "s-1vcpu-1gb",
				"image": "ubuntu-24-04-x64",
			},
		})

		require.ErrorContains(t, err, "region is required")
	})

	t.Run("missing size returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-droplet",
				"region": "nyc3",
				"image":  "ubuntu-24-04-x64",
			},
		})

		require.ErrorContains(t, err, "size is required")
	})

	t.Run("missing image returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-droplet",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
			},
		})

		require.ErrorContains(t, err, "image is required")
	})

	t.Run("expression name is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "{{ $.trigger.data.hostname }}",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
			},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my-droplet",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
			},
		})

		require.NoError(t, err)
	})
}

func Test__CreateDroplet__Execute(t *testing.T) {
	component := &CreateDroplet{}

	t.Run("successful creation -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"droplet": {
							"id": 98765432,
							"name": "my-droplet",
							"memory": 1024,
							"vcpus": 1,
							"disk": 25,
							"status": "new",
							"region": {"name": "New York 3", "slug": "nyc3"},
							"image": {"id": 12345, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64"},
							"size_slug": "s-1vcpu-1gb",
							"networks": {"v4": []},
							"tags": ["web"]
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
				"name":   "my-droplet",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
				"tags":   []string{"web"},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
		})

		require.NoError(t, err)

		// Should store droplet ID in metadata
		metadata, ok := metadataCtx.Metadata.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, 98765432, metadata["dropletID"])

		// Should schedule a poll action
		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 10*time.Second, requestCtx.Duration)

		// Should NOT emit yet (waiting for active status)
		assert.False(t, executionState.Passed)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Name is already in use"}`)),
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
				"name":   "my-droplet",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to create droplet")
	})
}

func Test__CreateDroplet__HandleAction(t *testing.T) {
	component := &CreateDroplet{}

	t.Run("droplet active -> emits with IP address", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplet": {
							"id": 98765432,
							"name": "my-droplet",
							"memory": 1024,
							"vcpus": 1,
							"disk": 25,
							"status": "active",
							"region": {"name": "New York 3", "slug": "nyc3"},
							"image": {"id": 12345, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64"},
							"size_slug": "s-1vcpu-1gb",
							"networks": {"v4": [{"ip_address": "104.131.186.241", "type": "public"}]},
							"tags": ["web"]
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
			Metadata: map[string]any{"dropletID": 98765432},
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
		assert.Equal(t, "digitalocean.droplet.created", executionState.Type)
	})

	t.Run("droplet still new -> schedules another poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplet": {
							"id": 98765432,
							"name": "my-droplet",
							"status": "new",
							"region": {"name": "New York 3", "slug": "nyc3"},
							"image": {"id": 12345, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64"},
							"size_slug": "s-1vcpu-1gb",
							"networks": {"v4": []},
							"tags": []
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
			Metadata: map[string]any{"dropletID": 98765432},
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
		assert.Equal(t, 10*time.Second, requestCtx.Duration)
	})

	t.Run("droplet terminal status -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"droplet": {
							"id": 98765432,
							"name": "my-droplet",
							"status": "off",
							"region": {"name": "New York 3", "slug": "nyc3"},
							"image": {"id": 12345, "name": "Ubuntu 24.04 (LTS) x64", "slug": "ubuntu-24-04-x64"},
							"size_slug": "s-1vcpu-1gb",
							"networks": {"v4": []},
							"tags": []
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
			Metadata: map[string]any{"dropletID": 98765432},
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
		assert.Contains(t, err.Error(), "unexpected status")
		assert.False(t, executionState.Passed)
	})
}
