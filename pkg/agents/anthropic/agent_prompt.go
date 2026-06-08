package anthropic

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
)

//go:embed agent_prompt.md
var defaultAgentPrompt string

func DefaultAgentPrompt() string {
	return defaultAgentPrompt
}

func SyncDefaultAgentPrompt(ctx context.Context, cfg Config) error {
	return SyncAgentPrompt(ctx, cfg, DefaultAgentPrompt())
}

func SyncAgentPrompt(ctx context.Context, cfg Config, prompt string) error {
	if strings.TrimSpace(cfg.AgentID) == "" {
		return fmt.Errorf("anthropic: AgentID is required")
	}
	if strings.TrimSpace(prompt) == "" {
		return fmt.Errorf("anthropic: agent prompt is required")
	}

	client, err := newClient(cfg)
	if err != nil {
		return err
	}

	return syncAgentPrompt(ctx, client, cfg.AgentID, prompt)
}

func syncAgentPrompt(ctx context.Context, client *Client, agentID, prompt string) error {
	current, err := client.getAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}
	if promptsEqual(current.System, prompt) {
		return nil
	}

	updated, err := client.updateAgentSystemPrompt(ctx, agentID, current.Version, prompt)
	if err != nil {
		return fmt.Errorf("update agent prompt: %w", err)
	}
	if !promptsEqual(updated.System, prompt) {
		return fmt.Errorf("update agent prompt: provider returned a different prompt")
	}

	return nil
}

func promptsEqual(a, b string) bool {
	return strings.TrimRight(a, "\r\n") == strings.TrimRight(b, "\r\n")
}
