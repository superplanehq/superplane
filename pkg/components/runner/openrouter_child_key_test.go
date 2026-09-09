package runner

import (
	"io"
	"net/http"
	"strings"
	"testing"

	log "github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func TestRevokeOpenRouterChildKeyDeletesHash(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
	}}
	state := &contexts.ExecutionStateContext{KVs: map[string]string{
		OpenRouterChildKeyHashKV: "or-hash-1",
	}}

	revokeOpenRouterChildKey(httpContext, state, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)

	require.Len(t, httpContext.Requests, 1)
	req := httpContext.Requests[0]
	assert.Equal(t, http.MethodDelete, req.Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", req.URL.String())
	assert.Equal(t, "Bearer sk-or-mgmt", req.Header.Get("Authorization"))
}

func TestRevokeOpenRouterChildKeySkipsWhenHashIsMissing(t *testing.T) {
	httpContext := &contexts.HTTPContext{}
	revokeOpenRouterChildKey(httpContext, &contexts.ExecutionStateContext{KVs: map[string]string{}}, &contexts.HostedLLMContext{
		Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
	}, nil)
	assert.Empty(t, httpContext.Requests)
}

func TestPollBrokerTaskDeletesOpenRouterChildKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"task-1","status":"succeeded","exit_code":0}`))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
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
}

func TestHandleBrokerWebhookDeletesOpenRouterChildKey(t *testing.T) {
	t.Setenv("TASK_BROKER_BASE_URL", "https://broker.example")
	t.Setenv("TASK_BROKER_AUTH_TOKEN", "token-1")
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
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
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
	}}

	err := cancelBrokerTask(core.ExecutionContext{
		HTTP: httpContext,
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{
			OpenRouterChildKeyHashKV: "or-hash-1",
			"task_id":                "task-1",
		}},
		HostedLLM: &contexts.HostedLLMContext{
			Access: core.HostedLLMAccess{ManagementKey: "sk-or-mgmt"},
		},
		Logger: log.NewEntry(log.New()),
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 2)
	assert.Equal(t, http.MethodDelete, httpContext.Requests[0].Method)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
	assert.Equal(t, http.MethodPost, httpContext.Requests[1].Method)
	assert.Equal(t, "https://broker.example/v1/tasks/task-1/cancel", httpContext.Requests[1].URL.String())
}

func TestCancelBrokerTaskDeletesOpenRouterChildKeyWhenAlreadyFinished(t *testing.T) {
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"deleted":true}`))},
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
	})
	require.NoError(t, err)
	require.Len(t, httpContext.Requests, 1)
	assert.Equal(t, "https://openrouter.ai/api/v1/keys/or-hash-1", httpContext.Requests[0].URL.String())
}
