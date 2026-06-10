package anthropic

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"sort"
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
	for key, currentTool := range currentByKey {
		if expectedByKey[key] != currentTool {
			return false
		}
	}
	return true
}

func toolsByKey(data json.RawMessage) (map[string]string, error) {
	if len(data) == 0 {
		return nil, fmt.Errorf("tools missing")
	}

	var tools []map[string]any
	if err := json.Unmarshal(data, &tools); err != nil {
		return nil, err
	}

	byKey := map[string]string{}
	for _, tool := range tools {
		key := toolIdentity(tool)
		if key == "" {
			return nil, fmt.Errorf("tool identity missing")
		}
		value, err := json.Marshal(tool)
		if err != nil {
			return nil, err
		}
		if _, exists := byKey[key]; exists {
			return nil, fmt.Errorf("duplicate tool identity")
		}
		byKey[key] = string(value)
	}
	return byKey, nil
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
	return []map[string]any{
		{
			"type": "agent_toolset_20260401",
		},
		{
			"type":        "custom",
			"name":        "superplane_canvas",
			"description": "Read and update the current SuperPlane app canvas without invoking the SuperPlane CLI. Use this tool before shell commands when you need the canvas YAML, console YAML, connected integration IDs, or when you need to save draft graph or console changes. The tool is bound to the current agent session's canvas and will reject attempts to access any other canvas. It never publishes drafts; update_draft only creates or updates the caller's private draft and returns the draft version ID plus validation issues.",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"action": map[string]any{
						"type":        "string",
						"enum":        []string{"read", "update_draft", "list_integrations"},
						"description": "Operation to run. Use read for current YAML, update_draft to save canvas_yaml and/or console_yaml to a draft, and list_integrations for connected integration IDs.",
					},
					"canvas_id": map[string]any{
						"type":        "string",
						"description": "Optional safety check. If provided, it must match the current session canvas_id from the preamble.",
					},
					"use_draft": map[string]any{
						"type":        "boolean",
						"description": "For read. Defaults to true: return the current user's draft when one exists, otherwise live.",
					},
					"include_console": map[string]any{
						"type":        "boolean",
						"description": "For read. Include console.yaml in the response.",
					},
					"include_integrations": map[string]any{
						"type":        "boolean",
						"description": "For read. Include connected integration IDs, names, vendors, and state.",
					},
					"canvas_yaml": map[string]any{
						"type":        "string",
						"description": "For update_draft. Complete canonical canvas.yaml content to save.",
					},
					"console_yaml": map[string]any{
						"type":        "string",
						"description": "For update_draft. Complete canonical console.yaml content to save.",
					},
					"auto_layout": map[string]any{
						"type":        "object",
						"description": "Optional auto-layout settings for canvas_yaml updates. If omitted for a canvas_yaml update, the backend applies horizontal full-canvas auto-layout by default. Omit this for console-only updates.",
						"properties": map[string]any{
							"scope": map[string]any{
								"type": "string",
								"enum": []string{"full_canvas", "connected_component"},
							},
							"node_ids": map[string]any{
								"type":  "array",
								"items": map[string]any{"type": "string"},
							},
						},
					},
				},
				"required": []string{"action"},
			},
		},
		{
			"type":        "custom",
			"name":        "superplane_component_schema",
			"description": "Lookup exact SuperPlane component, trigger, and widget schemas from the backend registry without reading mounted reference files. Use this before read/grep commands or researcher delegation when you need YAML component keys, configuration fields, output channel names, integration requirements, or compact examples. Prefer this tool for repeated schema lookups; mounted docs are fallback only.",
			"input_schema": map[string]any{
				"type": "object",
				"properties": map[string]any{
					"component_keys": map[string]any{
						"type":        "array",
						"description": "Exact component, trigger, or widget keys to look up, for example schedule, http, wait, slack.waitForButtonClick.",
						"items":       map[string]any{"type": "string"},
					},
					"vendors": map[string]any{
						"type":        "array",
						"description": "Vendor names to list schemas for, for example slack, github, grafana.",
						"items":       map[string]any{"type": "string"},
					},
					"query": map[string]any{
						"type":        "string",
						"description": "Search term used against component keys, labels, descriptions, kind, and required integration vendor.",
					},
					"include_examples": map[string]any{
						"type":        "boolean",
						"description": "Include compact example input/output payloads when available. Honored only for exact component_keys lookups; broad vendor and query lookups stay compact.",
					},
					"limit": map[string]any{
						"type":        "integer",
						"description": "Maximum schemas to return. Defaults to 40 and is capped at 40.",
					},
				},
			},
		},
	}
}
