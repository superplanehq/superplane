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

func Test__AssignReservedIP__Setup(t *testing.T) {
	component := &AssignReservedIP{}

	t.Run("missing reservedIP returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"action": "assign",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "reservedIP is required")
	})

	t.Run("missing action returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "action is required")
	})

	t.Run("invalid action returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "invalid",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "invalid action")
	})

	t.Run("assign without droplet returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "assign",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "droplet is required when action is assign")
	})

	t.Run("valid assign configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "assign",
				"droplet":    "98765432",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid unassign configuration -> no error (no dropletID needed)", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "unassign",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})
}

func Test__AssignReservedIP__Execute(t *testing.T) {
	component := &AssignReservedIP{}

	t.Run("successful assign -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576123,
							"status": "in-progress",
							"type": "assign",
							"started_at": "2026-03-13T10:10:00Z",
							"resource_id": 0,
							"resource_type": "reserved_ip",
							"region_slug": "nyc3"
						}
					}`)),
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
				"reservedIP": "104.131.186.241",
				"action":     "assign",
				"droplet":    "98765432",
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
		assert.Equal(t, 2048576123, metadata["actionID"])
		assert.Equal(t, "104.131.186.241", metadata["reservedIP"])
		assert.Equal(t, "assign", metadata["action"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 5*time.Second, requestCtx.Duration)
		assert.False(t, executionState.Passed)
	})

	t.Run("successful unassign -> stores metadata and schedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576124,
							"status": "in-progress",
							"type": "unassign",
							"started_at": "2026-03-13T10:15:00Z",
							"resource_id": 0,
							"resource_type": "reserved_ip",
							"region_slug": "nyc3"
						}
					}`)),
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
				"reservedIP": "104.131.186.241",
				"action":     "unassign",
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
		assert.Equal(t, 2048576124, metadata["actionID"])
		assert.Equal(t, "unassign", metadata["action"])

		assert.Equal(t, "poll", requestCtx.Action)
		assert.Equal(t, 5*time.Second, requestCtx.Duration)
	})

	t.Run("invalid droplet ID format -> returns error", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "assign",
				"droplet":    "not-a-number",
			},
			HTTP:           &contexts.HTTPContext{},
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "invalid droplet ID")
	})

	t.Run("API error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusUnprocessableEntity,
					Body:       io.NopCloser(strings.NewReader(`{"id":"unprocessable_entity","message":"Droplet already has this reserved IP assigned"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"reservedIP": "104.131.186.241",
				"action":     "assign",
				"droplet":    "98765432",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Metadata:       &contexts.MetadataContext{},
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to initiate reserved IP action")
	})
}

func Test__AssignReservedIP__HandleAction(t *testing.T) {
	component := &AssignReservedIP{}

	t.Run("poll: in-progress -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576123,
							"status": "in-progress",
							"type": "assign"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		requestCtx := &contexts.RequestContext{}
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"actionID":   2048576123,
					"reservedIP": "104.131.186.241",
					"action":     "assign",
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       requestCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requestCtx.Action)
		assert.False(t, executionState.Passed)
	})

	t.Run("poll: completed assign -> emits success", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576123,
							"status": "completed",
							"type": "assign",
							"started_at": "2026-03-13T10:10:00Z",
							"completed_at": "2026-03-13T10:10:05Z",
							"resource_id": 0,
							"resource_type": "reserved_ip",
							"region_slug": "nyc3"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"actionID":   2048576123,
					"reservedIP": "104.131.186.241",
					"action":     "assign",
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "digitalocean.reservedip.assign", executionState.Type)
		assert.Len(t, executionState.Payloads, 1)
	})

	t.Run("poll: completed unassign -> emits success with unassign type", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576124,
							"status": "completed",
							"type": "unassign",
							"started_at": "2026-03-13T10:15:00Z",
							"completed_at": "2026-03-13T10:15:03Z",
							"resource_type": "reserved_ip",
							"region_slug": "nyc3"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"actionID":   2048576124,
					"reservedIP": "104.131.186.241",
					"action":     "unassign",
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "digitalocean.reservedip.unassign", executionState.Type)
	})

	t.Run("poll: errored -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"action": {
							"id": 2048576123,
							"status": "errored",
							"type": "assign"
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{"apiToken": "test-token"},
		}

		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"actionID":   2048576123,
					"reservedIP": "104.131.186.241",
					"action":     "assign",
				},
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: executionState,
			Requests:       &contexts.RequestContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "errored")
	})

	t.Run("unknown action -> returns error", func(t *testing.T) {
		err := component.HandleAction(core.ActionContext{
			Name:           "unknown",
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       &contexts.MetadataContext{},
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "unknown action")
	})
}
