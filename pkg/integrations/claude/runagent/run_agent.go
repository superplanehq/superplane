package runagent

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
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

	aid := strings.TrimSpace(spec.Agent)
	createReq := CreateManagedSessionRequest{
		Agent:         aid,
		AgentVersion:  spec.Version,
		EnvironmentID: strings.TrimSpace(spec.EnvironmentID),
		VaultIDs:      spec.VaultIDs,
		Resources:     resources,
	}

	session, err := client.CreateManagedSession(createReq)
	if err != nil {
		return fmt.Errorf("failed to create managed agent session: %w", err)
	}

	metadata := ExecutionMetadata{}
	mergeSessionIntoMetadata(&metadata, session)
	if err := ctx.Metadata.Set(metadata); err != nil {
		return fmt.Errorf("failed to set execution metadata: %w", err)
	}

	if err := ctx.ExecutionState.SetKV("managed_session_id", session.ID); err != nil {
		return fmt.Errorf("failed to set managed_session_id: %w", err)
	}

	if len(resources) > 0 {
		fileIDs := make([]string, len(resources))
		for i, r := range resources {
			fileIDs[i] = r.FileID
		}
		encoded, _ := json.Marshal(fileIDs)
		if err := ctx.ExecutionState.SetKV("uploaded_file_ids", string(encoded)); err != nil {
			return fmt.Errorf("failed to store uploaded file IDs: %w", err)
		}
	}

	if err := client.SendManagedSessionUserMessage(session.ID, spec.Prompt); err != nil {
		return fmt.Errorf("failed to send user message: %w", err)
	}

	// Check if session already finished (fast tasks).
	// Don't write terminal status to metadata yet — only after emit.
	refreshed, err := client.GetManagedSession(session.ID)
	if err != nil {
		return fmt.Errorf("failed to get session: %w", err)
	}

	if refreshed != nil && isSessionTerminal(refreshed.Status) {
		sm, err := client.GetSessionMessagesWithRetry(session.ID, finalMessageReads, finalMessageDelay)
		if err != nil {
			ctx.Logger.Warnf("Failed to fetch messages for managed session %s: %v. Scheduling poll.", session.ID, err)
		} else if sm != nil && sm.Complete {
			out := buildOutputFromSessionMessages(refreshed.Status, session.ID, sm)
			if emitErr := ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out}); emitErr != nil {
				return emitErr
			}
			// Persist terminal status only after successful emit
			mergeSessionIntoMetadata(&metadata, refreshed)
			_ = ctx.Metadata.Set(metadata)
			if err := client.DeleteManagedSession(session.ID); err != nil {
				ctx.Logger.Warnf("Failed to delete managed session %s: %v", session.ID, err)
			}
			cleanupUploadedFiles(client, ctx, ctx.Logger.Warnf)
			return nil
		} else {
			ctx.Logger.Warnf("Events not complete for session %s after retries. Scheduling poll.", session.ID)
		}
	}

	ctx.Logger.Infof("Started Managed Agent session %s. Waiting for completion (polling)...", session.ID)
	return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, initialPoll)
}

func (a *RunAgent) Cleanup(ctx core.SetupContext) error { return nil }

// uploadRepositoryFiles reads files from the canvas repository, uploads them
// to the Anthropic Files API, and returns FileResource entries for session mounting.
// cleanupUploadedFiles deletes any files uploaded to the Anthropic Files API
// for this execution. File IDs are retrieved from execution state.
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
			return nil, fmt.Errorf("invalid file path %q: %w", path, err)
		}

		reader, err := ctx.Files.Read(normalized)
		if err != nil {
			return nil, fmt.Errorf("read file %q: %w", path, err)
		}

		fileID, err := client.UploadFile(reader, normalized)
		reader.Close()
		if err != nil {
			return nil, fmt.Errorf("upload file %q: %w", path, err)
		}

		resources = append(resources, FileResource{
			FileID:    fileID,
			MountPath: filepath.Base(normalized),
		})
	}
	return resources, nil
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
