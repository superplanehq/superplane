package semaphore

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__GetPipeline__Name(t *testing.T) {
	g := &GetPipeline{}
	assert.Equal(t, "semaphore.getPipeline", g.Name())
}

func Test__GetPipeline__Label(t *testing.T) {
	g := &GetPipeline{}
	assert.Equal(t, "Get Pipeline", g.Label())
}

func Test__GetPipeline__Description(t *testing.T) {
	g := &GetPipeline{}
	assert.NotEmpty(t, g.Description())
}

func Test__GetPipeline__Documentation(t *testing.T) {
	g := &GetPipeline{}
	assert.NotEmpty(t, g.Documentation())
	assert.Contains(t, g.Documentation(), "Pipeline ID")
}

func Test__GetPipeline__Configuration(t *testing.T) {
	g := &GetPipeline{}
	config := g.Configuration()

	require.Len(t, config, 1)
	assert.Equal(t, "pipelineId", config[0].Name)
	assert.True(t, config[0].Required)
}

func Test__GetPipeline__OutputChannels(t *testing.T) {
	g := &GetPipeline{}
	channels := g.OutputChannels(nil)

	require.Len(t, channels, 1)
	assert.Equal(t, "default", channels[0].Name)
}

func Test__GetPipeline__ExampleOutput(t *testing.T) {
	g := &GetPipeline{}
	example := g.ExampleOutput()

	assert.NotNil(t, example)
	assert.Contains(t, example, "name")
	assert.Contains(t, example, "ppl_id")
	assert.Contains(t, example, "wf_id")
	assert.Contains(t, example, "state")
	assert.Contains(t, example, "result")
}

func Test__GetPipeline__Setup(t *testing.T) {
	g := &GetPipeline{}

	t.Run("valid configuration", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "abc-123",
			},
		}

		err := g.Setup(ctx)
		require.NoError(t, err)
	})

	t.Run("empty configuration is allowed (expression might be used)", func(t *testing.T) {
		ctx := core.SetupContext{
			Configuration: map[string]any{},
		}

		err := g.Setup(ctx)
		require.NoError(t, err)
	})
}

func Test__GetPipeline__Execute(t *testing.T) {
	g := &GetPipeline{}

	t.Run("success - fetches pipeline and emits data", func(t *testing.T) {
		pipelineResponse := `{
			"pipeline": {
				"name": "Build and Test",
				"ppl_id": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
				"wf_id": "f0e1d2c3-b4a5-6789-0fed-cba987654321",
				"state": "done",
				"result": "passed"
			}
		}`

		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body:       io.NopCloser(strings.NewReader(pipelineResponse)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			ID:          uuid.New(),
			WorkflowID:  "workflow-123",
			Logger:      log.NewEntry(log.New()),
			HTTP:        httpContext,
			Integration: integrationCtx,
			Configuration: map[string]any{
				"pipelineId": "a1b2c3d4-e5f6-7890-abcd-ef1234567890",
			},
			ExecutionState: executionState,
		}

		err := g.Execute(ctx)

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "default", executionState.Channel)
		assert.Equal(t, "semaphore.pipeline", executionState.Type)

		require.Len(t, httpContext.Requests, 1)
		assert.Equal(t, "https://example.semaphoreci.com/api/v1alpha/pipelines/a1b2c3d4-e5f6-7890-abcd-ef1234567890", httpContext.Requests[0].URL.String())

		require.Len(t, executionState.Payloads, 1)
		payload := executionState.Payloads[0].(map[string]any)
		data := payload["data"].(map[string]any)
		assert.Equal(t, "Build and Test", data["name"])
		assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", data["ppl_id"])
		assert.Equal(t, "f0e1d2c3-b4a5-6789-0fed-cba987654321", data["wf_id"])
		assert.Equal(t, "done", data["state"])
		assert.Equal(t, "passed", data["result"])
	})

	t.Run("fails when pipeline ID is empty", func(t *testing.T) {
		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			ID:             uuid.New(),
			WorkflowID:     "workflow-123",
			Logger:         log.NewEntry(log.New()),
			Integration:    integrationCtx,
			Configuration:  map[string]any{},
			ExecutionState: executionState,
		}

		err := g.Execute(ctx)

		require.NoError(t, err) // Fail() returns nil
		assert.False(t, executionState.Passed)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "validation_error", executionState.FailureReason)
		assert.Contains(t, executionState.FailureMessage, "pipeline ID is required")
	})

	t.Run("fails when API returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"message": "pipeline not found"}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		executionState := &contexts.ExecutionStateContext{
			KVs: make(map[string]string),
		}

		ctx := core.ExecutionContext{
			ID:          uuid.New(),
			WorkflowID:  "workflow-123",
			Logger:      log.NewEntry(log.New()),
			HTTP:        httpContext,
			Integration: integrationCtx,
			Configuration: map[string]any{
				"pipelineId": "nonexistent-id",
			},
			ExecutionState: executionState,
		}

		err := g.Execute(ctx)

		require.NoError(t, err) // Fail() returns nil
		assert.False(t, executionState.Passed)
		assert.True(t, executionState.Finished)
		assert.Equal(t, "api_error", executionState.FailureReason)
	})
}

func Test__GetPipeline__Cancel(t *testing.T) {
	g := &GetPipeline{}
	err := g.Cancel(core.ExecutionContext{})
	require.NoError(t, err)
}

func Test__GetPipeline__Cleanup(t *testing.T) {
	g := &GetPipeline{}
	err := g.Cleanup(core.SetupContext{})
	require.NoError(t, err)
}

func Test__GetPipeline__Actions(t *testing.T) {
	g := &GetPipeline{}
	actions := g.Actions()
	assert.Empty(t, actions)
}
