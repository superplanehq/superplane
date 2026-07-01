package runcloudagent

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/integrations/claude/runagent"
)

type RunCloudAgent struct{}

func (a *RunCloudAgent) Name() string { return "claude.runCloudAgent" }

func (a *RunCloudAgent) Label() string { return "Run Claude Cloud Agent" }

func (a *RunCloudAgent) Description() string {
	return "Runs a Claude Managed Agent in Anthropic’s managed cloud environment with an optional repository checked out, and waits until the session is idle or terminated."
}

func (a *RunCloudAgent) Documentation() string {
	return `The **Run Claude Cloud Agent** component uses [Claude Managed Agents](https://platform.claude.com/docs/en/managed-agents/overview) to start a **session** with a selected agent and environment, optionally clones a **repository** into the agent's workspace, sends your task as a **user message**, and waits until the **session** reaches a terminal state (idle or terminated) by polling.

It builds on **Run Claude Agent** by letting you pick the agent and environment from the ones that already exist on your Claude account, and by checking out a git repository before the task runs.

## Prerequisites

- A **Claude API key** on the integration.
- An **agent** and **environment** already created in the Anthropic API (or Console). This step lists and references them.

## Configuration

- **Agent**: Pick a Managed Agent. Uses the latest version unless **Version** is set.
- **Version** (optional): Pins the session to a specific agent version.
- **Environment**: Pick the environment (sandbox) the session runs in.
- **Repository** (optional): A git repository to clone into the agent's working directory before the task runs.
- **Branch** (optional): The branch to check out for the repository.
- **Prompt**: The user message (task) sent to the agent.
- **Vault IDs** (optional): For MCP tools that need vault-backed credentials.
- **Files** (optional): Repository files to mount into the agent's working directory.
- **Secrets** (optional): SuperPlane secrets injected as environment variables.

## Output

Emits a finished payload with **session** status, **session id**, and the final **agent message** when available so downstream steps can branch or consume the result. For failure cases the status is still emitted when the **session** is *terminated* or the step times out.`
}

func (a *RunCloudAgent) Icon() string { return "bot" }

func (a *RunCloudAgent) Color() string { return "#C9784D" }

func (a *RunCloudAgent) ExampleOutput() map[string]any {
	return getExampleOutput()
}

func (a *RunCloudAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{{Name: defaultChannel, Label: "Default"}}
}

func (a *RunCloudAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agent",
			Label:       "Agent",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Claude Managed Agent to run. Uses the latest version unless **Version** is set.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "agent"},
			},
		},
		{
			Name:        "version",
			Label:       "Agent version",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "When set, pins the session to this agent version (otherwise latest is used).",
		},
		{
			Name:        "environmentId",
			Label:       "Environment",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Managed Agent environment (sandbox) for this session.",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "environment"},
			},
		},
		{
			Name:        "repository",
			Label:       "Repository",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "https://github.com/owner/repo.git",
			Description: "Optional git repository to clone into the agent's working directory before running the task (HTTPS or SSH clone URL).",
		},
		{
			Name:        "branch",
			Label:       "Branch",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "Branch to check out for the repository. Defaults to the repository's default branch.",
		},
		{
			Name:        "prompt",
			Label:       "Task",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "User message (task) for the agent",
		},
		{
			Name:     "vaultIds",
			Label:    "Vault IDs",
			Type:     configuration.FieldTypeList,
			Required: false,
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Vault ID",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeString,
					},
				},
			},
			Description: "Optional vault IDs for MCP authentication (see Managed Agents docs)",
		},
		{
			Name:        "files",
			Label:       "Files",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "File paths from the Files tab to mount into the agent's working directory",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File path",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeRepositoryFile,
					},
				},
			},
		},
		{
			Name:        "secrets",
			Label:       "Secrets",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "SuperPlane secrets to inject as environment variables in the agent session. Secrets are injected at the network egress layer and never exposed to the agent.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "Secret",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeObject,
						Schema: []configuration.Field{
							{
								Name:        "envName",
								Label:       "Environment Variable",
								Type:        configuration.FieldTypeString,
								Required:    true,
								Placeholder: "GITHUB_TOKEN",
								Description: "Name of the environment variable in the agent session",
							},
							{
								Name:        "value",
								Label:       "Secret",
								Type:        configuration.FieldTypeSecretKey,
								Required:    true,
								Description: "SuperPlane secret and key to inject",
							},
							{
								Name:        "allowedHosts",
								Label:       "Allowed Hosts",
								Type:        configuration.FieldTypeList,
								Required:    false,
								Description: "Restrict which domains this secret can be sent to. Leave empty for unrestricted.",
								TypeOptions: &configuration.TypeOptions{
									List: &configuration.ListTypeOptions{
										ItemLabel: "Host",
										ItemDefinition: &configuration.ListItemDefinition{
											Type: configuration.FieldTypeString,
										},
									},
								},
							},
						},
					},
				},
			},
		},
	}
}

func (a *RunCloudAgent) Setup(ctx core.SetupContext) error {
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
	if err := validateSecretBindings(spec.Secrets); err != nil {
		return err
	}
	return resolveNodeMetadata(ctx, spec)
}

// resolveNodeMetadata resolves the agent and environment names and stores them
// in the node metadata so the component card can display them without waiting
// for an execution. Name resolution is best-effort: on any failure the IDs are
// stored as the display names so configuring the node never breaks.
func resolveNodeMetadata(ctx core.SetupContext, spec Spec) error {
	if ctx.Metadata == nil {
		return nil
	}

	agentID := strings.TrimSpace(spec.Agent)
	environmentID := strings.TrimSpace(spec.EnvironmentID)

	// Reuse already-resolved names when the IDs are unchanged.
	var existing NodeMetadata
	if mapstructure.Decode(ctx.Metadata.Get(), &existing) == nil &&
		existing.AgentID == agentID && existing.EnvironmentID == environmentID &&
		existing.AgentName != "" && existing.EnvironmentName != "" {
		return nil
	}

	meta := NodeMetadata{
		AgentID:         agentID,
		AgentName:       agentID,
		EnvironmentID:   environmentID,
		EnvironmentName: environmentID,
	}

	client, err := runagent.NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(meta)
	}
	if name := resolveAgentName(client, agentID); name != "" {
		meta.AgentName = name
	}
	if name := resolveEnvironmentName(client, environmentID); name != "" {
		meta.EnvironmentName = name
	}
	return ctx.Metadata.Set(meta)
}

// resolveAgentName returns the agent's display name, or "" when it can't be resolved.
func resolveAgentName(client *runagent.Client, agentID string) string {
	if agentID == "" || strings.Contains(agentID, "{{") {
		return agentID
	}
	if a, err := client.GetAgent(agentID); err == nil && a != nil && strings.TrimSpace(a.Name) != "" {
		return a.Name
	}
	return ""
}

// resolveEnvironmentName returns the environment's display name, or "" when it can't be resolved.
func resolveEnvironmentName(client *runagent.Client, environmentID string) string {
	if environmentID == "" || strings.Contains(environmentID, "{{") {
		return environmentID
	}
	if e, err := client.GetEnvironment(environmentID); err == nil && e != nil && strings.TrimSpace(e.Name) != "" {
		return e.Name
	}
	return ""
}

// validateConfiguredFiles ensures every configured file path exists in the app repository.
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

// validateSecretBindings ensures each secret binding has an env name and a secret reference.
func validateSecretBindings(secrets []SecretBinding) error {
	for i, s := range secrets {
		if strings.TrimSpace(s.EnvName) == "" {
			return fmt.Errorf("secrets[%d].envName is required", i)
		}
		if s.Value.Secret == "" || s.Value.Key == "" {
			return fmt.Errorf("secrets[%d].value.secret and secrets[%d].value.key are required", i, i)
		}
	}
	return nil
}

func (a *RunCloudAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *RunCloudAgent) Execute(ctx core.ExecutionContext) error {
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

	resources, err := prepareFileResources(client, ctx, spec.Files)
	if err != nil {
		return err
	}

	vaultIDs, err := collectVaultIDs(client, ctx, spec)
	if err != nil {
		return err
	}

	aid := strings.TrimSpace(spec.Agent)
	environmentID := strings.TrimSpace(spec.EnvironmentID)
	createReq := runagent.CreateManagedSessionRequest{
		Agent:         aid,
		AgentVersion:  spec.Version,
		EnvironmentID: environmentID,
		VaultIDs:      vaultIDs,
		Resources:     resources,
	}

	session, err := client.CreateManagedSession(createReq)
	if err != nil {
		cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to create managed agent session: %w", err)
	}

	metadata := ExecutionMetadata{
		Repository: strings.TrimSpace(spec.Repository),
		Branch:     strings.TrimSpace(spec.Branch),
	}
	mergeSessionIntoMetadata(&metadata, session)
	if err := ctx.Metadata.Set(metadata); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV("managed_session_id", session.ID); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to set managed_session_id: %w", err)
	}

	// When a repository is configured, the agent has no first-class repository
	// field — instead, prepend a clone instruction so the agent checks it out.
	message := buildRepositoryPrompt(spec.Repository, spec.Branch, spec.Prompt)
	if err := client.SendManagedSessionUserMessage(session.ID, message); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to send user message: %w", err)
	}

	// Check if session already finished (fast tasks).
	refreshed, err := client.GetManagedSession(session.ID)
	if err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to get session: %w", err)
	}

	handled, err := a.emitIfAlreadyTerminal(client, ctx, &metadata, session.ID, refreshed)
	if err != nil {
		return err
	}
	if handled {
		return nil
	}

	ctx.Logger.Infof("Started Managed Agent session %s. Waiting for completion (polling)...", session.ID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, initialPoll)
}

// prepareFileResources uploads any configured repository files and records the
// resulting file IDs so they can be cleaned up on a later error.
func prepareFileResources(client *runagent.Client, ctx core.ExecutionContext, files []string) ([]runagent.FileResource, error) {
	if len(files) == 0 {
		return nil, nil
	}
	if ctx.Files == nil {
		return nil, fmt.Errorf("files configured but file access is not available in this execution context")
	}

	resources, err := uploadRepositoryFiles(client, ctx, files)
	if err != nil {
		return nil, fmt.Errorf("failed to upload files: %w", err)
	}

	fileIDs := make([]string, len(resources))
	for i, r := range resources {
		fileIDs[i] = r.FileID
		ctx.Logger.Infof("Mounting file: %s (file_id: %s)", r.MountPath, r.FileID)
	}
	encoded, _ := json.Marshal(fileIDs)
	if err := ctx.ExecutionState.SetKV("uploaded_file_ids", string(encoded)); err != nil {
		cleanupFileResources(client, resources, ctx.Logger.Warnf)
		return nil, fmt.Errorf("failed to persist uploaded file IDs: %w", err)
	}
	return resources, nil
}

// collectVaultIDs returns the configured vault IDs plus a temporary vault
// holding any secrets injected as environment variables.
func collectVaultIDs(client *runagent.Client, ctx core.ExecutionContext, spec Spec) ([]string, error) {
	vaultIDs := append([]string{}, spec.VaultIDs...)
	if len(spec.Secrets) == 0 {
		return vaultIDs, nil
	}

	vaultID, err := provisionSecretsVault(client, ctx, spec.Secrets)
	if err != nil {
		cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return nil, fmt.Errorf("failed to provision secrets vault: %w", err)
	}
	return append(vaultIDs, vaultID), nil
}

// emitIfAlreadyTerminal handles fast tasks that finish before the first poll.
// It returns (true, nil) when the run has been emitted and cleaned up.
func (a *RunCloudAgent) emitIfAlreadyTerminal(client *runagent.Client, ctx core.ExecutionContext, metadata *ExecutionMetadata, sessionID string, session *runagent.ManagedSession) (bool, error) {
	if session == nil || !isSessionTerminal(session.Status) {
		return false, nil
	}

	sm, err := client.GetSessionMessagesWithRetry(sessionID, finalMessageReads, finalMessageDelay)
	if err != nil {
		ctx.Logger.Warnf("Failed to fetch messages for managed session %s: %v. Scheduling poll.", sessionID, err)
		return false, nil
	}
	if sm == nil || !sm.Complete {
		ctx.Logger.Warnf("Events not complete for session %s after retries. Scheduling poll.", sessionID)
		return false, nil
	}

	out := buildOutputFromSessionMessages(session.Status, sessionID, sm)
	if err := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return false, err
	}

	// Persist terminal status only after successful emit
	mergeSessionIntoMetadata(metadata, session)
	_ = ctx.Metadata.Set(*metadata)
	if err := client.DeleteManagedSession(sessionID); err != nil {
		ctx.Logger.Warnf("Failed to delete managed session %s: %v", sessionID, err)
	}
	cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
	cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
	return true, nil
}

func (a *RunCloudAgent) Cleanup(ctx core.SetupContext) error { return nil }

// getUploadedFileIDs retrieves uploaded file IDs from execution state.
func getUploadedFileIDs(state core.ExecutionStateContext) []string {
	raw, err := state.GetKV("uploaded_file_ids")
	if err != nil || raw == "" {
		return nil
	}
	var fileIDs []string
	if err := json.Unmarshal([]byte(raw), &fileIDs); err != nil {
		return nil
	}
	return fileIDs
}

func cleanupUploadedFiles(client *runagent.Client, ctx core.ExecutionContext, logWarn func(string, ...any)) {
	client.CleanupFiles(getUploadedFileIDs(ctx.ExecutionState), logWarn)
}

func cleanupUploadedFilesFromHook(client *runagent.Client, ctx core.ActionHookContext, logWarn func(string, ...any)) {
	client.CleanupFiles(getUploadedFileIDs(ctx.ExecutionState), logWarn)
}

func uploadRepositoryFiles(client *runagent.Client, ctx core.ExecutionContext, files []string) ([]runagent.FileResource, error) {
	resources := make([]runagent.FileResource, 0, len(files))
	for _, path := range files {
		normalized, err := gitprovider.ValidateUserPath(path)
		if err != nil {
			cleanupFileResources(client, resources, ctx.Logger.Warnf)
			return nil, fmt.Errorf("invalid file path %q: %w", path, err)
		}

		reader, err := ctx.Files.Read(normalized)
		if err != nil {
			cleanupFileResources(client, resources, ctx.Logger.Warnf)
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}

		fileID, err := client.UploadFile(reader, normalized)
		reader.Close()
		if err != nil {
			cleanupFileResources(client, resources, ctx.Logger.Warnf)
			return nil, fmt.Errorf("upload file %q: %w", path, err)
		}

		resources = append(resources, runagent.FileResource{
			FileID:    fileID,
			MountPath: normalized,
		})
	}
	return resources, nil
}

// cleanupFileResources deletes already-uploaded files on partial failure.
func cleanupFileResources(client *runagent.Client, resources []runagent.FileResource, logWarn func(string, ...any)) {
	for _, r := range resources {
		if err := client.DeleteFile(r.FileID); err != nil && logWarn != nil {
			logWarn("Failed to delete uploaded file %s: %v", r.FileID, err)
		}
	}
}

// provisionSecretsVault creates a temporary Anthropic vault and adds
// environment_variable credentials for each configured secret binding.
// Returns the vault ID for session creation.
func provisionSecretsVault(client *runagent.Client, ctx core.ExecutionContext, secrets []SecretBinding) (string, error) {
	vaultID, err := client.CreateVault(
		fmt.Sprintf("superplane-%s", truncateID(ctx.ID.String(), 12)),
		map[string]string{"superplane_execution": ctx.ID.String()},
	)
	if err != nil {
		return "", fmt.Errorf("create vault: %w", err)
	}

	// Store vault ID immediately so cleanup works on later errors.
	if err := ctx.ExecutionState.SetKV("managed_vault_id", vaultID); err != nil {
		_ = client.DeleteVault(vaultID)
		return "", fmt.Errorf("persist vault ID: %w", err)
	}

	for _, s := range secrets {
		value, err := ctx.Secrets.GetKey(s.Value.Secret, s.Value.Key)
		if err != nil {
			_ = client.DeleteVault(vaultID)
			return "", fmt.Errorf("resolve secret %s/%s: %w", s.Value.Secret, s.Value.Key, err)
		}

		if err := client.CreateEnvVarCredential(vaultID, s.EnvName, s.EnvName, string(value), s.AllowedHosts); err != nil {
			_ = client.DeleteVault(vaultID)
			return "", fmt.Errorf("create credential for %s: %w", s.EnvName, err)
		}
		ctx.Logger.Infof("Injected secret as env var: %s", s.EnvName)
	}

	return vaultID, nil
}

// cleanupManagedVault deletes the temporary vault created for this execution.
func cleanupManagedVault(client *runagent.Client, ctx core.ExecutionContext, logWarn func(string, ...any)) {
	vaultID, err := ctx.ExecutionState.GetKV("managed_vault_id")
	if err != nil || vaultID == "" {
		return
	}
	if err := client.DeleteVault(vaultID); err != nil && logWarn != nil {
		logWarn("Failed to delete managed vault %s: %v", vaultID, err)
	}
}

// cleanupManagedVaultFromHook is the ActionHookContext variant.
func cleanupManagedVaultFromHook(client *runagent.Client, ctx core.ActionHookContext, logWarn func(string, ...any)) {
	vaultID, err := ctx.ExecutionState.GetKV("managed_vault_id")
	if err != nil || vaultID == "" {
		return
	}
	if err := client.DeleteVault(vaultID); err != nil && logWarn != nil {
		logWarn("Failed to delete managed vault %s: %v", vaultID, err)
	}
}

func truncateID(id string, maxLen int) string {
	if len(id) <= maxLen {
		return id
	}
	return id[:maxLen]
}

func decodeSpec(config any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if raw, ok := config.(map[string]any); ok {
		if v, ok := raw["vaultIds"]; ok {
			spec.VaultIDs = decodeStringList(v)
		}
		if v, ok := raw["files"]; ok {
			spec.Files = decodeStringList(v)
		}
	}
	return spec, nil
}

func decodeStringList(v any) []string {
	switch x := v.(type) {
	case nil:
		return nil
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
	if strings.TrimSpace(spec.Agent) == "" {
		return fmt.Errorf("agent is required")
	}
	if strings.TrimSpace(spec.EnvironmentID) == "" {
		return fmt.Errorf("environmentId is required")
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return fmt.Errorf("prompt is required")
	}
	return validateRepository(spec.Repository, spec.Branch)
}

// gitBranchPattern matches safe git ref characters (no whitespace or shell/prompt
// metacharacters), so the branch can be embedded in the clone instruction safely.
var gitBranchPattern = regexp.MustCompile(`^[A-Za-z0-9._][A-Za-z0-9._/-]*$`)

// validateRepository ensures a configured repository/branch are well-formed
// before they are embedded into the agent prompt. This rejects a branch set
// without a repository (which would be silently ignored) and constrains the
// values to safe characters so they cannot be used to inject prompt content.
// Expression placeholders ({{ ... }}) are resolved at runtime and skipped here.
func validateRepository(repository, branch string) error {
	repository = strings.TrimSpace(repository)
	branch = strings.TrimSpace(branch)

	if branch != "" && repository == "" {
		return fmt.Errorf("branch is set but repository is required")
	}

	if repository != "" && !containsExpression(repository) {
		if strings.ContainsAny(repository, " \t\r\n") {
			return fmt.Errorf("repository must not contain whitespace")
		}
		if !isGitRepositoryURL(repository) {
			return fmt.Errorf("repository must be a valid git URL (https://, http://, ssh://, git://, or user@host:path)")
		}
	}

	if branch != "" && !containsExpression(branch) && !gitBranchPattern.MatchString(branch) {
		return fmt.Errorf("branch %q contains invalid characters", branch)
	}

	return nil
}

func containsExpression(value string) bool {
	return strings.Contains(value, "{{")
}

// scpLikeGitURL matches the scp-style git remote form, e.g. git@github.com:owner/repo.git.
var scpLikeGitURL = regexp.MustCompile(`^[A-Za-z0-9._-]+@[A-Za-z0-9._-]+:.+$`)

func isGitRepositoryURL(repository string) bool {
	switch {
	case strings.HasPrefix(repository, "https://"),
		strings.HasPrefix(repository, "http://"),
		strings.HasPrefix(repository, "ssh://"),
		strings.HasPrefix(repository, "git://"):
		return true
	default:
		return scpLikeGitURL.MatchString(repository)
	}
}
