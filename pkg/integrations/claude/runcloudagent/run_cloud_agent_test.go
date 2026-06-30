package runcloudagent

import (
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/test/support/contexts"
)

func Test__RunCloudAgent__Setup(t *testing.T) {
	a := &RunCloudAgent{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":         "agent_01",
			"environmentId": "env_01",
			"prompt":        "Do the thing",
		},
		Integration: &contexts.IntegrationContext{},
		Metadata:    &contexts.MetadataContext{},
	}
	require.NoError(t, a.Setup(ctx))
}

func Test__RunCloudAgent__Setup__resolvesNodeMetadata(t *testing.T) {
	a := &RunCloudAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"agent_01","name":"My Agent"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"env_01","name":"My Env"}`))},
		},
	}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":         "agent_01",
			"environmentId": "env_01",
			"prompt":        "Do the thing",
		},
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		HTTP:        httpContext,
		Metadata:    metadataCtx,
	}
	require.NoError(t, a.Setup(ctx))

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "agent_01", md.AgentID)
	assert.Equal(t, "My Agent", md.AgentName)
	assert.Equal(t, "env_01", md.EnvironmentID)
	assert.Equal(t, "My Env", md.EnvironmentName)
	require.Len(t, httpContext.Requests, 2)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/agents/agent_01")
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/environments/env_01")
}

func Test__RunCloudAgent__Setup__nodeMetadataFallsBackToIDs(t *testing.T) {
	a := &RunCloudAgent{}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":         "agent_01",
			"environmentId": "env_01",
			"prompt":        "Do the thing",
		},
		// No apiKey: client creation fails, so names fall back to the IDs.
		Integration: &contexts.IntegrationContext{},
		Metadata:    metadataCtx,
	}
	require.NoError(t, a.Setup(ctx))

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "agent_01", md.AgentName)
	assert.Equal(t, "env_01", md.EnvironmentName)
}

func Test__RunCloudAgent__Setup__missingAgent(t *testing.T) {
	a := &RunCloudAgent{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"environmentId": "env_01",
			"prompt":        "x",
		},
		Integration: &contexts.IntegrationContext{},
		Metadata:    &contexts.MetadataContext{},
	}
	err := a.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "agent")
}

func Test__RunCloudAgent__Setup__missingEnvironment(t *testing.T) {
	a := &RunCloudAgent{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":  "agent_01",
			"prompt": "x",
		},
		Integration: &contexts.IntegrationContext{},
		Metadata:    &contexts.MetadataContext{},
	}
	err := a.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "environment")
}

func Test__RunCloudAgent__Execute__syncIdle(t *testing.T) {
	a := &RunCloudAgent{}
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
	assert.Equal(t, "claude.runCloudAgent", executionState.Type)
	assert.Equal(t, "idle", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
	assert.Equal(t, "Done", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).LastMessage)
	assert.Equal(t, "", requestsCtx.Action)

	require.Len(t, httpContext.Requests, 5) // create, send, get status, get events, delete
	assert.Equal(t, "POST", httpContext.Requests[0].Method)
	assert.Contains(t, httpContext.Requests[0].URL.Path, "/sessions")
	assert.Contains(t, httpContext.Requests[1].URL.Path, "/events")
}

func Test__RunCloudAgent__Execute__repositoryPrompt(t *testing.T) {
	a := &RunCloudAgent{}
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
		Configuration:  map[string]any{"agent": "a", "environmentId": "e", "prompt": "Fix the bug", "repository": "https://github.com/owner/repo.git", "branch": "main"},
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

	// The send-events request carries the clone instruction + task.
	sendReq := findRequest(httpContext.Requests, "/events")
	require.NotNil(t, sendReq)
	sendBody, _ := io.ReadAll(sendReq.Body)
	body := string(sendBody)
	assert.Contains(t, body, "Clone the git repository https://github.com/owner/repo.git")
	assert.Contains(t, body, "main")
	assert.Contains(t, body, "Fix the bug")

	// Repository/branch are persisted in metadata.
	md := ExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "https://github.com/owner/repo.git", md.Repository)
	assert.Equal(t, "main", md.Branch)
}

func Test__RunCloudAgent__Execute__schedulesPoll(t *testing.T) {
	a := &RunCloudAgent{}
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

// findRequest returns the first captured request whose path contains substr.
func findRequest(requests []*http.Request, substr string) *http.Request {
	for _, r := range requests {
		if strings.Contains(r.URL.Path, substr) {
			return r
		}
	}
	return nil
}

func Test__RunCloudAgent__poll__terminal(t *testing.T) {
	a := &RunCloudAgent{}
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

func Test__RunCloudAgent__poll__timeout(t *testing.T) {
	a := &RunCloudAgent{}
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

func Test__RunCloudAgent__scheduleNextPoll(t *testing.T) {
	a := &RunCloudAgent{}
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

func Test__buildRepositoryPrompt(t *testing.T) {
	t.Run("no repository returns the task unchanged", func(t *testing.T) {
		assert.Equal(t, "Do it", buildRepositoryPrompt("", "", "Do it"))
		assert.Equal(t, "Do it", buildRepositoryPrompt("   ", "main", "Do it"))
	})

	t.Run("repository without branch", func(t *testing.T) {
		got := buildRepositoryPrompt("https://github.com/o/r.git", "", "Do it")
		assert.Contains(t, got, "Clone the git repository https://github.com/o/r.git into your working directory")
		assert.NotContains(t, got, "check out")
		assert.Contains(t, got, "Do it")
	})

	t.Run("repository with branch", func(t *testing.T) {
		got := buildRepositoryPrompt("https://github.com/o/r.git", "develop", "Do it")
		assert.Contains(t, got, "check out the \"develop\" branch")
		assert.Contains(t, got, "Do it")
	})
}
