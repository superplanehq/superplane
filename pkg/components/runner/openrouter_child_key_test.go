package runner

import (
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRevokeOpenRouterChildKeyDeletesHash(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	err := revokeOpenRouterChildKey(httpContext, state, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)
	require.NoError(t, err)

	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", req.URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", req.Header.Get("Authorization"))
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestRevokeOpenRouterChildKeySkipsWhenHashIsMissing(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	err := revokeOpenRouterChildKey(httpContext, &contexts.ExecutionStateContext{KVs: map[string]string{}}, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)
	require.NoError(t, err)
	assert.Empty(t, httpContext.Requests)
}

func TestRevokeOpenRouterChildKeyReturnsGetKVError(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	err := revokeOpenRouterChildKey(httpContext, &kvErrorState{}, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, log.NewEntry(log.New()))
	require.EqualError(t, err, "kv unavailable")
	assert.Empty(t, httpContext.Requests)
}

func TestRevokeOpenRouterChildKeyRetriesDeleteThenSucceeds(t *testing.T) {
	setChildKeyDeleteRetryBackoff(t, 0)
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeleteErrorResponse(),
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	err := revokeOpenRouterChildKey(httpContext, state, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestRevokeOpenRouterChildKeyKeepsHashWhenDeleteFails(t *testing.T) {
	setChildKeyDeleteRetryBackoff(t, 0)
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeleteErrorResponse(),
		openRouterChildKeyDeleteErrorResponse(),
		openRouterChildKeyDeleteErrorResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	err := revokeOpenRouterChildKey(httpContext, state, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)
	require.Error(t, err)
	require.Len(t, httpContext.Requests, childKeyDeleteMaxAttempts)
	assert.Equal(t, "or-hash-1", state.KVs[OpenRouterChildKeyHashKV])
}

func TestPollTaskDeletesOpenRouterChildKey(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "succeeded"}}

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	err := pollTask(core.ActionHookContext{
		RunnerTasks:    tasks,
		Parameters:     map[string]any{"task_id": "task-1"},
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger:   log.NewEntry(log.New()),
		Requests: &contexts.RequestContext{},
	}, "runnerSuperPlane.finished")
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestPollTaskSchedulesRevokeRetryWhenAlreadyFinishedAndDeleteFails(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "canceled"}}
	setChildKeyDeleteRetryBackoff(t, 0)

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeleteErrorResponse(),
		openRouterChildKeyDeleteErrorResponse(),
		openRouterChildKeyDeleteErrorResponse(),
	}}
	state := &contexts.ExecutionStateContext{
		Finished: true,
		KVs: map[string]string{
			OpenRouterChildKeyHashKV: "or-hash-1",
		},
	}
	requests := &contexts.RequestContext{}

	err := pollTask(core.ActionHookContext{
		RunnerTasks: tasks,
		Parameters: map[string]any{
			"task_id":         "task-1",
			"organization_id": "org-1",
		},
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger:   log.NewEntry(log.New()),
		Requests: requests,
	}, "runnerSuperPlane.finished")
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, childKeyDeleteMaxAttempts)

	assert.Equal(t, hookActionPoll, requests.Action)
	assert.Equal(t, "task-1", requests.Params["task_id"])
	assert.Equal(t, "or-hash-1", state.KVs[OpenRouterChildKeyHashKV])
}

func TestCancelTaskDeletesOpenRouterChildKey(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "canceled"}}

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
		"task_id":                "task-1",
	}}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks:    tasks,
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger: log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)

	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestCancelTaskKeepsChildKeyWhenTaskCancelFails(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{CancelErr: errors.New("cancel failed")}

	httpContext := &contexts.HTTPContext{}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
		"task_id":                "task-1",
	}}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks:    tasks,
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger: log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.Error(t, err)
	assert.Empty(t, httpContext.Requests)

	assert.Equal(t, "or-hash-1", state.KVs[OpenRouterChildKeyHashKV])
}

func TestCancelTaskDeletesOpenRouterChildKeyWhenAlreadyFinished(t *testing.T) {
	tasks := &contexts.RunnerTaskContext{Task: &core.RunnerTask{ID: "task-1", State: "running"}}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}

	err := cancelTask(core.ExecutionContext{
		RunnerTasks: tasks,
		HTTP:        httpContext,
		ExecutionState: &contexts.ExecutionStateContext{
			Finished: true,
			KVs: map[string]string{
				OpenRouterChildKeyHashKV: "or-hash-1",
			},
		},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
}

func openRouterChildKeyDeletedResponse() *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))}
}

func openRouterChildKeyDeleteErrorResponse() *http.Response {
	return &http.Response{
		StatusCode: http.StatusInternalServerError,
		Body:       io.NopCloser(strings.NewReader(`{"error":{"message":"unavailable"}}`)),
	}
}

func setChildKeyDeleteRetryBackoff(t *testing.T, backoff time.Duration) {
	t.Helper()
	original := childKeyDeleteRetryBackoff
	childKeyDeleteRetryBackoff = backoff
	t.Cleanup(func() {
		childKeyDeleteRetryBackoff = original
	})
}

type kvErrorState struct {
	contexts.ExecutionStateContext
}

func (s *kvErrorState) GetKV(string) (string, error) {
	return "", errors.New("kv unavailable")
}
