package runagent

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
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
	}
}

func (a *RunAgent) Setup(ctx core.SetupContext) error {
	spec, err := decodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateSpec(spec)
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

	aid := strings.TrimSpace(spec.Agent)
	createReq := CreateManagedSessionRequest{
		Agent:         aid,
		AgentVersion:  spec.Version,
		EnvironmentID: strings.TrimSpace(spec.EnvironmentID),
		VaultIDs:      spec.VaultIDs,
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

	if err := client.SendManagedSessionUserMessage(session.ID, spec.Prompt); err != nil {
		return fmt.Errorf("failed to send user message: %w", err)
	}

	// Stream session events until completion.
	// This captures agent messages in real-time, avoiding the eventual
	// consistency issue with the events list API.
	ctx.Logger.Infof("Started Managed Agent session %s. Streaming events...", session.ID)
	streamCtx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()
	status, lastMessage, messages, streamErr := client.StreamSessionUntilIdle(streamCtx, session.ID)
	if streamErr != nil {
		ctx.Logger.Warnf("Stream failed for session %s: %v. Falling back to poll.", session.ID, streamErr)
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{"attempt": 1, "errors": 0}, initialPoll)
	}

	// Update metadata with final status
	metadata.Session.Status = status
	_ = ctx.Metadata.Set(metadata)

	// Clean up the session on Anthropic's side
	if err := client.DeleteManagedSession(session.ID); err != nil {
		ctx.Logger.Warnf("Failed to delete managed session %s: %v", session.ID, err)
	}

	out := buildOutput(status, session.ID, lastMessage, messages)
	return ctx.ExecutionState.Emit(defaultChannel, payloadType, []any{out})
}

func (a *RunAgent) Cleanup(ctx core.SetupContext) error { return nil }

func decodeSpec(config any) (Spec, error) {
	var spec Spec
	if err := mapstructure.Decode(config, &spec); err != nil {
		return spec, fmt.Errorf("failed to decode configuration: %w", err)
	}
	if raw, ok := config.(map[string]any); ok {
		if v, ok := raw["vaultIds"]; ok {
			spec.VaultIDs = decodeStringList(v)
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
