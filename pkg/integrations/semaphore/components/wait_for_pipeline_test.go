package components

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__WaitForPipeline__Setup(t *testing.T) {
	component := &WaitForPipeline{}

	t.Run("missing project -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"pipelineId": "ppl-1",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "project is required")
	})

	t.Run("both pipelineId and branch -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
				"branch":     "main",
				"commitSha":  "abc123",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "not both")
	})

	t.Run("neither pipelineId nor branch+commitSha -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "test-project",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "either pipelineId or both branch and commitSha must be provided")
	})

	t.Run("branch without commitSha -> error", func(t *testing.T) {
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project": "test-project",
				"branch":  "main",
			},
			Metadata: &contexts.MetadataContext{},
		})

		require.ErrorContains(t, err, "either pipelineId or both branch and commitSha must be provided")
	})

	t.Run("valid configuration with pipelineId -> resolves project and requests webhook", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"metadata": {"id": "proj-123", "name": "test-project"},
						"spec": {}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)

		metadata := metadataCtx.Get().(WaitForPipelineNodeMetadata)
		require.NotNil(t, metadata.Project)
		assert.Equal(t, "proj-123", metadata.Project.ID)
		assert.Equal(t, "test-project", metadata.Project.Name)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})

	t.Run("valid configuration with branch and commitSha -> resolves project and requests webhook", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"metadata": {"id": "proj-123", "name": "test-project"},
						"spec": {}
					}`)),
				},
			},
		}

		integrationCtx := &contexts.IntegrationContext{
			Configuration: map[string]any{
				"organizationUrl": "https://example.semaphoreci.com",
				"apiToken":        "token-123",
			},
		}

		metadataCtx := &contexts.MetadataContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":   "test-project",
				"branch":    "main",
				"commitSha": "abc123",
			},
			HTTP:        httpContext,
			Integration: integrationCtx,
			Metadata:    metadataCtx,
		})

		require.NoError(t, err)

		metadata := metadataCtx.Get().(WaitForPipelineNodeMetadata)
		require.NotNil(t, metadata.Project)
		assert.Equal(t, "proj-123", metadata.Project.ID)
		require.Len(t, integrationCtx.WebhookRequests, 1)
	})

	t.Run("same project already cached -> no HTTP calls", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineNodeMetadata{
				Project: &Project{ID: "proj-123", Name: "test-project"},
			},
		}

		httpContext := &contexts.HTTPContext{}
		err := component.Setup(core.SetupContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:     httpContext,
			Metadata: metadataCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, httpContext.Requests)
	})
}

func Test__WaitForPipeline__Execute_PipelineID(t *testing.T) {
	component := &WaitForPipeline{}

	nodeMetadata := &contexts.MetadataContext{
		Metadata: WaitForPipelineNodeMetadata{
			Project: &Project{ID: "proj-123", Name: "test-project"},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
			"apiToken":        "token-123",
		},
	}

	t.Run("pipeline already done and passed -> emits to passed immediately", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {
							"ppl_id": "ppl-1",
							"wf_id": "wf-1",
							"state": "done",
							"result": "passed"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.True(t, execState.Finished)
		assert.Empty(t, requestsCtx.Calls)
		assert.Equal(t, "ppl-1", execState.KVs["pipeline"])
	})

	t.Run("pipeline already done and failed -> emits to failed", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {
							"ppl_id": "ppl-1",
							"state": "done",
							"result": "failed"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, FailedOutputChannel, execState.Channel)
	})

	t.Run("pipeline not found -> schedules lookup and lookupTimeout", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":              "test-project",
				"pipelineId":           "ppl-1",
				"lookupTimeoutSeconds": 30,
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		require.Len(t, requestsCtx.Calls, 2)
		assert.Equal(t, waitForPipelineHookLookup, requestsCtx.Calls[0].Action)
		assert.Equal(t, waitForPipelineHookLookupTimeout, requestsCtx.Calls[1].Action)
	})

	t.Run("pipeline not done yet -> sets KV and schedules poll and waitTimeout", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {
							"ppl_id": "ppl-1",
							"state": "running"
						}
					}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Equal(t, "ppl-1", execState.KVs["pipeline"])
		require.Len(t, requestsCtx.Calls, 2)
		assert.Equal(t, waitForPipelineHookPoll, requestsCtx.Calls[0].Action)
		assert.Equal(t, waitForPipelineHookWaitTimeout, requestsCtx.Calls[1].Action)
	})

	t.Run("hard HTTP error -> returns error", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusInternalServerError,
					Body:       io.NopCloser(strings.NewReader(`{"error": "boom"}`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":    "test-project",
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.Error(t, err)
	})

	t.Run("invalid configuration -> error", func(t *testing.T) {
		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project": "test-project",
			},
			NodeMetadata: nodeMetadata,
		})

		require.ErrorContains(t, err, "either pipelineId or both branch and commitSha must be provided")
	})
}

func Test__WaitForPipeline__Execute_BranchAndCommit(t *testing.T) {
	component := &WaitForPipeline{}

	nodeMetadata := &contexts.MetadataContext{
		Metadata: WaitForPipelineNodeMetadata{
			Project: &Project{ID: "proj-123", Name: "test-project"},
		},
	}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
			"apiToken":        "token-123",
		},
	}

	t.Run("matching pipeline found and done -> emits", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-1", "branch_name": "main", "commit_sha": "abc123", "state": "done", "result": "passed"},
						{"ppl_id": "ppl-2", "branch_name": "main", "commit_sha": "other", "state": "done", "result": "passed"}
					]`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "test-project",
				"branch":    "main",
				"commitSha": "abc123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
		assert.Equal(t, "ppl-1", execState.KVs["pipeline"])
	})

	t.Run("no matching commit found -> schedules lookup and lookupTimeout", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-2", "branch_name": "main", "commit_sha": "other", "state": "done", "result": "passed"}
					]`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "test-project",
				"branch":    "main",
				"commitSha": "abc123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		require.Len(t, requestsCtx.Calls, 2)
		assert.Equal(t, waitForPipelineHookLookup, requestsCtx.Calls[0].Action)
		assert.Equal(t, waitForPipelineHookLookupTimeout, requestsCtx.Calls[1].Action)

		metadata := metadataCtx.Get().(WaitForPipelineExecutionMetadata)
		assert.Equal(t, "proj-123", metadata.ProjectID)
	})

	t.Run("matching pipeline found but not done -> schedules poll and waitTimeout", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`[
						{"ppl_id": "ppl-1", "wf_id": "wf-1", "branch_name": "main", "commit_sha": "abc123", "state": "running"}
					]`)),
				},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{}
		requestsCtx := &contexts.RequestContext{}

		err := component.Execute(core.ExecutionContext{
			Configuration: map[string]any{
				"project":   "test-project",
				"branch":    "main",
				"commitSha": "abc123",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			NodeMetadata:   nodeMetadata,
			Metadata:       metadataCtx,
			ExecutionState: execState,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
		assert.Equal(t, "ppl-1", execState.KVs["pipeline"])
		require.Len(t, requestsCtx.Calls, 2)
		assert.Equal(t, waitForPipelineHookPoll, requestsCtx.Calls[0].Action)
		assert.Equal(t, waitForPipelineHookWaitTimeout, requestsCtx.Calls[1].Action)

		metadata := metadataCtx.Get().(WaitForPipelineExecutionMetadata)
		require.NotNil(t, metadata.Workflow)
		assert.Equal(t, "https://example.semaphoreci.com/workflows/wf-1", metadata.Workflow.URL)
	})
}

func Test__WaitForPipeline__HandleHook_Lookup(t *testing.T) {
	component := &WaitForPipeline{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
			"apiToken":        "token-123",
		},
	}

	t.Run("already finished -> no-op", func(t *testing.T) {
		requestsCtx := &contexts.RequestContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookLookup,
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, requestsCtx.Calls)
	})

	t.Run("already resolved -> no-op", func(t *testing.T) {
		requestsCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: "running"},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookLookup,
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       metadataCtx,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		assert.Empty(t, requestsCtx.Calls)
	})

	t.Run("pipeline still not found -> reschedules lookup", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusNotFound,
					Body:       io.NopCloser(strings.NewReader(`{"error": "not found"}`)),
				},
			},
		}

		requestsCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{ProjectID: "proj-123"},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name: waitForPipelineHookLookup,
			Configuration: map[string]any{
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       metadataCtx,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		require.Len(t, requestsCtx.Calls, 1)
		assert.Equal(t, waitForPipelineHookLookup, requestsCtx.Calls[0].Action)
	})

	t.Run("pipeline found -> resolves and schedules poll/waitTimeout", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {"ppl_id": "ppl-1", "state": "running"}
					}`)),
				},
			},
		}

		requestsCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{ProjectID: "proj-123"},
		}
		execState := &contexts.ExecutionStateContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name: waitForPipelineHookLookup,
			Configuration: map[string]any{
				"pipelineId": "ppl-1",
			},
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
			Metadata:       metadataCtx,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		require.Len(t, requestsCtx.Calls, 2)
		assert.Equal(t, waitForPipelineHookPoll, requestsCtx.Calls[0].Action)
		assert.Equal(t, waitForPipelineHookWaitTimeout, requestsCtx.Calls[1].Action)
		assert.Equal(t, "ppl-1", execState.KVs["pipeline"])
	})
}

func Test__WaitForPipeline__HandleHook_LookupTimeout(t *testing.T) {
	component := &WaitForPipeline{}

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookLookupTimeout,
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		})

		require.NoError(t, err)
	})

	t.Run("already resolved -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1"},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookLookupTimeout,
			ExecutionState: execState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
	})

	t.Run("still unresolved -> fails execution", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{}
		execState := &contexts.ExecutionStateContext{}

		err := component.HandleHook(core.ActionHookContext{
			Name: waitForPipelineHookLookupTimeout,
			Configuration: map[string]any{
				"lookupTimeoutSeconds": 30,
			},
			ExecutionState: execState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Equal(t, "lookupTimeout", execState.FailureReason)
	})
}

func Test__WaitForPipeline__HandleHook_Poll(t *testing.T) {
	component := &WaitForPipeline{}

	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{
			"organizationUrl": "https://example.semaphoreci.com",
			"apiToken":        "token-123",
		},
	}

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookPoll,
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		})

		require.NoError(t, err)
	})

	t.Run("still not done -> reschedules poll", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {"ppl_id": "ppl-1", "state": "running"}
					}`)),
				},
			},
		}

		requestsCtx := &contexts.RequestContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: "running"},
			},
		}

		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookPoll,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: &contexts.ExecutionStateContext{},
			Metadata:       metadataCtx,
			Requests:       requestsCtx,
		})

		require.NoError(t, err)
		require.Len(t, requestsCtx.Calls, 1)
		assert.Equal(t, waitForPipelineHookPoll, requestsCtx.Calls[0].Action)
	})

	t.Run("done -> emits to the right channel", func(t *testing.T) {
		httpContext := &contexts.HTTPContext{
			Responses: []*http.Response{
				{
					StatusCode: http.StatusOK,
					Body: io.NopCloser(strings.NewReader(`{
						"pipeline": {"ppl_id": "ppl-1", "state": "done", "result": "passed"}
					}`)),
				},
			},
		}

		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: "running"},
				Workflow: &WorkflowMetadata{ID: "wf-1", URL: "https://example.semaphoreci.com/workflows/wf-1"},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookPoll,
			HTTP:           httpContext,
			Integration:    integrationCtx,
			ExecutionState: execState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.Equal(t, PassedOutputChannel, execState.Channel)
	})
}

func Test__WaitForPipeline__HandleHook_WaitTimeout(t *testing.T) {
	component := &WaitForPipeline{}

	t.Run("already finished -> no-op", func(t *testing.T) {
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookWaitTimeout,
			ExecutionState: &contexts.ExecutionStateContext{Finished: true},
		})

		require.NoError(t, err)
	})

	t.Run("pipeline already done -> no-op", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: PipelineStateDone},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name:           waitForPipelineHookWaitTimeout,
			ExecutionState: execState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.False(t, execState.Finished)
	})

	t.Run("still running -> fails execution", func(t *testing.T) {
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: "running"},
			},
		}

		execState := &contexts.ExecutionStateContext{}
		err := component.HandleHook(core.ActionHookContext{
			Name: waitForPipelineHookWaitTimeout,
			Configuration: map[string]any{
				"timeoutSeconds": 120,
			},
			ExecutionState: execState,
			Metadata:       metadataCtx,
		})

		require.NoError(t, err)
		assert.True(t, execState.Finished)
		assert.False(t, execState.Passed)
		assert.Equal(t, "timeout", execState.FailureReason)
	})
}

func Test__WaitForPipeline__HandleWebhook(t *testing.T) {
	component := &WaitForPipeline{}
	logger := logrus.NewEntry(logrus.New())

	t.Run("no signature -> 403", func(t *testing.T) {
		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Headers: http.Header{},
			Logger:  logger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("invalid signature -> 403", func(t *testing.T) {
		secret := "test-secret"
		headers := http.Header{}
		headers.Set("X-Semaphore-Signature-256", "sha256=invalid")

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    []byte(`{"pipeline":{"id":"ppl-1","state":"done"}}`),
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Logger:  logger,
		})

		assert.Equal(t, http.StatusForbidden, code)
		assert.ErrorContains(t, err, "invalid signature")
	})

	t.Run("valid signature but no matching execution -> 200 no-op", func(t *testing.T) {
		secret := "test-secret"
		body := []byte(`{"pipeline":{"id":"ppl-1","state":"done","result":"passed"}}`)
		headers := buildSemaphoreHeaders(secret, body)

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Logger:  logger,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return nil, assert.AnError
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
	})

	t.Run("valid signature and matching execution -> emits to passed and injects workflow url", func(t *testing.T) {
		secret := "test-secret"
		body := []byte(`{"pipeline":{"id":"ppl-1","state":"done","result":"passed"}}`)
		headers := buildSemaphoreHeaders(secret, body)

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: "running"},
				Workflow: &WorkflowMetadata{ID: "wf-1", URL: "https://example.semaphoreci.com/workflows/wf-1"},
			},
		}

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Logger:  logger,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				assert.Equal(t, "pipeline", key)
				assert.Equal(t, "ppl-1", value)
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Equal(t, PassedOutputChannel, execState.Channel)

		payload := execState.Payloads[0].(map[string]any)["data"].(map[string]any)
		workflowData := payload["workflow"].(map[string]any)
		assert.Equal(t, "https://example.semaphoreci.com/workflows/wf-1", workflowData["url"])
	})

	t.Run("already done execution -> idempotent no-op", func(t *testing.T) {
		secret := "test-secret"
		body := []byte(`{"pipeline":{"id":"ppl-1","state":"done","result":"passed"}}`)
		headers := buildSemaphoreHeaders(secret, body)

		execState := &contexts.ExecutionStateContext{}
		metadataCtx := &contexts.MetadataContext{
			Metadata: WaitForPipelineExecutionMetadata{
				Pipeline: &PipelineMetadata{ID: "ppl-1", State: PipelineStateDone, Result: PipelineResultPassed},
			},
		}

		code, _, err := component.HandleWebhook(core.WebhookRequestContext{
			Body:    body,
			Headers: headers,
			Webhook: &contexts.NodeWebhookContext{Secret: secret},
			Logger:  logger,
			FindExecutionByKV: func(key, value string) (*core.ExecutionContext, error) {
				return &core.ExecutionContext{
					Metadata:       metadataCtx,
					ExecutionState: execState,
				}, nil
			},
		})

		require.NoError(t, err)
		assert.Equal(t, http.StatusOK, code)
		assert.Empty(t, execState.Channel)
	})
}

func Test__WaitForPipeline__ExampleOutput(t *testing.T) {
	component := &WaitForPipeline{}
	output := component.ExampleOutput()

	require.NotNil(t, output)
	assert.Contains(t, output, "data")
}
