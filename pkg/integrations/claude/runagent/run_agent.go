package runagent

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	log "github.com/sirupsen/logrus"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

type RunAgent struct{}

func (a *RunAgent) Name() string { return "claude.runAgent" }

func (a *RunAgent) Label() string { return "Run Claude Agent" }

func (a *RunAgent) Description() string {
	return "Runs a Claude Managed Agent in Anthropic’s managed environment and waits until the session is idle or terminated."
}

func (a *RunAgent) Documentation() string {
	return `The **Run Claude Agent** component uses [Claude Managed Agents](https://platform.claude.com/docs/en/managed-agents/overview) to start a **session** with a configured agent and environment, sends your task as a **user message**, and waits until the **session** reaches a terminal state (idle or terminated) by polling. Log streaming is not used.

## Prerequisites

- A **Claude API key** on the integration.
- An **agent** and **environment** already created in the Anthropic API (or Console). This step references them by ID.

## Configuration

- **Agent ID** and optional **Version**: the Managed Agent to run (latest, or a pinned version if **Version** is set).
- **Environment ID**: The environment the session runs in.
- **Prompt**: The user message (task) sent to the agent.
- **Vault IDs** (optional): For MCP tools that need vault-backed credentials.
- **Keep Session After Run** (optional): By default the session is deleted once the run finishes. Enable this to keep it so you can read the full transcript in the Anthropic Console when debugging. It applies only to runs that finish — a cancelled run is always cleaned up. Kept sessions are never reclaimed automatically, so delete them yourself when you're done.

## Output

Emits a finished payload with **session** status, **session id**, and the final **agent message** when available so downstream steps can branch or consume the result. For failure cases the status is still emitted when the **session** is *terminated* or the step times out.`
}

func (a *RunAgent) Icon() string { return "bot" }

func (a *RunAgent) Color() string { return "#C9784D" }

func (a *RunAgent) ExampleOutput() map[string]any {
	return getExampleOutput()
}

func (a *RunAgent) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{{Name: defaultChannel, Label: "Default"}}
}

func (a *RunAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "agent",
			Label:       "Agent ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of a Claude Managed Agent. Uses the latest version unless **Version** is set.",
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
			Label:       "Environment ID",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Description: "ID of the Managed Agent environment (container) for this session",
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
			Name:        "persistSession",
			Label:       "Keep Session After Run",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Keep the Managed Agents session after the run finishes so its transcript stays readable in the Anthropic Console. Sessions are deleted by default.",
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

func (a *RunAgent) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	if len(spec.Files) > 0 {
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
		for _, f := range spec.Files {
			norm, err := gitprovider.ValidateUserPath(f)
			if err != nil {
				return fmt.Errorf("invalid file path %q: %v", f, err)
			}
			if !fileSet[norm] {
				return fmt.Errorf("file %q not found in app repository", f)
			}
		}
	}

	for i, s := range spec.Secrets {
		if strings.TrimSpace(s.EnvName) == "" {
			return fmt.Errorf("secrets[%d].envName is required", i)
		}
		if s.Value.Secret == "" || s.Value.Key == "" {
			return fmt.Errorf("secrets[%d].value.secret and secrets[%d].value.key are required", i, i)
		}
	}

	return nil
}

func (a *RunAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *RunAgent) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Upload files and prepare resources for session mounting
	var resources []FileResource
	if len(spec.Files) > 0 {
		if ctx.Files == nil {
			return fmt.Errorf("files configured but file access is not available in this execution context")
		}
		resources, err = uploadRepositoryFiles(client, ctx, spec.Files)
		if err != nil {
			return fmt.Errorf("failed to upload files: %w", err)
		}
	}

	// Store file IDs early so cleanup works on any later error path.
	if len(resources) > 0 {
		fileIDs := make([]string, len(resources))
		for i, r := range resources {
			fileIDs[i] = r.FileID
			ctx.Logger.Infof("Mounting file: %s (file_id: %s)", r.MountPath, r.FileID)
		}
		encoded, _ := json.Marshal(fileIDs)
		if err := ctx.ExecutionState.SetKV("uploaded_file_ids", string(encoded)); err != nil {
			cleanupFileResources(client, resources, ctx.Logger.Warnf)
			return fmt.Errorf("failed to persist uploaded file IDs: %w", err)
		}
	}

	// Create a temporary vault and inject secrets as environment variables.
	vaultIDs := append([]string{}, spec.VaultIDs...)
	if len(spec.Secrets) > 0 {
		vaultID, vaultErr := provisionSecretsVault(client, ctx, spec.Secrets)
		if vaultErr != nil {
			cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
			cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
			return fmt.Errorf("failed to provision secrets vault: %w", vaultErr)
		}
		vaultIDs = append(vaultIDs, vaultID)
	}

	aid := strings.TrimSpace(spec.Agent)
	createReq := CreateManagedSessionRequest{
		Agent:         aid,
		AgentVersion:  spec.Version,
		EnvironmentID: strings.TrimSpace(spec.EnvironmentID),
		VaultIDs:      vaultIDs,
		Resources:     resources,
	}

	session, err := client.CreateManagedSession(createReq)
	if err != nil {
		cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to create managed agent session: %w", err)
	}

	metadata := ExecutionMetadata{}
	mergeSessionIntoMetadata(&metadata, session)
	if err := ctx.Metadata.Set(metadata); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV("managed_session_id", session.ID); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to set managed_session_id: %w", err)
	}

	if err := client.SendManagedSessionUserMessage(session.ID, spec.Prompt); err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to send user message: %w", err)
	}

	// Check if session already finished (fast tasks).
	// Don't write terminal status to metadata yet — only after emit.
	refreshed, err := client.GetManagedSession(session.ID)
	if err != nil {
		cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
		return fmt.Errorf("failed to get session: %w", err)
	}

	if refreshed != nil && isSessionTerminal(refreshed.Status) {
		sm, err := client.GetSessionMessagesWithRetry(session.ID, finalMessageReads, finalMessageDelay)
		if err != nil {
			ctx.Logger.Warnf("Failed to fetch messages for managed session %s: %v. Scheduling poll.", session.ID, err)
		} else if sm != nil && sm.Complete {
			out := buildOutputFromSessionMessages(refreshed.Status, session.ID, sm)
			if emitErr := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); emitErr != nil {
				cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
				return emitErr
			}
			// Persist terminal status only after successful emit
			mergeSessionIntoMetadata(&metadata, refreshed)
			_ = ctx.Metadata.Set(metadata)
			reclaimSession(client, session.ID, spec.PersistSession, ctx.Logger)
			cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
			cleanupManagedVault(client, ctx, ctx.Logger.Warnf)
			return nil
		} else {
			ctx.Logger.Warnf("Events not complete for session %s after retries. Scheduling poll.", session.ID)
		}
	}

	ctx.Logger.Infof("Started Managed Agent session %s. Waiting for completion (polling)...", session.ID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, initialPoll)
}

func (a *RunAgent) Cleanup(ctx core.SetupContext) error { return nil }

// stopAndReclaim ends a run whose session may still be working: a timeout or a
// dead poll means we stopped watching, not that the agent stopped. The interrupt
// is what makes the delete possible — the API rejects deleting a live session —
// and it stops the agent even when the session is kept for debugging.
func stopAndReclaim(client *Client, sessionID string, persist bool, logger *log.Entry) {
	if err := client.SendManagedSessionInterrupt(sessionID); err != nil {
		logger.Warnf("Failed to interrupt managed session %s: %v", sessionID, err)
	}
	reclaimSession(client, sessionID, persist, logger)
}

// reclaimSession deletes the session unless the step is configured to keep it,
// in which case the transcript stays readable in the Anthropic Console.
func reclaimSession(client *Client, sessionID string, persist bool, logger *log.Entry) {
	if persist {
		logger.Infof("Keeping managed session %s: 'Keep Session After Run' is enabled", sessionID)
		return
	}
	if err := client.DeleteManagedSession(sessionID); err != nil {
		logger.Warnf("Failed to delete managed session %s: %v", sessionID, err)
	}
}

// persistSessionFromConfig reports whether a step is configured to keep its
// session. It defaults to deleting when the configuration cannot be decoded, and
// says so: silently deleting a session the user asked to keep is otherwise
// impossible to diagnose from the Console.
func persistSessionFromConfig(config any, logger *log.Entry) bool {
	spec, err := decodeSpec(config)
	if err != nil {
		logger.Warnf("Cannot read 'Keep Session After Run' from configuration (%v); reclaiming the session", err)
		return false
	}
	return spec.PersistSession
}

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

func cleanupUploadedFiles(client *Client, ctx core.ExecutionContext, logWarn func(string, ...any)) {
	client.CleanupFiles(getUploadedFileIDs(ctx.ExecutionState), logWarn)
}

func cleanupUploadedFilesFromHook(client *Client, ctx core.ActionHookContext, logWarn func(string, ...any)) {
	client.CleanupFiles(getUploadedFileIDs(ctx.ExecutionState), logWarn)
}

func uploadRepositoryFiles(client *Client, ctx core.ExecutionContext, files []string) ([]FileResource, error) {
	resources := make([]FileResource, 0, len(files))
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

		resources = append(resources, FileResource{
			FileID:    fileID,
			MountPath: normalized,
		})
	}
	return resources, nil
}

// cleanupFileResources deletes already-uploaded files on partial failure.
func cleanupFileResources(client *Client, resources []FileResource, logWarn func(string, ...any)) {
	for _, r := range resources {
		if err := client.DeleteFile(r.FileID); err != nil && logWarn != nil {
			logWarn("Failed to delete uploaded file %s: %v", r.FileID, err)
		}
	}
}

// provisionSecretsVault creates a temporary Anthropic vault and adds
// environment_variable credentials for each configured secret binding.
// Returns the vault ID for session creation.
func provisionSecretsVault(client *Client, ctx core.ExecutionContext, secrets []SecretBinding) (string, error) {
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
func cleanupManagedVault(client *Client, ctx core.ExecutionContext, logWarn func(string, ...any)) {
	vaultID, err := ctx.ExecutionState.GetKV("managed_vault_id")
	if err != nil || vaultID == "" {
		return
	}
	if err := client.DeleteVault(vaultID); err != nil && logWarn != nil {
		logWarn("Failed to delete managed vault %s: %v", vaultID, err)
	}
}

// cleanupManagedVaultFromHook is the ActionHookContext variant.
func cleanupManagedVaultFromHook(client *Client, ctx core.ActionHookContext, logWarn func(string, ...any)) {
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
	return nil
}
