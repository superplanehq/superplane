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

// A node stores the environment under the legacy "environmentId" key and the
// version as a number; both must decode.
func Test__decodeSpec(t *testing.T) {
	spec, err := decodeSpec(map[string]any{
		"agent":         "agent_1",
		"environmentId": "env_1",
		"version":       float64(2), // JSON numbers decode to float64
		"prompt":        "do it",
	})
	require.NoError(t, err)
	assert.Equal(t, "agent_1", spec.Agent)
	assert.Equal(t, "env_1", spec.Environment)
	require.NotNil(t, spec.Version)
	assert.Equal(t, 2, *spec.Version)

	// An unset version stays nil (the agent's latest is used).
	spec, err = decodeSpec(map[string]any{"agent": "a", "environmentId": "e", "prompt": "p"})
	require.NoError(t, err)
	assert.Nil(t, spec.Version)
}

func Test__RunAgent__Execute__syncIdle(t *testing.T) {
	a := &RunAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Done"}]},{"type":"user.message","content":[{"type":"text","text":"Hello"}]}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_out1","filename":"report.md","mime_type":"text/markdown","size_bytes":4096,"downloadable":true},{"id":"file_in1","filename":"input.txt","mime_type":"text/plain","size_bytes":10,"downloadable":false}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("# Report\n"))},
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
	out := executionState.Payloads[0].(map[string]any)["data"].(OutputPayload)
	assert.Equal(t, "idle", out.Status)
	assert.Equal(t, "Done", out.LastMessage)
	assert.Equal(t, "", requestsCtx.Action)

	// Only the downloadable (agent-generated) file becomes an artifact, and
	// its content is embedded in the payload.
	require.Len(t, out.Artifacts, 1)
	assert.Equal(t, "file_out1", out.Artifacts[0].FileID)
	assert.Equal(t, "report.md", out.Artifacts[0].Filename)
	assert.Equal(t, "https://api.anthropic.com/v1/files/file_out1/content", out.Artifacts[0].DownloadURL)
	assert.Equal(t, "text", out.Artifacts[0].Encoding)
	assert.Equal(t, "# Report\n", out.Artifacts[0].Content)

	require.Len(t, httpContext.Requests, 7) // create, send, get status, get events, list files, download file, delete
	assert.Equal(t, "POST", httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/sessions")
	assert.Equal(t, anthropicBetaManagedAgents, httpContext.Requests[0].Header.Get("anthropic-beta"))
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/events")
	assert.Equal(t, "GET", httpContext.Requests[2].Method)
	assert.Equal(t, "GET", httpContext.Requests[3].Method)
	assert.Contains(t, httpContext.Requests[3].URL.Path, "/events")
	assert.Equal(t, "desc", httpContext.Requests[3].URL.Query().Get("order"))
	assert.Equal(t, sessionEventsPageLimit, httpContext.Requests[3].URL.Query().Get("limit"))
	assert.Contains(t, httpContext.Requests[4].URL.Path, "/files")
	assert.Equal(t, "sess_1", httpContext.Requests[4].URL.Query().Get("scope_id"))
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
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_out1","filename":"report.md","mime_type":"text/markdown","size_bytes":4096,"downloadable":true}]}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("# Report\n"))},
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
	out := executionState.Payloads[0].(map[string]any)["data"].(OutputPayload)
	assert.Equal(t, "idle", out.Status)
	assert.Equal(t, "Final", out.LastMessage)
	assert.Equal(t, []string{"Earlier", "Final"}, out.Messages)
	require.Len(t, out.Artifacts, 1)
	assert.Equal(t, "file_out1", out.Artifacts[0].FileID)
	assert.Equal(t, "text", out.Artifacts[0].Encoding)
	assert.Equal(t, "# Report\n", out.Artifacts[0].Content)
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

func TestClient_ListSessionFiles(t *testing.T) {
	t.Run("paginates with after_id", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_1","filename":"a.txt","downloadable":true}],"last_id":"file_1","has_more":true}`))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_2","filename":"b.txt","downloadable":true}],"last_id":"file_2","has_more":false}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		files, err := client.ListSessionFiles("sess_1")
		require.NoError(t, err)
		require.Len(t, files, 2)
		assert.Equal(t, "file_1", files[0].ID)
		assert.Equal(t, "file_2", files[1].ID)

		require.Len(t, httpCtx.Requests, 2)
		assert.Equal(t, "sess_1", httpCtx.Requests[0].URL.Query().Get("scope_id"))
		assert.Empty(t, httpCtx.Requests[0].URL.Query().Get("after_id"))
		assert.Equal(t, "file_1", httpCtx.Requests[1].URL.Query().Get("after_id"))
		assert.Equal(t, anthropicBetaManagedAgents, httpCtx.Requests[0].Header.Get("anthropic-beta"))
	})

	t.Run("requires session id", func(t *testing.T) {
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: &contexts.HTTPContext{}}
		_, err := client.ListSessionFiles("")
		require.Error(t, err)
	})

	t.Run("retry re-lists when empty", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_1","downloadable":true}]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		files, err := client.ListSessionFilesWithRetry("sess_1", 2, 0)
		require.NoError(t, err)
		require.Len(t, files, 1)
		require.Len(t, httpCtx.Requests, 2)
	})

	t.Run("retry re-lists when only mounted inputs are indexed", func(t *testing.T) {
		// Mounted inputs (never downloadable) can be indexed before the
		// agent's outputs; a listing with only inputs must not stop the retry.
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_in1","filename":"input.txt","downloadable":false}]}`))},
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"id":"file_in1","filename":"input.txt","downloadable":false},{"id":"file_out1","filename":"report.md","downloadable":true}]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		files, err := client.ListSessionFilesWithRetry("sess_1", 2, 0)
		require.NoError(t, err)
		require.Len(t, files, 2)
		require.Len(t, httpCtx.Requests, 2)
	})
}

func TestCollectSessionArtifacts_listingErrorYieldsNoArtifacts(t *testing.T) {
	httpCtx := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`boom`))},
		},
	}
	client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
	artifacts := CollectSessionArtifacts(client, "sess_1", true, nil)
	assert.Nil(t, artifacts)
}

func Test__SessionMessages__ExpectsArtifacts(t *testing.T) {
	t.Run("set when events mention the outputs directory", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Saved the report to /mnt/session/outputs/report.md"}]}]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		sm, err := client.GetSessionMessages("sess_1")
		require.NoError(t, err)
		assert.True(t, sm.ExpectsArtifacts)
	})

	t.Run("unset when events never mention it", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Done"}]}]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		sm, err := client.GetSessionMessages("sess_1")
		require.NoError(t, err)
		assert.False(t, sm.ExpectsArtifacts)
	})
}

func Test__CollectSessionArtifacts__retryOnlyWhenExpected(t *testing.T) {
	t.Run("no expected artifacts lists exactly once", func(t *testing.T) {
		httpCtx := &contexts.HTTPContext{
			Responses: []*http.Response{
				{StatusCode: 200, Body: io.NopCloser(strings.NewReader(`{"data":[]}`))},
			},
		}
		client := &Client{APIKey: "k", BaseURL: defaultBaseURL, http: httpCtx}
		artifacts := CollectSessionArtifacts(client, "sess_1", false, nil)
		assert.Nil(t, artifacts)
		// A single request and no sleep: artifact-less runs finish immediately.
		require.Len(t, httpCtx.Requests, 1)
	})
}
