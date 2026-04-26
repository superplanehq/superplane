package claude

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunAgent__Setup(t *testing.T) {
	a := &RunAgent{}
	integrationCtx := &contexts.IntegrationContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":         "agent_01",
			"environmentId": "env_01",
			"prompt":        "Do the thing",
		},
		Integration: integrationCtx,
	}
	require.NoError(t, a.Setup(ctx))
}

func Test__RunAgent__Setup__validation(t *testing.T) {
	a := &RunAgent{}
	integrationCtx := &contexts.IntegrationContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"environmentId": "env_01",
			"prompt":        "x",
		},
		Integration: integrationCtx,
	}
	err := a.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent")
}

func Test__RunAgent__Execute__syncIdle(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{
		Configuration: map[string]any{"apiKey": "sk-test"},
	}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  map[string]any{"agent": "ag_1", "environmentId": "ev_1", "prompt": "Hello"},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := a.Execute(execCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, runAgentPayloadType, executionState.Type)
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(RunAgentOutputPayload).Status)
	assert.Equal(t, "", requestsCtx.Action)

	require.Len(t, httpContext.Requests, 3)
	assert.Equal(t, "POST", httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/sessions")
	assert.Equal(t, anthropicBetaManagedAgents, httpContext.Requests[0].Header.Get("anthropic-beta"))
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/events")
	assert.Equal(t, "GET", httpContext.Requests[2].Method)
}

func Test__RunAgent__Execute__schedulesPoll(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	metadataCtx := &contexts.MetadataContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}

	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  map[string]any{"agent": "a", "environmentId": "e", "prompt": "p"},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Requests:       requestsCtx,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := a.Execute(execCtx)
	require.NoError(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "poll", requestsCtx.Action)
	assert.Equal(t, runAgentInitialPoll, requestsCtx.Duration)
}

func Test__RunAgent__poll__terminal(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunAgentExecutionMetadata{
			Session: &RunAgentSessionMetadata{ID: "sess_1", Status: "running"},
		},
	}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    integrationCtx,
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}

	err := a.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(RunAgentOutputPayload).Status)
}

func Test__RunAgent__poll__timeout(t *testing.T) {
	a := &RunAgent{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: RunAgentExecutionMetadata{
			Session: &RunAgentSessionMetadata{ID: "sess_1", Status: "running"},
		},
	}
	hookCtx := core.ActionHookContext{
		Name: "poll",
		Parameters: map[string]any{
			"attempt": float64(runAgentMaxPollAttempts + 1),
			"errors":  float64(0),
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := a.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, "timeout", executionState.Payloads[0].(map[string]any)["data"].(RunAgentOutputPayload).Status)
}

func Test__RunAgent__scheduleNextPoll(t *testing.T) {
	a := &RunAgent{}
	rc := &contexts.RequestContext{}
	hookCtx := core.ActionHookContext{
		Requests:       rc,
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Metadata: &contexts.MetadataContext{
			Metadata: RunAgentExecutionMetadata{Session: &RunAgentSessionMetadata{ID: "s", Status: "running"}},
		},
	}
	err := a.scheduleNextPoll(hookCtx, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, 4*runAgentInitialPoll, rc.Duration)
	assert.LessOrEqual(t, rc.Duration, runAgentMaxPollInterval)
}

func Test__buildCreateSessionBody__latest(t *testing.T) {
	b, err := buildCreateSessionBody(CreateManagedSessionRequest{
		Agent:         "  ag  ",
		EnvironmentID: "env",
	})
	require.NoError(t, err)
	s, ok := b.Agent.(string)
	require.True(t, ok)
	assert.Equal(t, "ag", s)
}

func Test__buildCreateSessionBody__pinned(t *testing.T) {
	v := 2
	b, err := buildCreateSessionBody(CreateManagedSessionRequest{
		Agent:         "ag",
		AgentVersion:  &v,
		EnvironmentID: "env",
	})
	require.NoError(t, err)
	m, ok := b.Agent.(map[string]any)
	require.True(t, ok)
	assert.Equal(t, "ag", m["id"])
	assert.Equal(t, 2, m["version"])
}
