package runcodeagent

import (
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

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
		{"invalid repository", func(c map[string]any) { c["repository"] = "nonsense" }, "owner/repo or an https"},
		{"pr mode missing prUrl", func(c map[string]any) { c["sourceMode"] = "pr"; delete(c, "repository") }, "prUrl is required"},
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

func Test__RunCodeAgent__Setup__structuredOutputInvalidSchema(t *testing.T) {
	a := &RunCodeAgent{}
	cfg := repoConfig()
	cfg["outputSchema"] = `{"type":"object"}` // missing "properties"
	err := a.Setup(core.SetupContext{Configuration: cfg, Integration: &contexts.IntegrationContext{}, Metadata: &contexts.MetadataContext{}})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "properties")
}

func Test__RunCodeAgent__Setup__structuredOutputMetadata(t *testing.T) {
	a := &RunCodeAgent{}
	cfg := repoConfig()
	cfg["outputSchema"] = `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	metadataCtx := &contexts.MetadataContext{}
	require.NoError(t, a.Setup(core.SetupContext{Configuration: cfg, Integration: &contexts.IntegrationContext{}, Metadata: metadataCtx}))

	md := NodeMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.True(t, md.StructuredOutput)
}

func Test__RunCodeAgent__validateRepository(t *testing.T) {
	valid := []string{"owner/repo", "https://github.com/owner/repo.git", "{{ event.repo }}"}
	for _, v := range valid {
		assert.NoError(t, validateRepository(v), v)
	}
	// Non-github hosts, ssh:// and git:// are rejected: the GitHub token is
	// embedded in the clone URL, so it must only ever target github.com.
	invalid := []string{
		"not a repo", "owner/repo extra", "https://github.com/o/r.git\ninjection",
		"ssh://git@github.com/o/r.git", "git://github.com/o/r.git",
		"https://evil.com/o/r.git", "https://gitlab.com/o/r.git",
	}
	for _, v := range invalid {
		assert.Error(t, validateRepository(v), v)
	}
}

func Test__RunCodeAgent__authenticatedCloneURL(t *testing.T) {
	assert.Equal(t, "https://x-access-token:$GITHUB_TOKEN@github.com/owner/repo.git", authenticatedCloneURL("owner/repo"))
	assert.Equal(t, "https://x-access-token:$GITHUB_TOKEN@github.com/o/r.git", authenticatedCloneURL("https://github.com/o/r.git"))
	// A non-github URL must never receive the token.
	assert.Equal(t, "https://evil.com/o/r.git", authenticatedCloneURL("https://evil.com/o/r.git"))
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
	got := buildPrompt(spec, nil, "claude/agent-abc", false, commitAttribution{}, nil)
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
	got := buildPrompt(spec, nil, "claude/agent-abc", false, attr, nil)
	assert.Contains(t, got, `git config user.name "Octo Cat"`)
	assert.Contains(t, got, `git config user.email "1+octocat@users.noreply.github.com"`)
	assert.Contains(t, got, "Co-Authored-By")
}

func Test__RunCodeAgent__buildPrompt__pr(t *testing.T) {
	pr := &pullRequestInfo{BaseRepo: "owner/repo", HeadRef: "feature-x", HTMLURL: "https://github.com/owner/repo/pull/9"}
	got := buildPrompt(Spec{SourceMode: "pr", Task: "address review"}, pr, "feature-x", false, commitAttribution{}, nil)
	assert.Contains(t, got, "git switch feature-x")
	assert.Contains(t, got, "do NOT open a new pull request")
	assert.Contains(t, got, "address review")
	assert.Contains(t, got, "https://github.com/owner/repo/pull/9")
}

func Test__RunCodeAgent__buildPrompt__structuredOutput(t *testing.T) {
	schema := map[string]any{
		"type":       "object",
		"properties": map[string]any{"summary": map[string]any{"type": "string"}},
		"required":   []any{"summary"},
	}
	spec := Spec{SourceMode: "repository", Repository: "owner/repo", Task: "fix the bug"}
	got := buildPrompt(spec, nil, "claude/agent-abc", false, commitAttribution{}, schema)

	assert.Contains(t, got, "fenced json code block")
	assert.Contains(t, got, `"summary"`)
	// The structured-output instructions must precede the FINAL line marker,
	// so the marker instruction is still the last thing the agent is told.
	assert.Less(t, strings.Index(got, "fenced json code block"), strings.Index(got, "output on the FINAL line"))
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

func Test__RunCodeAgent__Execute__structuredOutputInPrompt(t *testing.T) {
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
	cfg := repoConfig()
	cfg["outputSchema"] = `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`
	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  cfg,
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

	var sendReq *http.Request
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/events") {
			sendReq = r
		}
	}
	require.NotNil(t, sendReq)
	body, _ := io.ReadAll(sendReq.Body)
	// Structured-output instructions and the PR_URL marker must both be present,
	// and in the order that keeps the marker the true final line.
	assert.Contains(t, string(body), "fenced json code block")
	assert.Contains(t, string(body), `\"summary\"`)
	assert.Contains(t, string(body), "PR_URL=")
	assert.Less(t, strings.Index(string(body), "fenced json code block"), strings.Index(string(body), "PR_URL="))
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

func Test__RunCodeAgent__Execute__prModeSchedulesPoll(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"number":5,"state":"open","html_url":"https://github.com/o/r/pull/5","head":{"ref":"feature","repo":{"full_name":"o/r"}},"base":{"ref":"main","repo":{"full_name":"o/r"}}}`), // resolve PR
		resp(`{"id":"agent_1"}`),                   // create agent
		resp(`{"id":"env_1"}`),                     // create environment
		resp(`{"id":"vault_1"}`),                   // create vault
		resp(`{}`),                                 // credential
		resp(`{"id":"sess_1","status":"running"}`), // create session
		resp(`{}`),                                 // send message
		resp(`{"id":"sess_1","status":"running"}`), // get session (fast-path)
	}}
	metadataCtx := &contexts.MetadataContext{}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	requestsCtx := &contexts.RequestContext{}
	execCtx := core.ExecutionContext{
		ID: uuid.New(),
		Configuration: map[string]any{
			"sourceMode":  "pr",
			"prUrl":       "https://github.com/o/r/pull/5",
			"task":        "address the review",
			"githubToken": map[string]any{"secret": "gh", "key": "token"},
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"gh/token": []byte("ghp_123")}},
		Metadata:       metadataCtx,
		ExecutionState: execState,
		Requests:       requestsCtx,
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.Execute(execCtx))
	assert.Equal(t, "poll", requestsCtx.Action)

	md := ExecutionMetadata{}
	require.NoError(t, mapstructure.Decode(metadataCtx.Get(), &md))
	assert.Equal(t, "https://github.com/o/r/pull/5", md.PrURL)
	assert.Equal(t, "feature", md.Branch)

	// The prompt updates the PR branch in place, not a new PR.
	var sendReq *http.Request
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodPost && strings.Contains(r.URL.Path, "/events") {
			sendReq = r
		}
	}
	require.NotNil(t, sendReq)
	body, _ := io.ReadAll(sendReq.Body)
	assert.Contains(t, string(body), "git switch feature")
	assert.Contains(t, string(body), "do NOT open a new pull request")
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
		resp(`{"data":[{"id":"file_out1","filename":"migration-notes.md","mime_type":"text/markdown","size_bytes":2048,"downloadable":true}]}`), // session artifacts
		resp("# Migration notes\n"),                    // artifact content
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
	require.Len(t, out.Artifacts, 1)
	assert.Equal(t, "file_out1", out.Artifacts[0].FileID)
	assert.Equal(t, "migration-notes.md", out.Artifacts[0].Filename)
	assert.Equal(t, "text", out.Artifacts[0].Encoding)
	assert.Equal(t, "# Migration notes\n", out.Artifacts[0].Content)
}

func Test__RunCodeAgent__poll__structuredOutput(t *testing.T) {
	a := &RunCodeAgent{}
	agentMessage := "Done.\n\n```json\n{\"summary\": \"fixed the bug\"}\n```\n\nPR_URL=https://github.com/o/r/pull/7"
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"idle"}`),
		resp(fmt.Sprintf(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":%q}]}]}`, agentMessage)),
		resp(`{"data":[]}`),                            // session artifacts (empty)
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:       "poll",
		Parameters: map[string]any{"attempt": float64(1), "errors": float64(0)},
		Configuration: map[string]any{
			"sourceMode":   "repository",
			"repository":   "o/r",
			"task":         "do it",
			"githubToken":  map[string]any{"secret": "gh", "key": "token"},
			"outputSchema": `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`,
		},
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

	parsed, ok := out.Parsed.(map[string]any)
	require.True(t, ok, "expected Parsed to be an object, got %T", out.Parsed)
	assert.Equal(t, "fixed the bug", parsed["summary"])
}

func Test__RunCodeAgent__poll__persistSessionKeepsSessionAndEnvironment(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"idle"}`),
		resp(`{"data":[{"type":"session.status_idle"},{"type":"agent.message","content":[{"type":"text","text":"Done. PR_URL=https://github.com/o/r/pull/7"}]}]}`),
		resp(`{}`), resp(`{}`), // teardown: vault delete + agent archive only
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:       "poll",
		Parameters: map[string]any{"attempt": float64(1), "errors": float64(0)},
		Configuration: map[string]any{
			"sourceMode":     "repository",
			"repository":     "o/r",
			"task":           "do it",
			"githubToken":    map[string]any{"secret": "gh", "key": "token"},
			"persistSession": true,
		},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       terminalMeta(),
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "idle", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)

	var sessionDeleted, envDeleted, vaultDeleted bool
	for _, r := range httpCtx.Requests {
		if r.Method != http.MethodDelete {
			continue
		}
		switch {
		case strings.Contains(r.URL.Path, "/sessions/sess_1"):
			sessionDeleted = true
		case strings.Contains(r.URL.Path, "/environments/env_1"):
			envDeleted = true
		case strings.Contains(r.URL.Path, "/vaults/vault_1"):
			vaultDeleted = true
		}
	}
	assert.False(t, sessionDeleted, "session must be kept when persistSession is enabled")
	assert.False(t, envDeleted, "environment must outlive a kept session")
	assert.True(t, vaultDeleted, "vault holds the GitHub token and must always be reclaimed")
}

// A cancelled run never finished, and cancelling often accompanies deleting the
// node — which drops the only record of these IDs. Reclaim regardless.
func Test__RunCodeAgent__Cancel__reclaimsDespitePersistSession(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // interrupt, session, env, vault, archive
	}}
	execCtx := core.ExecutionContext{
		Configuration: map[string]any{
			"sourceMode":     "repository",
			"repository":     "o/r",
			"task":           "do it",
			"githubToken":    map[string]any{"secret": "gh", "key": "token"},
			"persistSession": true,
		},
		HTTP:        httpCtx,
		Integration: &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:    terminalMeta(),
		Logger:      logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.Cancel(execCtx))

	var sessionDeleted, envDeleted bool
	for _, r := range httpCtx.Requests {
		if r.Method != http.MethodDelete {
			continue
		}
		switch {
		case strings.Contains(r.URL.Path, "/sessions/sess_1"):
			sessionDeleted = true
		case strings.Contains(r.URL.Path, "/environments/env_1"):
			envDeleted = true
		}
	}
	assert.True(t, sessionDeleted, "cancel must reclaim the session even when persistSession is enabled")
	assert.True(t, envDeleted, "cancel must reclaim the environment even when persistSession is enabled")
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

func Test__RunCodeAgent__poll__errorReclaims(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		{StatusCode: 500, Body: io.NopCloser(strings.NewReader(`{"error":{"message":"boom"}}`))}, // get session fails
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown (interrupt, delete session, env, vault, archive)
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(2), "errors": float64(maxPollErrors - 1)},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       terminalMeta(),
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "error", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
	var deleted bool
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1") {
			deleted = true
		}
	}
	assert.True(t, deleted, "session should be reclaimed after repeated poll errors")
}

// failingEmitState wraps the execution-state double to force Emit to fail,
// without modifying the shared test-support package.
type failingEmitState struct {
	*contexts.ExecutionStateContext
	err error
}

func (f *failingEmitState) Emit(channel, payloadType string, payloads []any) error {
	return f.err
}

func Test__RunCodeAgent__poll__timeoutEmitFailurePreservesSession(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"running"}`), // get session (still running)
	}}
	execState := &failingEmitState{
		ExecutionStateContext: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		err:                   fmt.Errorf("boom"),
	}
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

	require.Error(t, a.HandleHook(hookCtx))
	assert.False(t, execState.Finished)
	// The session must NOT be deleted when the emit fails, so a retry can recover.
	for _, r := range httpCtx.Requests {
		assert.NotEqual(t, http.MethodDelete, r.Method, "session must survive an emit failure")
	}
}

func Test__RunCodeAgent__poll__missingSessionFinishesError(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{}`), resp(`{}`), resp(`{}`), // teardown of stray env/vault/agent
	}}
	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	metadataCtx := &contexts.MetadataContext{Metadata: ExecutionMetadata{
		AgentID: "agent_1", EnvironmentID: "env_1", VaultID: "vault_1", Branch: "b",
	}}
	hookCtx := core.ActionHookContext{
		Name:           "poll",
		Parameters:     map[string]any{"attempt": float64(1), "errors": float64(0)},
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Metadata:       metadataCtx,
		ExecutionState: execState,
		Requests:       &contexts.RequestContext{},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.NoError(t, a.HandleHook(hookCtx))
	require.True(t, execState.Finished)
	assert.Equal(t, "error", execState.Payloads[0].(map[string]any)["data"].(OutputPayload).Status)
}

// failingSchedule implements core.RequestContext with a ScheduleActionCall that
// always fails, without touching the shared test-support package.
type failingSchedule struct{ err error }

func (f *failingSchedule) ScheduleActionCall(actionName string, parameters map[string]any, interval time.Duration) error {
	return f.err
}

func Test__RunCodeAgent__Execute__reclaimsOnScheduleFailure(t *testing.T) {
	a := &RunCodeAgent{}
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"agent_1"}`),                                   // create agent
		resp(`{"id":"env_1"}`),                                     // create environment
		resp(`{"id":"vault_1"}`),                                   // create vault
		resp(`{}`),                                                 // credential
		resp(`{"id":"sess_1","status":"running"}`),                 // create session
		resp(`{}`),                                                 // send message
		resp(`{"id":"sess_1","status":"running"}`),                 // get session (fast-path, not terminal)
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown
	}}
	execCtx := core.ExecutionContext{
		ID:             uuid.New(),
		Configuration:  repoConfig(),
		HTTP:           httpCtx,
		Integration:    &contexts.IntegrationContext{Configuration: map[string]any{"apiKey": "k"}},
		Secrets:        &contexts.SecretsContext{Values: map[string][]byte{"gh/token": []byte("ghp_123")}},
		Metadata:       &contexts.MetadataContext{},
		ExecutionState: &contexts.ExecutionStateContext{KVs: map[string]string{}},
		Requests:       &failingSchedule{err: fmt.Errorf("schedule boom")},
		Logger:         logrus.NewEntry(logrus.New()),
	}

	require.Error(t, a.Execute(execCtx))
	var deleted bool
	for _, r := range httpCtx.Requests {
		if r.Method == http.MethodDelete && strings.Contains(r.URL.Path, "/sessions/sess_1") {
			deleted = true
		}
	}
	assert.True(t, deleted, "resources must be reclaimed when the poll cannot be scheduled")
}

func Test__RunCodeAgent__resolvePullRequestForRun__missingRef(t *testing.T) {
	ctx := core.ExecutionContext{HTTP: &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"state":"open","head":{"ref":"","repo":{"full_name":"o/r"}},"base":{"ref":"main","repo":{"full_name":"o/r"}}}`),
	}}}
	_, err := resolvePullRequestForRun(ctx, Spec{PrURL: "https://github.com/o/r/pull/1"}, "tok")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "missing its base repository or head branch")
}

func Test__RunCodeAgent__poll__terminalWithUnavailableEventsPastBudget(t *testing.T) {
	a := &RunCodeAgent{}
	// The session is terminal but its events are unavailable and the poll
	// budget is exhausted: the run must still finish (no artifacts) instead
	// of panicking on the missing session messages.
	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"idle"}`),
		{StatusCode: http.StatusInternalServerError, Body: io.NopCloser(strings.NewReader(`boom`))}, // events fetch fails
		resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`), // teardown
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
	out := execState.Payloads[0].(map[string]any)["data"].(OutputPayload)
	assert.Equal(t, "idle", out.Status)
	assert.Nil(t, out.Artifacts)
}

func Test__RunCodeAgent__poll__terminalWithIncompleteEventsSkipsStructuredOutput(t *testing.T) {
	a := &RunCodeAgent{}
	restore := finalMessageDelay
	finalMessageDelay = time.Millisecond
	t.Cleanup(func() { finalMessageDelay = restore })

	httpCtx := &contexts.HTTPContext{Responses: []*http.Response{
		resp(`{"id":"sess_1","status":"idle"}`),
	}}
	// Events never include session.status_idle, so Complete never becomes
	// true, even though the message looks like valid structured-output JSON.
	for range finalMessageReads {
		httpCtx.Responses = append(httpCtx.Responses, resp(
			`{"data":[{"type":"agent.message","content":[{"type":"text","text":"{\"summary\":\"partial\"}"}]}]}`))
	}
	httpCtx.Responses = append(httpCtx.Responses, resp(`{}`), resp(`{}`), resp(`{}`), resp(`{}`)) // teardown

	execState := &contexts.ExecutionStateContext{KVs: map[string]string{}}
	hookCtx := core.ActionHookContext{
		Name:       "poll",
		Parameters: map[string]any{"attempt": float64(maxPollAttempts + 1), "errors": float64(0)},
		Configuration: map[string]any{
			"sourceMode":   "repository",
			"repository":   "o/r",
			"task":         "do it",
			"githubToken":  map[string]any{"secret": "gh", "key": "token"},
			"outputSchema": `{"type":"object","properties":{"summary":{"type":"string"}},"required":["summary"]}`,
		},
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
	assert.NotEmpty(t, out.LastMessage, "the partial message we did collect must still be emitted")
	assert.Nil(t, out.Parsed, "incomplete events must never be trusted for structured output")
}
