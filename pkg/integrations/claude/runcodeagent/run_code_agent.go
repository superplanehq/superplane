package runcodeagent

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

const (
	defaultModel        = "claude-sonnet-4-6"
	defaultSystemPrompt = "You are an autonomous software engineering agent. You work carefully, make focused changes, follow the repository's existing conventions, and verify your work before finishing."
)

type RunCodeAgent struct{}

func (a *RunCodeAgent) Name() string { return "claude.runCodeAgent" }

func (a *RunCodeAgent) Label() string { return "Run Code Agent" }

func (a *RunCodeAgent) Description() string {
	return "Runs an autonomous Claude coding agent against a repository in Anthropic's managed sandbox and opens (or updates) a pull request."
}

func (a *RunCodeAgent) Documentation() string {
	return `The **Run Code Agent** component runs an autonomous Claude coding agent using [Claude Managed Agents](https://platform.claude.com/docs/en/managed-agents/overview). You pick a repository (or an existing pull request) and describe a task; the component provisions a network-enabled sandbox with a scoped GitHub token, and the agent clones the repo, does the work, commits, pushes, and opens or updates a pull request — entirely on its own.

## Prerequisites

- A **Claude API key** on the integration.
- A **GitHub token** (stored as a SuperPlane secret) with permission to read the repo and open pull requests.

## Modes

- **Repository** — start new work: the agent branches from the base branch, implements the task, and opens a PR.
- **Pull request** — update an existing PR: the agent checks out the PR's branch, applies the task, and pushes to it. Only same-repository pull requests are supported; PRs opened from forks are rejected.

## Output

Emits the final **status**, the **pull request URL**, the working **branch**, and a **summary** so downstream steps can branch or post the result.

Files the agent saves under ` + "`/mnt/session/outputs/`" + ` are additionally emitted as **artifacts** with their content included in the payload (text files as plain text, everything else base64-encoded; files over 10MB carry metadata and a download link only).`
}

func (a *RunCodeAgent) Icon() string { return "bot" }

func (a *RunCodeAgent) Color() string { return "#C9784D" }

func (a *RunCodeAgent) ExampleOutput() map[string]any { return getExampleOutput() }

func (a *RunCodeAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{{Name: defaultChannel, Label: "Default"}}
}

func (a *RunCodeAgent) Configuration() []configuration.Field {
	repoMode := []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{sourceModeRepository}}}
	prMode := []configuration.VisibilityCondition{{Field: "sourceMode", Values: []string{sourceModePR}}}
	limitedNet := []configuration.VisibilityCondition{{Field: "networking", Values: []string{networkingLimited}}}

	return []configuration.Field{
		{
			Name: "sourceMode", Label: "Source", Type: configuration.FieldTypeSelect, Required: true, Default: sourceModeRepository,
			Description: "Start new work on a repository, or update an existing pull request.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Repository (new work)", Value: sourceModeRepository},
				{Label: "Pull request (update existing)", Value: sourceModePR},
			}}},
		},
		{
			Name: "repository", Label: "Repository", Type: configuration.FieldTypeString, Required: false,
			Placeholder: "owner/repo", Description: "Target repository (owner/repo or clone URL).",
			VisibilityConditions: repoMode,
		},
		{
			Name: "baseBranch", Label: "Base Branch", Type: configuration.FieldTypeString, Required: false,
			Description:          "Branch to start from and target the PR against. Defaults to the repository's default branch.",
			VisibilityConditions: repoMode,
		},
		{
			Name: "branchName", Label: "Working Branch", Type: configuration.FieldTypeString, Required: false,
			Description:          "Branch the agent creates and pushes. Defaults to claude/agent-<execution-id>.",
			VisibilityConditions: repoMode,
		},
		{
			Name: "prUrl", Label: "Pull Request URL", Type: configuration.FieldTypeString, Required: false,
			Placeholder: "https://github.com/owner/repo/pull/42", Description: "Existing pull request to update in place.",
			VisibilityConditions: prMode,
		},
		{
			Name: "task", Label: "Task", Type: configuration.FieldTypeText, Required: true,
			Description: "What the agent should do.",
		},
		{
			Name: "githubToken", Label: "GitHub Token", Type: configuration.FieldTypeSecretKey, Required: true,
			Description: "SuperPlane secret holding a GitHub token with repo + pull request permissions.",
		},
		{
			Name: "model", Label: "Model", Type: configuration.FieldTypeIntegrationResource, Required: false,
			Description: "Claude model that powers the agent.",
			TypeOptions: &configuration.TypeOptions{Resource: &configuration.ResourceTypeOptions{Type: "model"}},
		},
		{
			Name: "files", Label: "Files", Type: configuration.FieldTypeList, Required: false,
			Description: "Files from the Files tab to mount into the agent's workspace.",
			TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{
				ItemLabel:      "File path",
				ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeRepositoryFile},
			}},
		},
		{
			Name: "networking", Label: "Networking", Type: configuration.FieldTypeSelect, Required: false, Default: networkingUnrestricted,
			Description: "Sandbox outbound access.",
			TypeOptions: &configuration.TypeOptions{Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
				{Label: "Unrestricted", Value: networkingUnrestricted},
				{Label: "Limited (GitHub + registries)", Value: networkingLimited},
			}}},
		},
		{
			Name: "allowedHosts", Label: "Allowed Hosts", Type: configuration.FieldTypeList, Required: false,
			Description:          "Extra hosts permitted when networking is limited.",
			VisibilityConditions: limitedNet,
			TypeOptions: &configuration.TypeOptions{List: &configuration.ListTypeOptions{
				ItemLabel:      "Host",
				ItemDefinition: &configuration.ListItemDefinition{Type: configuration.FieldTypeString},
			}},
		},
		{
			Name: "autoCreatePr", Label: "Open a Pull Request", Type: configuration.FieldTypeBool, Required: false, Default: true,
			Description:          "Have the agent open a pull request when finished.",
			VisibilityConditions: repoMode,
		},
		{
			Name: "actAsBot", Label: "Act as Bot", Type: configuration.FieldTypeBool, Required: false, Default: true,
			Description: "When enabled, commits are attributed to the Claude agent. Disable to attribute commits to the GitHub user that owns the token.",
		},
	}
}

func (a *RunCodeAgent) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}
	if err := validateConfiguredFiles(ctx, spec.Files); err != nil {
		return err
	}
	return setNodeMetadata(ctx, spec)
}

func (a *RunCodeAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *RunCodeAgent) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	token, err := ctx.Secrets.GetKey(spec.GithubToken.Secret, spec.GithubToken.Key)
	if err != nil {
		return fmt.Errorf("failed to resolve GitHub token: %w", err)
	}

	var pr *pullRequestInfo
	if spec.SourceMode == sourceModePR {
		pr, err = resolvePullRequestForRun(ctx, spec, string(token))
		if err != nil {
			return err
		}
	}

	attribution, err := resolveCommitAttribution(ctx, spec, string(token))
	if err != nil {
		return err
	}

	meta := &ExecutionMetadata{
		Repository: repositoryForRun(spec, pr),
		Branch:     branchForRun(spec, pr, ctx.ID),
	}
	if pr != nil {
		meta.PrURL = pr.HTMLURL
	}

	resources, err := a.provisionResources(ctx, client, spec, string(token), meta)
	if err != nil {
		return err
	}

	if err := a.startSession(ctx, client, spec, pr, attribution, meta, resources); err != nil {
		return err
	}

	refreshed, err := client.GetManagedSession(meta.Session.ID)
	if err != nil {
		a.teardown(client, meta, true, ctx.Logger.Warnf)
		return fmt.Errorf("failed to get session: %w", err)
	}

	handled, err := a.emitIfTerminal(ctx, client, meta, refreshed)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	ctx.Logger.Infof("Started code agent session %s. Waiting for completion...", meta.Session.ID)
	if err := ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, initialPoll); err != nil {
		// Without a scheduled poll nothing will finish or reclaim the run, so
		// tear everything down rather than leaking the running session.
		a.teardown(client, meta, true, ctx.Logger.Warnf)
		return fmt.Errorf("failed to schedule poll: %w", err)
	}
	return nil
}

// startSession creates the session, persists metadata, and sends the task prompt.
// On any failure it reclaims all provisioned resources.
func (a *RunCodeAgent) startSession(ctx core.ExecutionContext, client *runagent.Client, spec Spec, pr *pullRequestInfo, attribution commitAttribution, meta *ExecutionMetadata, resources []runagent.FileResource) error {
	session, err := client.CreateManagedSession(runagent.CreateManagedSessionRequest{
		Agent:         meta.AgentID,
		EnvironmentID: meta.EnvironmentID,
		VaultIDs:      []string{meta.VaultID},
		Resources:     resources,
	})
	if err != nil {
		a.teardown(client, meta, false, ctx.Logger.Warnf)
		return fmt.Errorf("failed to create managed agent session: %w", err)
	}
	mergeSessionIntoMetadata(meta, session)
	if err := ctx.Metadata.Set(*meta); err != nil {
		a.teardown(client, meta, false, ctx.Logger.Warnf)
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	message := buildPrompt(spec, pr, meta.Branch, len(spec.Files) > 0, attribution)
	if err := client.SendManagedSessionUserMessage(session.ID, message); err != nil {
		a.teardown(client, meta, true, ctx.Logger.Warnf)
		return fmt.Errorf("failed to send task to agent: %w", err)
	}
	return nil
}

func (a *RunCodeAgent) Cleanup(ctx core.SetupContext) error { return nil }

// provisionResources creates the agent, environment, uploaded files, and vault,
// recording each in metadata so cleanup can always reclaim them. On any failure
// it tears down what was created and returns the error.
func (a *RunCodeAgent) provisionResources(ctx core.ExecutionContext, client *runagent.Client, spec Spec, token string, meta *ExecutionMetadata) ([]runagent.FileResource, error) {
	agentID, err := client.CreateAgent(runagent.CreateAgentRequest{
		Name:   fmt.Sprintf("superplane-%s", shortID(ctx.ID.String())),
		Model:  modelForRun(spec),
		System: defaultSystemPrompt,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create agent: %w", err)
	}
	meta.AgentID = agentID

	envID, err := client.CreateEnvironment(fmt.Sprintf("superplane-%s", shortID(ctx.ID.String())), buildEnvironmentConfig(spec))
	if err != nil {
		a.teardown(client, meta, false, ctx.Logger.Warnf)
		return nil, fmt.Errorf("failed to create environment: %w", err)
	}
	meta.EnvironmentID = envID

	resources, err := uploadFiles(client, ctx, spec.Files)
	if err != nil {
		meta.FileIDs = fileIDsOf(resources)
		a.teardown(client, meta, false, ctx.Logger.Warnf)
		return nil, err
	}
	meta.FileIDs = fileIDsOf(resources)

	vaultID, err := provisionVault(client, ctx, spec, token)
	if err != nil {
		meta.VaultID = vaultID // may be set even on partial failure
		a.teardown(client, meta, false, ctx.Logger.Warnf)
		return nil, err
	}
	meta.VaultID = vaultID

	return resources, nil
}

// teardown best-effort reclaims every provisioned resource. Safe to call with a
// partially-populated metadata (the client delete calls no-op on empty IDs).
func (a *RunCodeAgent) teardown(client *runagent.Client, meta *ExecutionMetadata, interrupt bool, logWarn func(string, ...any)) {
	if meta.Session != nil && meta.Session.ID != "" {
		if interrupt {
			warnErr(logWarn, client.SendManagedSessionInterrupt(meta.Session.ID), "interrupt session %s", meta.Session.ID)
		}
		warnErr(logWarn, client.DeleteManagedSession(meta.Session.ID), "delete session %s", meta.Session.ID)
	}
	warnErr(logWarn, client.DeleteEnvironment(meta.EnvironmentID), "delete environment %s", meta.EnvironmentID)
	client.CleanupFiles(meta.FileIDs, logWarn)
	warnErr(logWarn, client.DeleteVault(meta.VaultID), "delete vault %s", meta.VaultID)
	warnErr(logWarn, client.ArchiveAgent(meta.AgentID), "archive agent %s", meta.AgentID)
}

// warnErr logs a best-effort cleanup failure without interrupting teardown.
func warnErr(logWarn func(string, ...any), err error, format string, args ...any) {
	if err != nil {
		logWarn("failed to "+format+": %v", append(args, err)...)
	}
}

// emitIfTerminal handles fast tasks that finish before the first poll.
func (a *RunCodeAgent) emitIfTerminal(ctx core.ExecutionContext, client *runagent.Client, meta *ExecutionMetadata, session *runagent.ManagedSession) (bool, error) {
	if session == nil || !isSessionTerminal(session.Status) {
		return false, nil
	}

	sm, err := client.GetSessionMessagesWithRetry(meta.Session.ID, finalMessageReads, finalMessageDelay)
	if err != nil || sm == nil || !sm.Complete {
		ctx.Logger.Warnf("Session %s terminal but events not ready; scheduling poll.", meta.Session.ID)
		return false, nil
	}

	out := buildOutput(session.Status, meta.Session.ID, meta.Branch, sm, meta.PrURL)
	out.Artifacts = runagent.CollectSessionArtifacts(client, meta.Session.ID, sm.ExpectsArtifacts, ctx.Logger.Warnf)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		ctx.Logger.Warnf("Failed to emit result for session %s: %v; scheduling poll.", meta.Session.ID, err)
		return false, nil
	}
	mergeSessionIntoMetadata(meta, session)
	_ = ctx.Metadata.Set(*meta)
	a.teardown(client, meta, false, ctx.Logger.Warnf)
	return true, nil
}

func uploadFiles(client *runagent.Client, ctx core.ExecutionContext, files []string) ([]runagent.FileResource, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if ctx.Files == nil {
		return nil, fmt.Errorf("files configured but file access is not available")
	}
	resources := make([]runagent.FileResource, 0, len(files))
	for _, path := range files {
		normalized, err := gitprovider.ValidateUserPath(path)
		if err != nil {
			return resources, fmt.Errorf("invalid file path %q: %w", path, err)
		}
		reader, err := ctx.Files.Read(normalized)
		if err != nil {
			return resources, fmt.Errorf("read file %q: %w", path, err)
		}
		fileID, err := client.UploadFile(reader, normalized)
		reader.Close()
		if err != nil {
			return resources, fmt.Errorf("upload file %q: %w", path, err)
		}
		resources = append(resources, runagent.FileResource{FileID: fileID, MountPath: attachmentsMountDir + "/" + normalized})
	}
	return resources, nil
}

func provisionVault(client *runagent.Client, ctx core.ExecutionContext, spec Spec, token string) (string, error) {
	vaultID, err := client.CreateVault(fmt.Sprintf("superplane-%s", shortID(ctx.ID.String())), map[string]string{"superplane_execution": ctx.ID.String()})
	if err != nil {
		return "", fmt.Errorf("failed to create vault: %w", err)
	}

	// Restrict the token to GitHub hosts only (never the user's extra allowedHosts)
	// so it can never be sent to another host at the egress layer.
	if err := client.CreateEnvVarCredential(vaultID, "GITHUB_TOKEN", "GITHUB_TOKEN", token, defaultGitHubHosts); err != nil {
		return vaultID, fmt.Errorf("failed to inject GitHub token: %w", err)
	}
	return vaultID, nil
}

func buildEnvironmentConfig(spec Spec) runagent.EnvironmentConfig {
	cfg := runagent.EnvironmentConfig{Type: "cloud"}
	if spec.Networking == networkingLimited {
		allow := true
		hosts := append([]string{}, defaultGitHubHosts...)
		hosts = append(hosts, spec.AllowedHosts...)
		cfg.Networking = runagent.EnvironmentNetworking{
			Type:                 "limited",
			AllowedHosts:         hosts,
			AllowPackageManagers: &allow,
		}
	} else {
		cfg.Networking = runagent.EnvironmentNetworking{Type: "unrestricted"}
	}
	return cfg
}

// resolvePullRequestForRun looks up the PR and validates it is workable.
func resolvePullRequestForRun(ctx core.ExecutionContext, spec Spec, token string) (*pullRequestInfo, error) {
	pr, err := resolvePullRequest(ctx.HTTP, spec.PrURL, token)
	if err != nil {
		return nil, err
	}
	if !strings.EqualFold(pr.State, "open") {
		return nil, fmt.Errorf("pull request %s is %s; only open pull requests can be updated", spec.PrURL, pr.State)
	}
	if pr.isFork() {
		return nil, fmt.Errorf("pull request %s was opened from a fork (%s); updating fork PRs is not yet supported", spec.PrURL, pr.HeadRepo)
	}
	if strings.TrimSpace(pr.BaseRepo) == "" || strings.TrimSpace(pr.HeadRef) == "" {
		return nil, fmt.Errorf("pull request %s is missing its base repository or head branch", spec.PrURL)
	}
	return pr, nil
}

func decodeSpec(config any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if raw, ok := config.(map[string]any); ok {
		if v, ok := raw["files"]; ok {
			spec.Files = decodeStringList(v)
		}
		if v, ok := raw["allowedHosts"]; ok {
			spec.AllowedHosts = decodeStringList(v)
		}
	}
	if strings.TrimSpace(spec.SourceMode) == "" {
		spec.SourceMode = sourceModeRepository
	}
	return spec, nil
}

func decodeStringList(v any) []string {
	switch x := v.(type) {
	case []string:
		return x
	case []any:
		out := make([]string, 0, len(x))
		for _, e := range x {
			if s, ok := e.(string); ok {
				out = append(out, s)
			}
		}
		return out
	default:
		return nil
	}
}

func validateSpec(spec Spec) error {
	if strings.TrimSpace(spec.Task) == "" {
		return fmt.Errorf("task is required")
	}
	if !spec.GithubToken.isSet() {
		return fmt.Errorf("githubToken is required")
	}
	switch spec.SourceMode {
	case sourceModeRepository:
		if strings.TrimSpace(spec.Repository) == "" {
			return fmt.Errorf("repository is required in repository mode")
		}
		if err := validateRepository(spec.Repository); err != nil {
			return err
		}
		if err := validateGitRef("baseBranch", spec.BaseBranch); err != nil {
			return err
		}
		if err := validateGitRef("branchName", spec.BranchName); err != nil {
			return err
		}
	case sourceModePR:
		if strings.TrimSpace(spec.PrURL) == "" {
			return fmt.Errorf("prUrl is required in pull request mode")
		}
	default:
		return fmt.Errorf("invalid sourceMode %q", spec.SourceMode)
	}
	return nil
}

func validateConfiguredFiles(ctx core.SetupContext, files []string) error {
	if len(files) == 0 {
		return nil
	}
	if ctx.Files == nil {
		return fmt.Errorf("files configured but file access is not available")
	}
	available, err := ctx.Files.List()
	if err != nil {
		return fmt.Errorf("failed to list repository files: %v", err)
	}
	fileSet := make(map[string]bool, len(available))
	for _, f := range available {
		if norm, err := gitprovider.NormalizePath(f); err == nil {
			fileSet[norm] = true
		}
	}
	for _, f := range files {
		norm, err := gitprovider.ValidateUserPath(f)
		if err != nil {
			return fmt.Errorf("invalid file path %q: %v", f, err)
		}
		if !fileSet[norm] {
			return fmt.Errorf("file %q not found in app repository", f)
		}
	}
	return nil
}

var gitRefPattern = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._/-]*$`)

func validateGitRef(field, ref string) error {
	ref = strings.TrimSpace(ref)
	if ref == "" || strings.Contains(ref, "{{") {
		return nil
	}
	if !gitRefPattern.MatchString(ref) {
		return fmt.Errorf("%s %q contains invalid characters", field, ref)
	}
	return nil
}

var repoOwnerRepoPattern = regexp.MustCompile(`^[A-Za-z0-9._-]+/[A-Za-z0-9._-]+$`)

func validateRepository(repository string) error {
	repository = strings.TrimSpace(repository)
	if strings.Contains(repository, "{{") {
		return nil
	}
	if i := strings.IndexFunc(repository, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsControl(r)
	}); i >= 0 {
		return fmt.Errorf("repository must not contain whitespace or control characters")
	}
	if repoOwnerRepoPattern.MatchString(repository) {
		return nil
	}
	// Only owner/repo and https://github.com URLs are accepted. The GitHub token
	// is embedded in the clone URL, so pointing at any other host would leak the
	// credential to a non-GitHub server.
	if strings.HasPrefix(repository, "https://github.com/") {
		return nil
	}
	return fmt.Errorf("repository must be owner/repo or an https://github.com/ URL")
}

func setNodeMetadata(ctx core.SetupContext, spec Spec) error {
	if ctx.Metadata == nil {
		return nil
	}
	meta := NodeMetadata{
		SourceMode: spec.SourceMode,
		Model:      modelForRun(spec),
	}
	if spec.SourceMode == sourceModePR {
		meta.PrURL = strings.TrimSpace(spec.PrURL)
	} else {
		meta.Repository = strings.TrimSpace(spec.Repository)
		meta.BaseBranch = strings.TrimSpace(spec.BaseBranch)
	}
	return ctx.Metadata.Set(meta)
}

func modelForRun(spec Spec) string {
	if m := strings.TrimSpace(spec.Model); m != "" {
		return m
	}
	return defaultModel
}

// commitAttribution is the git author identity the agent should commit as. When
// disabled (empty), the agent keeps its default identity and attribution trailers.
type commitAttribution struct {
	AuthorName  string
	AuthorEmail string
}

func (c commitAttribution) enabled() bool { return c.AuthorName != "" }

func actAsBot(spec Spec) bool {
	return spec.ActAsBot == nil || *spec.ActAsBot
}

// resolveCommitAttribution returns the author identity to attribute commits to
// when "Act as Bot" is off, so commits appear as the token's GitHub user rather
// than as the agent.
func resolveCommitAttribution(ctx core.ExecutionContext, spec Spec, token string) (commitAttribution, error) {
	if actAsBot(spec) {
		return commitAttribution{}, nil
	}
	name, email, err := resolveGitHubUser(ctx.HTTP, token)
	if err != nil {
		return commitAttribution{}, fmt.Errorf("failed to resolve GitHub user for commit attribution: %w", err)
	}
	return commitAttribution{AuthorName: name, AuthorEmail: email}, nil
}

func repositoryForRun(spec Spec, pr *pullRequestInfo) string {
	if pr != nil {
		return pr.BaseRepo
	}
	return strings.TrimSpace(spec.Repository)
}

func branchForRun(spec Spec, pr *pullRequestInfo, id uuid.UUID) string {
	if pr != nil {
		return pr.HeadRef
	}
	if b := strings.TrimSpace(spec.BranchName); b != "" {
		return b
	}
	return "claude/agent-" + shortID(id.String())
}

func fileIDsOf(resources []runagent.FileResource) []string {
	if len(resources) == 0 {
		return nil
	}
	ids := make([]string, len(resources))
	for i, r := range resources {
		ids[i] = r.FileID
	}
	return ids
}

func shortID(id string) string {
	id = strings.ReplaceAll(id, "-", "")
	if len(id) > 12 {
		return id[:12]
	}
	return id
}
