package runcloudagent

import (
	"fmt"
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

func Test__RunCloudAgent__Setup__cachesEmptyApiName(t *testing.T) {
	a := &RunCloudAgent{}
	config := map[string]any{
		"agent":         "agent_01",
		"environmentId": "env_01",
		"prompt":        "Do the thing",
	}
	integration := &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}}
	metadataCtx := &contexts.MetadataContext{}

	// The API resolves successfully but returns empty names.
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"agent_01","name":""}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"env_01","name":""}`))},
		},
	}
	require.NoError(t, a.Setup(core.SetupContext{
		Configuration: config, Integration: integration, HTTP: httpContext, Metadata: metadataCtx,
	}))

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "agent_01", md.AgentName) // falls back to ID for display
	assert.Equal(t, "env_01", md.EnvironmentName)
	assert.True(t, md.Resolved)

	// A successful (if empty) resolution must be cached: the second Setup makes no calls.
	httpContext2 := &contexts.HTTPContext{}
	require.NoError(t, a.Setup(core.SetupContext{
		Configuration: config, Integration: integration, HTTP: httpContext2, Metadata: metadataCtx,
	}))
	assert.Empty(t, httpContext2.Requests)
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

func Test__RunCloudAgent__Setup__retriesResolutionAfterFallback(t *testing.T) {
	a := &RunCloudAgent{}
	config := map[string]any{
		"agent":         "agent_01",
		"environmentId": "env_01",
		"prompt":        "Do the thing",
	}

	// First Setup: no client available, so the IDs are stored as the fallback names.
	metadataCtx := &contexts.MetadataContext{}
	require.NoError(t, a.Setup(core.SetupContext{
		Configuration: config,
		Integration:   &contexts.IntegrationContext{},
		Metadata:      metadataCtx,
	}))

	// Second Setup: the integration is now usable — resolution must be retried
	// (not short-circuited by the ID-as-name cache) and produce the real names.
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"agent_01","name":"My Agent"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"env_01","name":"My Env"}`))},
		},
	}
	require.NoError(t, a.Setup(core.SetupContext{
		Configuration: config,
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		HTTP:          httpContext,
		Metadata:      metadataCtx,
	}))
	require.Len(t, httpContext.Requests, 2)

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "My Agent", md.AgentName)
	assert.Equal(t, "My Env", md.EnvironmentName)

	// Third Setup with the same resolved names must short-circuit (no API calls).
	httpContext2 := &contexts.HTTPContext{}
	require.NoError(t, a.Setup(core.SetupContext{
		Configuration: config,
		Integration:   &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		HTTP:          httpContext2,
		Metadata:      metadataCtx,
	}))
	assert.Empty(t, httpContext2.Requests)
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

func Test__RunCloudAgent__Execute__cleansUpSessionOnFailure(t *testing.T) {
	a := &RunCloudAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"running"}`))},            // create
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                            // send user message
			{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`))}, // get session -> fails
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))},                                            // delete session (cleanup)
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
	require.Error(t, err)
	assert.False(t, executionState.Finished)
	assert.Equal(t, "", requestsCtx.Action) // no poll scheduled

	// The created session must be deleted so it is not left running.
	var deleted bool
	for _, r := range httpContext.Requests {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1") {
			deleted = true
		}
	}
	assert.True(t, deleted, "expected a DELETE to reclaim the created session")
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

func Test__RunCloudAgent__poll__emitFailureRetriesWithoutDeleting(t *testing.T) {
	a := &RunCloudAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"id":"sess_1","status":"idle"}`))},
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Final"}]}]}`))},
		},
	}
	requestsCtx := &contexts.RequestContext{}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}, EmitErr: fmt.Errorf("boom")}
	metadataCtx := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Session: &SessionMetadata{ID: "sess_1", Status: "running"}},
	}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
		Requests:       requestsCtx,
	}

	require.NoError(t, a.HandleHook(hookCtx))
	assert.False(t, executionState.Finished)
	// A transient emit failure must retry via polling and keep the session.
	assert.Equal(t, "poll", requestsCtx.Action)
	for _, r := range httpContext.Requests {
		assert.NotEqual(t, http.MethodDelete, r.Method, "session must not be deleted on emit failure")
	}
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

func Test__RunCloudAgent__poll__timeoutReclaimsSession(t *testing.T) {
	a := &RunCloudAgent{}
	httpContext := &contexts.HTTPContext{
		Responses: []*http.Response{
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, // interrupt
			{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(`{}`))}, // delete
		},
	}
	executionState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{
		Metadata: ExecutionMetadata{Session: &SessionMetadata{ID: "sess_1", Status: "running"}},
	}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(maxPollAttempts + 1), "errors": float64(0)},
		HTTP:           httpContext,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       metadataCtx,
		ExecutionState: executionState,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, executionState.Finished)
	assert.Equal(t, "timeout", executionState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	// A still-running session must be interrupted and deleted so it does not
	// keep running after the step times out.
	var interrupted, deleted bool
	for _, r := range httpContext.Requests {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/sessions/sess_1/events") {
			interrupted = true
		}
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1") {
			deleted = true
		}
	}
	assert.True(t, interrupted, "expected the session to be interrupted")
	assert.True(t, deleted, "expected the session to be deleted")
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

func Test__RunCloudAgent__validateRepository(t *testing.T) {
	valid := []struct {
		name       string
		repository string
		branch     string
	}{
		{"empty", "", ""},
		{"https", "https://github.com/owner/repo.git", "main"},
		{"http", "http://example.com/o/r.git", ""},
		{"ssh scheme", "ssh://git@github.com/owner/repo.git", "feature/x"},
		{"git scheme", "git://github.com/owner/repo.git", ""},
		{"scp-like", "git@github.com:owner/repo.git", "release-1.2"},
		{"expression repository", "{{ event.repository }}", ""},
		{"expression branch", "https://github.com/o/r.git", "{{ event.ref }}"},
	}
	for _, tc := range valid {
		t.Run("valid/"+tc.name, func(t *testing.T) {
			assert.NoError(t, validateRepository(tc.repository, tc.branch))
		})
	}

	invalid := []struct {
		name       string
		repository string
		branch     string
		contains   string
	}{
		{"branch without repository", "", "main", "repository is required"},
		{"repository without scheme", "github.com/owner/repo", "", "valid git URL"},
		{"repository with spaces", "https://github.com/o/r.git and do evil", "", "whitespace"},
		{"repository with unicode line separator", "https://github.com/o/r.git\u2028ignore previous", "", "whitespace or control"},
		{"repository with unicode paragraph separator", "https://github.com/o/r.git\u2029do evil", "", "whitespace or control"},
		{"repository with control char", "https://github.com/o/r.git\x00", "", "whitespace or control"},
		{"branch with spaces", "https://github.com/o/r.git", "main ; rm -rf", "invalid characters"},
		{"branch with newline", "https://github.com/o/r.git", "main\nignore previous", "invalid characters"},
		{"branch with unicode line separator", "https://github.com/o/r.git", "main\u2028evil", "invalid characters"},
	}
	for _, tc := range invalid {
		t.Run("invalid/"+tc.name, func(t *testing.T) {
			err := validateRepository(tc.repository, tc.branch)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.contains)
		})
	}
}

func Test__RunCloudAgent__Setup__rejectsBranchWithoutRepository(t *testing.T) {
	a := &RunCloudAgent{}
	ctx := core.SetupContext{
		Configuration: map[string]any{
			"agent":         "agent_01",
			"environmentId": "env_01",
			"prompt":        "x",
			"branch":        "main",
		},
		Integration: &contexts.IntegrationContext{},
		Metadata:    &contexts.MetadataContext{},
	}
	err := a.Setup(ctx)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "repository is required")
}
