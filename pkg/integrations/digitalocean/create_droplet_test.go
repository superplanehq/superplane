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

	t.Run("invalid hostname characters -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"name":   "my_droplet",
				"region": "nyc3",
				"size":   "s-1vcpu-1gb",
				"image":  "ubuntu-24-04-x64",
			},
		})

		require.ErrorContains(t, err, "invalid droplet name")
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

	t.Run("successful creation -> emits droplet data", func(t *testing.T) {
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
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.droplet.created", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://api.digitalocean.com/v2/droplets", httpContext.Requests[0].URL.String())
		assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
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
