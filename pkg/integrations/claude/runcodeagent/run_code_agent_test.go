package runcodeagent

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

func resp(body string) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(body))}
}

func repoConfig() map[string]any {
	return map[string]any{
		"sourceMode":  "repository",
		"repository":  "owner/repo",
		"task":        "do the thing",
		"githubToken": map[string]any{"secret": "gh", "key": "token"},
	}
}

// --- Setup / validation ---

func Test__RunCodeAgent__Setup__valid(t *testing.T) {
	a := &RunCodeAgent{}
	metadataCtx := &contexts.MetadataContext{}
	ctx := core.SetupContext{Configuration: repoConfig(), Integration: &contexts.IntegrationContext{}, Metadata: metadataCtx}
	require.NoError(t, a.Setup(ctx))

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "owner/repo", md.Repository)
	assert.Equal(t, "repository", md.SourceMode)
	assert.Equal(t, defaultModel, md.Model)
}

func Test__RunCodeAgent__Setup__validation(t *testing.T) {
	a := &RunCodeAgent{}
	cases := []struct {
		name    string
		mutate  func(map[string]any)
		wantErr string
	}{
		{"missing task", func(c map[string]any) { delete(c, "task") }, "task is required"},
		{"missing token", func(c map[string]any) { delete(c, "githubToken") }, "githubToken is required"},
		{"repo mode missing repository", func(c map[string]any) { delete(c, "repository") }, "repository is required"},
		{"invalid repository", func(c map[string]any) { c["repository"] = "nonsense" }, "valid git URL"},
		{"pr mode missing prUrl", func(c map[string]any) { c["sourceMode"] = "pr"; delete(c, "repository") }, "prUrl is required"},
		{"pr mode invalid prUrl", func(c map[string]any) { c["sourceMode"] = "pr"; c["prUrl"] = "nope" }, "invalid pull request URL"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			cfg := repoConfig()
			tc.mutate(cfg)
			err := a.Setup(core.SetupContext{Configuration: cfg, Integration: &contexts.IntegrationContext{}, Metadata: &contexts.MetadataContext{}})
			require.Error(t, err)
			assert.Contains(t, err.Error(), tc.wantErr)
		})
	}
}

func Test__RunCodeAgent__validateRepository(t *testing.T) {
	valid := []string{"owner/repo", "https://github.com/owner/repo.git", "ssh://git@github.com/o/r.git", "{{ event.repo }}"}
	for _, v := range valid {
		assert.NoError(t, validateRepository(v), v)
	}
	invalid := []string{"not a repo", "owner/repo extra", "https://github.com/o/r.git\ninjection"}
	for _, v := range invalid {
		assert.Error(t, validateRepository(v), v)
	}
}

func Test__RunCodeAgent__parsePRURL(t *testing.T) {
	owner, repo, number, err := parsePRURL("https://github.com/acme/widgets/pull/42")
	require.NoError(t, err)
	assert.Equal(t, "acme", owner)
	assert.Equal(t, "widgets", repo)
	assert.Equal(t, 42, number)

	_, _, _, err = parsePRURL("https://github.com/acme/widgets/issues/42")
	require.Error(t, err)
}

func Test__RunCodeAgent__extractPRURL(t *testing.T) {
	assert.Equal(t, "https://github.com/o/r/pull/7",
		extractPRURL([]string{"working..."}, "Done. PR_URL=https://github.com/o/r/pull/7"))
	assert.Equal(t, "", extractPRURL([]string{"no marker here"}, "NO_PR"))
	assert.Equal(t, "", extractPRURL(nil, ""))
}

func Test__RunCodeAgent__buildEnvironmentConfig(t *testing.T) {
	unrestricted := buildEnvironmentConfig(Spec{Networking: networkingUnrestricted})
	assert.Equal(t, "unrestricted", unrestricted.Networking.Type)

	limited := buildEnvironmentConfig(Spec{Networking: networkingLimited, AllowedHosts: []string{"registry.internal"}})
	assert.Equal(t, "limited", limited.Networking.Type)
	assert.Contains(t, limited.Networking.AllowedHosts, "github.com")
	assert.Contains(t, limited.Networking.AllowedHosts, "registry.internal")
	require.NotNil(t, limited.Networking.AllowPackageManagers)
	assert.True(t, *limited.Networking.AllowPackageManagers)
}

func Test__RunCodeAgent__buildPrompt__repository(t *testing.T) {
	spec := Spec{SourceMode: "repository", Repository: "owner/repo", Task: "fix the bug", BaseBranch: "main"}
	got := buildPrompt(spec, nil, "claude/agent-abc", false, commitAttribution{})
	assert.Contains(t, got, "git clone https://x-access-token:$GITHUB_TOKEN@github.com/owner/repo.git")
	assert.Contains(t, got, "git checkout -b claude/agent-abc")
	assert.Contains(t, got, "fix the bug")
	assert.Contains(t, got, "Open a pull request")
	assert.Contains(t, got, "PR_URL=")
	// Acting as bot (default): no git identity config, no trailer suppression.
	assert.NotContains(t, got, "git config user.name")
	assert.NotContains(t, got, "Co-Authored-By")
}

func Test__RunCodeAgent__buildPrompt__actAsUser(t *testing.T) {
	spec := Spec{SourceMode: "repository", Repository: "owner/repo", Task: "fix the bug"}
	attr := commitAttribution{AuthorName: "Octo Cat", AuthorEmail: "1+octocat@users.noreply.github.com"}
	got := buildPrompt(spec, nil, "claude/agent-abc", false, attr)
	assert.Contains(t, got, `git config user.name "Octo Cat"`)
	assert.Contains(t, got, `git config user.email "1+octocat@users.noreply.github.com"`)
	assert.Contains(t, got, "Co-Authored-By")
}

func Test__RunCodeAgent__buildPrompt__pr(t *testing.T) {
	pr := &pullRequestInfo{BaseRepo: "owner/repo", HeadRef: "feature-x", HTMLURL: "https://github.com/owner/repo/pull/9"}
	got := buildPrompt(Spec{SourceMode: "pr", Task: "address review"}, pr, "feature-x", false, commitAttribution{})
	assert.Contains(t, got, "git checkout feature-x")
	assert.Contains(t, got, "do NOT open a new pull request")
	assert.Contains(t, got, "address review")
	assert.Contains(t, got, "https://github.com/owner/repo/pull/9")
}

func Test__RunCodeAgent__actAsBot(t *testing.T) {
	no := false
	yes := true
	assert.True(t, actAsBot(Spec{})) // default
	assert.True(t, actAsBot(Spec{ActAsBot: &yes}))
	assert.False(t, actAsBot(Spec{ActAsBot: &no}))
}

// --- Execute ---

func Test__RunCodeAgent__Execute__repositoryMode_schedulesPoll(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"agent_1"}`),                   // create agent
		resp(`{"id":"env_1"}`),                     // create environment
		resp(`{"id":"vault_1"}`),                   // create vault
		resp(`{}`),                                 // add GITHUB_TOKEN credential
		resp(`{"id":"sess_1","status":"running"}`), // create session
		resp(`{}`),                                 // send message
		resp(`{"id":"sess_1","status":"running"}`), // get session (fast-path check)
	}}
	metadataCtx := &contexts.MetadataContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}
	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  repoConfig(),
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"gh/token": []byte("ghp_123")}},
		Metadata:       metadataCtx,
		ExecutionState: execState,
		Requests:       requestsCtx,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.Execute(execCtx))
	assert.False(t, execState.Finished)
	assert.Equal(t, "poll", requestsCtx.Action)

	md := ExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "agent_1", md.AgentID)
	assert.Equal(t, "env_1", md.EnvironmentID)
	assert.Equal(t, "vault_1", md.VaultID)
	assert.Equal(t, "sess_1", md.Session.ID)
	assert.Equal(t, "owner/repo", md.Repository)
	assert.True(t, strings.HasPrefix(md.Branch, "claude/agent-"))

	// Sanity: the send-events body carries the clone + task.
	var sendReq *http.Request
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/events") {
			sendReq = r
		}
	}
	require.NotNil(t, sendReq)
	body, _ := io.ReadAll(sendReq.Body)
	assert.Contains(t, string(body), "git clone")
	assert.Contains(t, string(body), "do the thing")
}

func Test__RunCodeAgent__Execute__cleansUpOnSessionFailure(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"agent_1"}`), // create agent
		resp(`{"id":"env_1"}`),   // create environment
		resp(`{"id":"vault_1"}`), // create vault
		resp(`{}`),               // credential
		{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`))}, // create session fails
		resp(`{}`), // teardown: delete environment
		resp(`{}`), // teardown: delete vault
		resp(`{}`), // teardown: archive agent
	}}
	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  repoConfig(),
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"gh/token": []byte("ghp_123")}},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.Error(t, a.Execute(execCtx))

	var envDeleted, vaultDeleted, agentArchived bool
	for _, r := range httpCtx.Requests {
		switch {
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/environments/env_1"):
			envDeleted = true
		case r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/vaults/vault_1"):
			vaultDeleted = true
		case r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/agents/agent_1/archive"):
			agentArchived = true
		}
	}
	assert.True(t, envDeleted, "environment should be deleted")
	assert.True(t, vaultDeleted, "vault should be deleted")
	assert.True(t, agentArchived, "agent should be archived")
}

// --- poll ---

func terminalMeta() *contexts.MetadataContext {
	return &contexts.MetadataContext{Metadata: ExecutionMetadata{
		Session:       &SessionMetadata{ID: "sess_1", Status: "running"},
		AgentID:       "agent_1",
		EnvironmentID: "env_1",
		VaultID:       "vault_1",
		Branch:        "claude/agent-abc",
	}}
}

func Test__RunCodeAgent__poll__terminalExtractsPR(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"idle"}`),
		resp(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Done. PR_URL=https://github.com/o/r/pull/7"}]}]}`),
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       terminalMeta(),
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	out := execState.Payloads[0].(map[string]any)["data"].(OutputPayload)
	assert.Equal(t, "idle", out.Status)
	assert.Equal(t, "https://github.com/o/r/pull/7", out.PrURL)
	assert.Equal(t, "claude/agent-abc", out.Branch)
}

func Test__RunCodeAgent__poll__timeoutReclaims(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"running"}`), // still running
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown (interrupt+delete+env+vault+agent)
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(maxPollAttempts + 1), "errors": float64(0)},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       terminalMeta(),
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "timeout", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	var deleted bool
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1") {
			deleted = true
		}
	}
	assert.True(t, deleted, "session should be deleted on timeout")
}

func Test__RunCodeAgent__poll__clientErrorReportsError(t *testing.T) {
	a := &RunCodeAgent{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(2), "errors": float64(maxPollErrors - 1)},
		Integration:    &contexts.IntegrationContext{}, // no apiKey -> client creation fails
		Metadata:       terminalMeta(),
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "error", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
}
