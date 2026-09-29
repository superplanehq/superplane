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

func TestPollBrokerTaskDeletesOpenRouterChildKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"task-1","status":"succeeded","exit_code":0}`))},
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	err := pollBrokerTask(core.ActionHookContext{
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
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, "https://broker.example/v1/tasks/task-1", httpContext.Requests[0].URL.String())
	assert.Equal(t, http.MethodDelete, httpContext.Requests[1].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[1].URL.String())
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestPollBrokerTaskSchedulesRevokeRetryWhenAlreadyFinishedAndDeleteFails(t *testing.T) {
	setChildKeyDeleteRetryBackoff(t, 0)
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"task-1","status":"canceled"}`))},
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

	err := pollBrokerTask(core.ActionHookContext{
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
	require.Len(t, httpContext.Requests, 1+childKeyDeleteMaxAttempts)
	assert.Equal(t, http.MethodGet, httpContext.Requests[0].Method)
	assert.Equal(t, hookActionPoll, requests.Action)
	assert.Equal(t, "task-1", requests.Params["task_id"])
	assert.Equal(t, "or-hash-1", state.KVs[OpenRouterChildKeyHashKV])
}

func TestHandleBrokerWebhookDeletesOpenRouterChildKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	status, _, err := handleBrokerWebhook(core.WebhookRequestContext{
		Body:   []byte(`{"task_id":"task-1","status":"succeeded","exit_code":0}`),
		HTTP:   httpContext,
		Logger: log.NewEntry(log.New()),
		FindExecutionByKV: func(key string, value string) (*core.ExecutionContext, error) {
			assert.Equal(t, "task_id", key)
			assert.Equal(t, "task-1", value)
			return &core.ExecutionContext{
				HTTP:           httpContext,
				ExecutionState: state,
				HostedLLM: &contexts.HostedLLMContext{
					Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
				},
			}, nil
		},
	}, "runnerSuperPlane.finished")
	require.NoError(t, err)
	assert.Equal(t, http.StatusOK, status)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
}

func TestCancelBrokerTaskDeletesOpenRouterChildKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"task-1","status":"canceled"}`))},
		openRouterChildKeyDeletedResponse(),
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
		"task_id":                "task-1",
	}}

	err := cancelBrokerTask(core.ExecutionContext{
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger: log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 3)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://broker.example/v1/tasks/task-1/cancel", httpContext.Requests[0].URL.String())
	assert.Equal(t, http.MethodGet, httpContext.Requests[1].Method)
	assert.Equal(t, "https://broker.example/v1/tasks/task-1", httpContext.Requests[1].URL.String())
	assert.Equal(t, http.MethodDelete, httpContext.Requests[2].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[2].URL.String())
	assert.Empty(t, state.KVs[OpenRouterChildKeyHashKV])
}

func TestCancelBrokerTaskKeepsChildKeyWhenBrokerCancelFails(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`error`))},
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
		"task_id":                "task-1",
	}}

	err := cancelBrokerTask(core.ExecutionContext{
		HTTP:           httpContext,
		ExecutionState: state,
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger: log.NewEntry(log.New()),
	}, "runnerClaudeCode.finished")
	require.Error(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, http.MethodPost, httpContext.Requests[0].Method)
	assert.Equal(t, "https://broker.example/v1/tasks/task-1/cancel", httpContext.Requests[0].URL.String())
	assert.Equal(t, "or-hash-1", state.KVs[OpenRouterChildKeyHashKV])
}

func TestCancelBrokerTaskDeletesOpenRouterChildKeyWhenAlreadyFinished(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		openRouterChildKeyDeletedResponse(),
	}}

	err := cancelBrokerTask(core.ExecutionContext{
		HTTP: httpContext,
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
