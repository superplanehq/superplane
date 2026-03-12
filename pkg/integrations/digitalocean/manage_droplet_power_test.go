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

func Test__ManageDropletPower__Setup(t *testing.T) {
	component := &ManageDropletPower{}

	t.Run("missing droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"operation": "reboot",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "droplet is required")
	})

	t.Run("missing operation returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet": "98765432",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "operation is required")
	})

	t.Run("invalid operation returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":   "98765432",
				"operation": "invalid_op",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "invalid operation")
	})

	t.Run("valid power_on operation -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":   "98765432",
				"operation": "power_on",
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

	t.Run("valid shutdown operation -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":   "98765432",
				"operation": "shutdown",
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

	t.Run("expression droplet is accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"droplet":   "{{ $.trigger.data.dropletId }}",
				"operation": "reboot",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__ManageDropletPower__Execute(t *testing.T) {
	component := &ManageDropletPower{}

	t.Run("successful action initiation -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 987654321,
							"status": "in-progress",
							"type": "reboot",
							"started_at": "2024-03-12T10:15:00Z",
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
				"droplet":   "98765432",
				"operation": "reboot",
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
		assert.Equal(t, 987654321, metadata["actionID"])
		assert.Equal(t, 98765432, metadata["dropletID"])
		assert.Equal(t, "reboot", metadata["operation"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 5*time.Second, requestCtx.Duration)
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
				"droplet":   "not-a-number",
				"operation": "reboot",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid droplet")
		assert.False(t, executionState.Passed)
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Droplet is already powered on"}`)),
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
				"droplet":   "98765432",
				"operation": "power_on",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initiate power action")
		assert.False(t, executionState.Passed)
	})
}

func Test__ManageDropletPower__HandleAction(t *testing.T) {
	component := &ManageDropletPower{}

	t.Run("action completed -> emits with action data", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 987654321,
							"status": "completed",
							"type": "reboot",
							"started_at": "2024-03-12T10:15:00Z",
							"completed_at": "2024-03-12T10:17:30Z",
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
				"actionID":  987654321,
				"dropletID": 98765432,
				"operation": "reboot",
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
		assert.Equal(t, "digitalocean.droplet.power.reboot", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("action still in-progress -> schedules another poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 987654321,
							"status": "in-progress",
							"type": "reboot",
							"started_at": "2024-03-12T10:15:00Z",
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
				"actionID":  987654321,
				"dropletID": 98765432,
				"operation": "reboot",
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
							"id": 987654321,
							"status": "errored",
							"type": "reboot",
							"started_at": "2024-03-12T10:15:00Z",
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
				"actionID":  987654321,
				"dropletID": 98765432,
				"operation": "reboot",
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
		assert.Contains(t, err.Error(), "power action failed")
		assert.False(t, executionState.Passed)
	})
}
