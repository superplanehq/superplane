package anthropic

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
	"strings"

	agenttools "github.com/superplanehq/superplane/pkg/agents/agent_tools"
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
	expectedTools := defaultAgentTools()
	current, err := client.getAgent(ctx, agentID)
	if err != nil {
		return fmt.Errorf("get agent: %w", err)
	}
	if promptsEqual(current.System, prompt) && toolsEqual(current.Tools, expectedTools) {
		return nil
	}

	updated, err := client.updateAgentSystemPrompt(ctx, agentID, current.Version, prompt)
	if err != nil {
		return fmt.Errorf("update agent prompt: %w", err)
	}
	if !promptsEqual(updated.System, prompt) {
		return fmt.Errorf("update agent prompt: provider returned a different prompt")
	}
	if !toolsEqual(updated.Tools, expectedTools) {
		return fmt.Errorf("update agent prompt: provider returned different tools")
	}

	return nil
}

func promptsEqual(a, b string) bool {
	return strings.TrimRight(a, "\r\n") == strings.TrimRight(b, "\r\n")
}

func toolsEqual(current json.RawMessage, expected []map[string]any) bool {
	currentByKey, err := toolsByKey(current)
	if err != nil {
		return false
	}

	expectedBytes, err := json.Marshal(expected)
	if err != nil {
		return false
	}

	expectedByKey, err := toolsByKey(expectedBytes)
	if err != nil {
		return false
	}

	if len(currentByKey) != len(expectedByKey) {
		return false
	}
	for key, expectedTool := range expectedByKey {
		currentTool, ok := currentByKey[key]
		if !ok || !toolContainsExpectedFields(currentTool, expectedTool) {
			return false
		}
	}
	return true
}

func toolsByKey(data json.RawMessage) (map[string]map[string]any, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("tools missing")
	}

	var tools []map[string]any
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}

	byKey := map[string]map[string]any{}
	for _, tool := range tools {
		key := toolIdentity(tool)
		if key == "" {
			return nil, fmt.Errorf("tool identity missing")
		}
		if _, exists := byKey[key]; exists {
			return nil, fmt.Errorf("duplicate tool identity")
		}
		byKey[key] = tool
	}
	return byKey, nil
}

func toolContainsExpectedFields(current, expected map[string]any) bool {
	for key, expectedValue := range expected {
		currentValue, ok := current[key]
		if !ok || !valueMatchesExpected(currentValue, expectedValue) {
			return false
		}
	}
	return true
}

func valueMatchesExpected(current, expected any) bool {
	expectedMap, expectedIsMap := expected.(map[string]any)
	if expectedIsMap {
		currentMap, ok := current.(map[string]any)
		return ok && toolContainsExpectedFields(currentMap, expectedMap)
	}

	expectedSlice, expectedIsSlice := expected.([]any)
	if expectedIsSlice {
		currentSlice, ok := current.([]any)
		if !ok || len(currentSlice) != len(expectedSlice) {
			return false
		}
		for i := range expectedSlice {
			if !valueMatchesExpected(currentSlice[i], expectedSlice[i]) {
				return false
			}
		}
		return true
	}

	return fmt.Sprint(current) == fmt.Sprint(expected)
}

func toolIdentity(tool map[string]any) string {
	parts := []string{}
	if value, ok := tool["type"].(string); ok && value != "" {
		parts = append(parts, "type="+value)
	}
	if value, ok := tool["name"].(string); ok && value != "" {
		parts = append(parts, "name="+value)
	}
	sort.Strings(parts)
	return strings.Join(parts, "|")
}

func defaultAgentTools() []map[string]any {
	tools := []map[string]any{
		{
			"type": "agent_toolset_20260401",
		},
	}
	return append(tools, agenttools.DefinitionMaps()...)
}
