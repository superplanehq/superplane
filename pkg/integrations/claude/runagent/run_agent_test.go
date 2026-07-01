package runagent

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

func Test__RunAgent__poll__timeout(t *testing.T) {
	a := &RunAgent{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{
			Session: &SessionMetadata{ID: "sess_1", Status: "running"},
		},
	}
	hookCtx := core.ActionHookContext{
		Name: "poll",
		Parameters: map[string]any{
			"attempt": float64(maxPollAttempts + 1),
			"errors":  float64(0),
		},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	err := a.HandleHook(hookCtx)
	require.NoError(t, err)
	require.True(t, executionState.Finished)
	assert.Equal(t, "timeout", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
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

func TestClient_ListAgents(t *testing.T) {
	t.Run("single page", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{{
				StatusCode: 200,
				Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"agent_1","name":"A"},{"id":"agent_2","name":"B"}]}`)),
			}},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		agents, err := client.ListAgents()
		require.NoError(t, err)
		require.Len(t, agents, 2)
		assert.Equal(t, "agent_1", agents[0].ID)
		assert.Equal(t, "A", agents[0].Name)
		require.Len(t, httpCtx.Requests, 1)
		assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
		assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/agents"))
		assert.Equal(t, anthropicBetaManagedAgents, httpCtx.Requests[0].Header.Get("anthropic-beta"))
	})

	t.Run("follows pagination", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"agent_1","name":"A"}],"next_page":"page_2"}`))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"agent_2","name":"B"}]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		agents, err := client.ListAgents()
		require.NoError(t, err)
		require.Len(t, agents, 2)
		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "page_2", httpCtx.Requests[1].URL.Query().Get("page"))
	})

	t.Run("stops when next_page repeats", func(t *testing.T) {
		// A misbehaving server that always returns the same next_page cursor
		// must not cause an infinite loop.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"agent_1","name":"A"}],"next_page":"page_2"}`))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"agent_2","name":"B"}],"next_page":"page_2"}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		agents, err := client.ListAgents()
		require.NoError(t, err)
		require.Len(t, agents, 2)
		require.Len(t, httpCtx.Requests, 2)
	})
}

func TestClient_GetAgent(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"agent_1","name":"Coding Assistant"}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
	agent, err := client.GetAgent("agent_1")
	require.NoError(t, err)
	assert.Equal(t, "agent_1", agent.ID)
	assert.Equal(t, "Coding Assistant", agent.Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.Equal(t, http.MethodGet, httpCtx.Requests[0].Method)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/agents/agent_1"))

	_, err = client.GetAgent("")
	require.Error(t, err)
}

func TestClient_GetEnvironment(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"id":"env_1","name":"python-dev"}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
	env, err := client.GetEnvironment("env_1")
	require.NoError(t, err)
	assert.Equal(t, "env_1", env.ID)
	assert.Equal(t, "python-dev", env.Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/environments/env_1"))

	_, err = client.GetEnvironment("")
	require.Error(t, err)
}

func TestClient_ListEnvironments(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{{
			StatusCode: 200,
			Body:       io.NopCloser(strings.NewReader(`{"data":[{"id":"env_1","name":"dev"}]}`)),
		}},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
	envs, err := client.ListEnvironments()
	require.NoError(t, err)
	require.Len(t, envs, 1)
	assert.Equal(t, "env_1", envs[0].ID)
	assert.Equal(t, "dev", envs[0].Name)
	require.Len(t, httpCtx.Requests, 1)
	assert.True(t, strings.HasSuffix(httpCtx.Requests[0].URL.Path, "/environments"))
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
