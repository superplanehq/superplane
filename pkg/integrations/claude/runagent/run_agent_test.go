package runagent

import (
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Done"}]},{"type":"user.message","content":[{"type":"text","text":"Hello"}]}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
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
	assert.Equal(t, payloadType, executionState.Type)
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
	assert.Equal(t, "Done", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).LastMessage)
	assert.Equal(t, "", requestsCtx.Action)

	require.Len(t, httpContext.Requests, 5) // create, send, get status, get events, delete
	assert.Equal(t, "POST", httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/sessions")
	assert.Equal(t, anthropicBetaManagedAgents, httpContext.Requests[0].Header.Get("anthropic-beta"))
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/events")
	assert.Equal(t, "GET", httpContext.Requests[2].Method)
	assert.Equal(t, "GET", httpContext.Requests[3].Method)
	assert.Contains(t, httpContext.Requests[3].URL.Path, "/events")
	assert.Equal(t, "desc", httpContext.Requests[3].URL.Query().Get("order"))
	assert.Equal(t, sessionEventsPageLimit, httpContext.Requests[3].URL.Query().Get("limit"))
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
	assert.Equal(t, initialPoll, requestsCtx.Duration)
}

func Test__RunAgent__poll__terminal(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Final"}]},{"type":"agent.message","content":[{"type":"text","text":"Earlier"}]}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
		},
	}
	integrationCtx := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{
			Session: &SessionMetadata{ID: "sess_1", Status: "running"},
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
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
	assert.Equal(t, "Final", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).LastMessage)
	assert.Equal(t, []string{"Earlier", "Final"}, executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Messages)
}

func Test__RunAgent__poll__persistSessionKeepsSession(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Final"}]}]}`))},
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{
			Session: &SessionMetadata{ID: "sess_1", Status: "running"},
		},
	}
	hookCtx := core.ActionHookContext{
		Name:       "poll",
		Parameters: map[string]any{"attempt": float64(1), "errors": float64(0)},
		Configuration: map[string]any{
			"agent":          "agent_1",
			"environmentId":  "env_1",
			"prompt":         "do it",
			"persistSession": true,
		},
		HTTP:           httpContext,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, executionState.Finished)
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	for _, r := range httpContext.Requests {
		assert.False(t, r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1"),
			"session must be kept when persistSession is enabled")
	}
}

// A client that never builds must terminate the run rather than poll forever:
// the timeout check sits below the status read and would never be reached.
func Test__RunAgent__poll__clientErrorReportsError(t *testing.T) {
	a := &RunAgent{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name: "poll",
		Parameters: map[string]any{
			"attempt": float64(2),
			"errors":  float64(maxPollErrors - 1),
		},
		Integration:    &contexts.IntegrationContext{}, // no apiKey -> client creation fails
		Metadata:       &contexts.MetadataContext{Metadata: ExecutionMetadata{Session: &SessionMetadata{ID: "sess_1", Status: "running"}}},
		ExecutionState: executionState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, executionState.Finished)
	assert.Equal(t, "error", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
}

// sessionCalls reports whether the session was interrupted and whether it was
// deleted, so the reclaim paths can be asserted precisely.
func sessionCalls(httpContext *contexts.HTTPContext) (interrupted, deleted bool) {
	for _, r := range httpContext.Requests {
		switch {
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/sessions/sess_1/events"):
			interrupted = true
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1"):
			deleted = true
		}
	}
	return interrupted, deleted
}

func timeoutHookCtx(httpContext *contexts.HTTPContext, execState *contexts.ExecutionStateContext, config map[string]any) core.ActionHookContext {
	return core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(maxPollAttempts + 1), "errors": float64(0)},
		Configuration:  config,
		HTTP:           httpContext,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       &contexts.MetadataContext{Metadata: ExecutionMetadata{Session: &SessionMetadata{ID: "sess_1", Status: "running"}}},
		ExecutionState: execState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       &contexts.RequestContext{},
	}
}

// A timeout means we stopped watching, not that the agent stopped: the session
// must be interrupted and reclaimed rather than left running in Anthropic.
func Test__RunAgent__poll__timeoutReclaimsSession(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))}, // get session
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                 // interrupt
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                 // delete session
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	require.NoError(t, a.HandleHook(timeoutHookCtx(httpContext, execState, map[string]any{
		"agent": "agent_1", "environmentId": "env_1", "prompt": "do it",
	})))
	require.True(t, execState.Finished)
	assert.Equal(t, "timeout", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	interrupted, deleted := sessionCalls(httpContext)
	assert.True(t, interrupted, "a timed-out session must be interrupted, not left running")
	assert.True(t, deleted, "a timed-out session must be deleted by default")
}

// Keeping the transcript must still stop the agent.
func Test__RunAgent__poll__timeoutPersistSessionInterruptsButKeeps(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))}, // get session
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                 // interrupt
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}

	require.NoError(t, a.HandleHook(timeoutHookCtx(httpContext, execState, map[string]any{
		"agent": "agent_1", "environmentId": "env_1", "prompt": "do it", "persistSession": true,
	})))
	require.True(t, execState.Finished)

	interrupted, deleted := sessionCalls(httpContext)
	assert.True(t, interrupted, "a kept session must still be interrupted so the agent stops")
	assert.False(t, deleted, "session must be kept when persistSession is enabled")
}

// The event stream can lag the status endpoint: the session reports idle while
// the terminal event has not been written, so sm.Complete stays false. Once the
// retry budget is spent we must still report the real outcome — never "timeout",
// and never destroy the transcript of a run that finished.
func Test__RunAgent__poll__terminalWithIncompleteEventsReportsRealStatus(t *testing.T) {
	a := &RunAgent{}
	restore := finalMessageDelay
	finalMessageDelay = time.Millisecond
	t.Cleanup(func() { finalMessageDelay = restore })

	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))}, // get session: finished
	}}
	// Events never include session.status_idle, so Complete never becomes true.
	for range finalMessageReads {
		httpContext.Responses = append(httpContext.Responses,
			&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(
				`{"data":[{"type":"agent.message","content":[{"type":"text","text":"Partial work"}]}]}`))})
	}
	httpContext.Responses = append(httpContext.Responses,
		&http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}) // delete session

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	require.NoError(t, a.HandleHook(timeoutHookCtx(httpContext, execState, map[string]any{
		"agent": "agent_1", "environmentId": "env_1", "prompt": "do it",
	})))

	require.True(t, execState.Finished)
	out := execState.Payloads[0].(map[string]any)["data"].(OutputPayload)
	assert.Equal(t, "idle", out.Status, "a finished session must report its real status, not timeout")
	assert.Equal(t, "Partial work", out.LastMessage, "the messages we did collect must still be emitted")

	interrupted, deleted := sessionCalls(httpContext)
	assert.False(t, interrupted, "a session that already finished must not be interrupted")
	assert.True(t, deleted, "a finished session is still reclaimed by default")
}

// Repeated poll failures mean we lost sight of the session, not that it ended.
func Test__RunAgent__poll__repeatedErrorsReclaimSession(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`))}, // get session
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                            // interrupt
		{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                            // delete session
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := timeoutHookCtx(httpContext, execState, map[string]any{
		"agent": "agent_1", "environmentId": "env_1", "prompt": "do it",
	})
	hookCtx.Parameters = map[string]any{"attempt": float64(2), "errors": float64(maxPollErrors - 1)}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "error", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	interrupted, deleted := sessionCalls(httpContext)
	assert.True(t, interrupted, "an unreachable session must be interrupted")
	assert.True(t, deleted, "an unreachable session must be reclaimed")
}

func Test__RunAgent__scheduleNextPoll(t *testing.T) {
	a := &RunAgent{}
	rc := &contexts.RequestContext{}
	hookCtx := core.ActionHookContext{
		Requests:       rc,
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Metadata: &contexts.MetadataContext{
			Metadata: ExecutionMetadata{Session: &SessionMetadata{ID: "s", Status: "running"}},
		},
	}
	err := a.scheduleNextPoll(hookCtx, 3, 0)
	require.NoError(t, err)
	assert.Equal(t, 4*initialPoll, rc.Duration)
	assert.LessOrEqual(t, rc.Duration, maxPollInterval)
}

func TestClient_ManagedSessions(t *testing.T) {
	t.Run("CreateManagedSession", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`)),
			}},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		s, err := client.CreateManagedSession(CreateManagedSessionRequest{
			Agent:         "ag_1",
			EnvironmentID: "env_1",
		})
		require.NoError(t, err)
		assert.Equal(t, "sess_1", s.ID)
		require.Len(t, httpCtx.Requests, 1)

		req := httpCtx.Requests[0]
		assert.Equal(t, anthropicBetaManagedAgents, req.Header.Get("anthropic-beta"))
		assert.Equal(t, http.MethodPost, req.Method)
		assert.True(t, strings.HasSuffix(req.URL.Path, "/sessions"))
	})

	t.Run("GetManagedSession", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"id":"s","status":"idle"}`)),
			}},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		s, err := client.GetManagedSession("s")
		require.NoError(t, err)
		assert.Equal(t, "idle", s.Status)
	})
}

func TestClient_GetLastManagedSessionAgentMessage(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"type":"user.message","content":[{"type":"text","text":"Hello"}]}],"next_page":"page_2"}`)),
			},
			{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"type":"agent.message","content":[{"type":"text","text":"Done"}]}],"next_page":"page_3"}`)),
			},
		},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}

	message, events, err := client.GetLastManagedSessionAgentMessage("sess_1")
	require.NoError(t, err)
	assert.Equal(t, "Done", message)
	require.Len(t, events, 2)
	require.Len(t, httpCtx.Requests, 2)

	assert.Equal(t, "desc", httpCtx.Requests[0].URL.Query().Get("order"))
	assert.Equal(t, sessionEventsPageLimit, httpCtx.Requests[0].URL.Query().Get("limit"))
	assert.Empty(t, httpCtx.Requests[0].URL.Query().Get("page"))
	assert.Equal(t, "page_2", httpCtx.Requests[1].URL.Query().Get("page"))
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
