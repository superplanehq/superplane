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

func Test__RunEvaluation__Setup(t *testing.T) {
	component := &RunEvaluation{}

	t.Run("missing testCaseId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"agentId": "agent-123",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "testCaseId is required")
	})

	t.Run("missing agentId returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"testCaseId": "tc-123",
				"runName":    "my-run",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "agentId is required")
	})

	t.Run("missing runName returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"testCaseId": "tc-123",
				"agentId":    "agent-123",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "runName is required")
	})

	t.Run("runName over 64 chars returns error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"testCaseId": "tc-123",
				"agentId":    "agent-123",
				"runName":    strings.Repeat("a", 65),
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "runName must be 64 characters or less")
	})

	t.Run("expression values are accepted at setup time", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"testCaseId": "{{ $.trigger.data.testCaseId }}",
				"agentId":    "{{ $.trigger.data.agentId }}",
				"runName":    "{{ $.trigger.data.runName }}",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.NoError(t, err)
	})

	t.Run("valid configuration -> no error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"testCaseId": "tc-uuid-123",
				"agentId":    "agent-uuid-456",
				"runName":    "my-eval-run",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// ListEvaluationTestCases
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_test_cases": [{"test_case_uuid": "tc-uuid-123", "name": "My Test Case", "workspace_uuid": "ws-uuid-001"}]
						}`)),
					},
					{
						// GetAgent
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"agent": {"uuid": "agent-uuid-456", "name": "staging-bot", "workspace": {"uuid": "ws-uuid-001"}}
						}`)),
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

func Test__RunEvaluation__Execute(t *testing.T) {
	component := &RunEvaluation{}

	t.Run("starts evaluation run and schedules poll", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"testCaseId": "tc-uuid-123",
				"agentId":    "agent-uuid-456",
				"runName":    "my-eval-run",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// RunEvaluation — POST returns only UUIDs
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run_uuids": ["run-uuid-789"]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       metadata,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"testCaseId":    "tc-uuid-123",
					"testCaseName":  "My Test Case",
					"workspaceUUID": "ws-uuid-001",
					"agentId":       "agent-uuid-456",
					"agentName":     "staging-bot",
				},
			},
			Requests: requests,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requests.Action)

		// names from node metadata must be stored in the execution metadata
		stored, ok := metadata.Metadata.(evalRunMetadata)
		require.True(t, ok)
		assert.Equal(t, "My Test Case", stored.TestCaseName)
		assert.Equal(t, "staging-bot", stored.AgentName)
		assert.Equal(t, "ws-uuid-001", stored.WorkspaceUUID)
	})

	t.Run("expression agentId -> agentName is resolved UUID and workspace resolved from API", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"testCaseId": "tc-uuid-123",
				"agentId":    "agent-uuid-456",
				"runName":    "my-eval-run",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// RunEvaluation
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"evaluation_run_uuids": ["run-uuid-789"]}`)),
					},
					{
						// GetAgent (workspace resolution for expression)
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"agent": {"uuid": "agent-uuid-456", "name": "staging-bot", "workspace": {"uuid": "ws-uuid-001"}}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       metadata,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "{{ root().staging_agent_uuid }}",
					"agentName":    "{{ root().staging_agent_uuid }}",
				},
			},
			Requests: requests,
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(evalRunMetadata)
		require.True(t, ok)
		assert.Equal(t, "agent-uuid-456", stored.AgentName)
		assert.Equal(t, "ws-uuid-001", stored.WorkspaceUUID)
	})

	t.Run("expression testCaseId -> testCaseName falls back to resolved UUID", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"testCaseId": "tc-uuid-123",
				"agentId":    "agent-uuid-456",
				"runName":    "my-eval-run",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body:       io.NopCloser(strings.NewReader(`{"evaluation_run_uuids": ["run-uuid-789"]}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       metadata,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"testCaseId":   "{{ $.trigger.data.testCaseId }}",
					"testCaseName": "{{ $.trigger.data.testCaseId }}",
					"agentId":      "agent-uuid-456",
					"agentName":    "staging-bot",
				},
			},
			Requests: requests,
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(evalRunMetadata)
		require.True(t, ok)
		// Expression was resolved to a UUID at execution time — name must be the UUID, not the expression
		assert.Equal(t, "tc-uuid-123", stored.TestCaseName)
	})

	t.Run("expression agentId -> agentName falls back to resolved UUID", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		metadata := &contexts.MetadataContext{}
		requests := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"testCaseId": "tc-uuid-123",
				"agentId":    "agent-uuid-456",
				"runName":    "my-eval-run",
			},
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run_uuids": ["run-uuid-789"]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			ExecutionState: executionState,
			Metadata:       metadata,
			NodeMetadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "{{ root().staging_agent_uuid }}",
					"agentName":    "{{ root().staging_agent_uuid }}",
				},
			},
			Requests: requests,
		})

		require.NoError(t, err)
		stored, ok := metadata.Metadata.(evalRunMetadata)
		require.True(t, ok)
		// Expression was resolved to a UUID at execution time — name must be the UUID, not the expression
		assert.Equal(t, "agent-uuid-456", stored.AgentName)
	})
}

func Test__RunEvaluation__HandleAction(t *testing.T) {
	component := &RunEvaluation{}

	t.Run("running status -> reschedules poll", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetEvaluationRun
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_STATUS_RUNNING"
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"evalRunUUID":  "run-uuid-789",
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "agent-uuid-456",
					"agentName":    "staging-bot",
				},
			},
			ExecutionState: executionState,
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.Equal(t, "poll", requests.Action)
		assert.False(t, executionState.Passed)
	})

	t.Run("completed + passed -> emits to passed channel", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetEvaluationRun
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_SUCCESSFUL",
								"pass_status": true,
								"agent_name": "staging-bot",
								"test_case_name": "My Test Case",
								"star_metric_result": {"metric_name": "Correctness", "number_value": 4.5}
							}
						}`)),
					},
					{
						// GetEvaluationRunResults
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_SUCCESSFUL",
								"pass_status": true,
								"agent_name": "staging-bot",
								"test_case_name": "My Test Case",
								"star_metric_result": {"metric_name": "Correctness", "number_value": 4.5}
							},
							"prompts": [
								{"input": "Test prompt?", "output": "Test answer", "ground_truth": "Expected answer"}
							]
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"evalRunUUID":  "run-uuid-789",
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "agent-uuid-456",
					"agentName":    "staging-bot",
				},
			},
			ExecutionState: executionState,
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "passed", executionState.Channel)
		assert.Equal(t, "digitalocean.evaluation.passed", executionState.Type)
	})

	t.Run("completed + not passed -> emits to failed channel", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetEvaluationRun
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_STATUS_COMPLETED",
								"pass_status": false,
								"star_metric_result": {"metric_name": "Correctness", "number_value": 2.1}
							}
						}`)),
					},
					{
						// GetEvaluationRunResults
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_STATUS_COMPLETED",
								"pass_status": false,
								"star_metric_result": {"metric_name": "Correctness", "number_value": 2.1}
							},
							"prompts": []
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"evalRunUUID":  "run-uuid-789",
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "agent-uuid-456",
					"agentName":    "staging-bot",
				},
			},
			ExecutionState: executionState,
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "failed", executionState.Channel)
		assert.Equal(t, "digitalocean.evaluation.failed", executionState.Type)
	})

	t.Run("eval run failed -> emits to failed channel with error", func(t *testing.T) {
		executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
		requests := &contexts.RequestContext{}

		err := component.HandleAction(core.ActionContext{
			Name: "poll",
			HTTP: &contexts.HTTPContext{
				Responses: []*http.Response{
					{
						// GetEvaluationRun
						StatusCode: http.StatusOK,
						Body: io.NopCloser(strings.NewReader(`{
							"evaluation_run": {
								"evaluation_run_uuid": "run-uuid-789",
								"status": "EVALUATION_RUN_STATUS_FAILED",
								"pass_status": false,
								"error_description": "agent timed out"
							}
						}`)),
					},
				},
			},
			Integration: &contexts.IntegrationContext{
				Configuration: map[string]any{"apiToken": "test-token"},
			},
			Metadata: &contexts.MetadataContext{
				Metadata: map[string]any{
					"evalRunUUID":  "run-uuid-789",
					"testCaseId":   "tc-uuid-123",
					"testCaseName": "My Test Case",
					"agentId":      "agent-uuid-456",
					"agentName":    "staging-bot",
				},
			},
			ExecutionState: executionState,
			Requests:       requests,
		})

		require.NoError(t, err)
		assert.True(t, executionState.Passed)
		assert.Equal(t, "failed", executionState.Channel)
		assert.Equal(t, "digitalocean.evaluation.failed", executionState.Type)
	})
}
