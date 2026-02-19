package terraformcloud

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

func Test__TriggerRun__Setup(t *testing.T) {
	t.Run("missing organization -> error", func(t *testing.T) {
		component := &TriggerRun{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"workspaceId": "ws-123",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "organization is required")
	})

	t.Run("missing workspace -> error", func(t *testing.T) {
		component := &TriggerRun{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"organization": "my-org",
			},
			Metadata: &contexts.MetadataContext{Metadata: map[string]any{}},
		})

		require.Error(t, err)
		assert.ErrorContains(t, err, "workspace is required")
	})

	t.Run("valid configuration -> success", func(t *testing.T) {
		component := &TriggerRun{}
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(`{"data":{"id":"ws-abc123","attributes":{"name":"my-workspace"}}}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: map[string]any{},
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"organization": "my-org",
				"workspaceId":  "ws-abc123",
				"message":      "Test run",
			},
			Metadata:    &contexts.MetadataContext{Metadata: map[string]any{}},
			HTTP:        httpContext,
			Integration: integrationCtx,
		})

		require.NoError(t, err)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})
}

func Test__TriggerRun__Execute(t *testing.T) {
	t.Run("creates a run and schedules poll", func(t *testing.T) {
		component := &TriggerRun{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusCreated,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "run-abc123",
							"attributes": {
								"status": "pending",
								"message": "Test run",
								"created-at": "2026-01-28T10:30:00.000Z",
								"source": "tfe-api"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Metadata: map[string]any{},
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		metadataCtx := &contexts.MetadataContext{Metadata: map[string]any{}}
		requestCtx := &contexts.RequestContext{}
		executionStateCtx := &contexts.ExecutionStateContext{
			KVs: map[string]string{},
		}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"organization": "my-org",
				"workspaceId":  "ws-abc123",
				"message":      "Test run",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionStateCtx,
		})

		require.NoError(t, err)
		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://app.terraform.io/api/v2/runs", httpContext.Requests[0].URL.String())
		assert.Equal(t, "Bearer test-token", httpContext.Requests[0].Header.Get("Authorization"))
		assert.Equal(t, "run-abc123", executionStateCtx.KVs["run"])
		assert.Equal(t, "poll", requestCtx.Action)
	})
}

func Test__TriggerRun__Poll(t *testing.T) {
	t.Run("run still running -> schedule another poll", func(t *testing.T) {
		component := &TriggerRun{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "run-abc123",
							"attributes": {
								"status": "planning",
								"message": "Test run",
								"created-at": "2026-01-28T10:30:00.000Z"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerRunExecutionMetadata{
				Run: RunInfo{
					ID:     "run-abc123",
					Status: "pending",
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionStateCtx,
		})

		require.NoError(t, err)
		assert.False(t, executionStateCtx.Finished)
		assert.Equal(t, "poll", requestCtx.Action)
	})

	t.Run("run applied -> emit success", func(t *testing.T) {
		component := &TriggerRun{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "run-abc123",
							"attributes": {
								"status": "applied",
								"message": "Test run",
								"created-at": "2026-01-28T10:30:00.000Z"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerRunExecutionMetadata{
				Run: RunInfo{
					ID:     "run-abc123",
					Status: "pending",
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionStateCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.Equal(t, TriggerRunSuccessChannel, executionStateCtx.Channel)
	})

	t.Run("run errored -> emit failed", func(t *testing.T) {
		component := &TriggerRun{}

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"data": {
							"id": "run-abc123",
							"attributes": {
								"status": "errored",
								"message": "Test run",
								"created-at": "2026-01-28T10:30:00.000Z"
							}
						}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"apiToken": "test-token",
				"hostname": "app.terraform.io",
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: TriggerRunExecutionMetadata{
				Run: RunInfo{
					ID:     "run-abc123",
					Status: "pending",
				},
			},
		}

		requestCtx := &contexts.RequestContext{}
		executionStateCtx := &contexts.ExecutionStateContext{KVs: map[string]string{}}

		err := component.HandleAction(core.ActionContext{
			Name:           "poll",
			HTTP:           httpContext,
			Integration:    integrationCtx,
			Metadata:       metadataCtx,
			Requests:       requestCtx,
			ExecutionState: executionStateCtx,
		})

		require.NoError(t, err)
		assert.True(t, executionStateCtx.Finished)
		assert.Equal(t, TriggerRunFailedChannel, executionStateCtx.Channel)
	})
}

func Test__TriggerRun__IsTerminalRunStatus(t *testing.T) {
	assert.True(t, IsTerminalRunStatus("applied"))
	assert.True(t, IsTerminalRunStatus("planned_and_finished"))
	assert.True(t, IsTerminalRunStatus("errored"))
	assert.True(t, IsTerminalRunStatus("discarded"))
	assert.True(t, IsTerminalRunStatus("canceled"))
	assert.True(t, IsTerminalRunStatus("force_canceled"))

	assert.False(t, IsTerminalRunStatus("pending"))
	assert.False(t, IsTerminalRunStatus("planning"))
	assert.False(t, IsTerminalRunStatus("applying"))
	assert.False(t, IsTerminalRunStatus("plan_queued"))
}

func Test__TriggerRun__IsSuccessRunStatus(t *testing.T) {
	assert.True(t, IsSuccessRunStatus("applied"))
	assert.True(t, IsSuccessRunStatus("planned_and_finished"))

	assert.False(t, IsSuccessRunStatus("errored"))
	assert.False(t, IsSuccessRunStatus("discarded"))
	assert.False(t, IsSuccessRunStatus("canceled"))
}
