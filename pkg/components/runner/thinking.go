package runner

import (
	"fmt"
	"slices"
	"strings"
)

const (
	ThinkingLevelKey     = "thinkingLevel"
	ThinkingLevelNone    = "none"
	ThinkingLevelMinimal = "minimal"
	ThinkingLevelLow     = "low"
	ThinkingLevelMedium  = "medium"
	ThinkingLevelHigh    = "high"
	ThinkingLevelXHigh   = "xhigh"
	ThinkingLevelMax     = "max"
	// DispatchThinkingDefault is the Start override that uses runner Default
	// instead of the saved agent thinking level. Empty dispatch thinking means Auto.
	DispatchThinkingDefault = "default"
)

// Levels that each agent CLI accepts. A model can support fewer levels;
// see ModelThinkingLevels.
var (
	// ClaudeCodeThinkingLevels are the values of `claude --effort`.
	ClaudeCodeThinkingLevels = []string{
		ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh, ThinkingLevelMax,
	}
	// CodexThinkingLevels are the values of Codex `model_reasoning_effort`.
	CodexThinkingLevels = []string{
		ThinkingLevelNone, ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh,
	}
	// OpenCodeThinkingLevels are the `opencode run --variant` values that
	// SuperPlane sends.
	OpenCodeThinkingLevels = []string{
		ThinkingLevelNone, ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh, ThinkingLevelMax,
	}
	// allThinkingLevels lists every known level in display order.
	allThinkingLevels = OpenCodeThinkingLevels
)

// NormalizeThinkingLevel accepts an empty value or any level that one of the
// agent CLIs supports.
func NormalizeThinkingLevel(value string) (string, error) {
	return normalizeThinkingLevelIn(value, allThinkingLevels)
}

func NormalizeClaudeCodeThinkingLevel(value string) (string, error) {
	return normalizeThinkingLevelIn(value, ClaudeCodeThinkingLevels)
}

func NormalizeCodexThinkingLevel(value string) (string, error) {
	return normalizeThinkingLevelIn(value, CodexThinkingLevels)
}

func NormalizeOpenCodeThinkingLevel(value string) (string, error) {
	return normalizeThinkingLevelIn(value, OpenCodeThinkingLevels)
}

func NormalizeDispatchThinkingLevel(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == DispatchThinkingDefault {
		return trimmed, nil
	}
	normalized, err := NormalizeThinkingLevel(trimmed)
	if err != nil {
		return "", fmt.Errorf("thinking_level must be empty, %s, or one of: %s", DispatchThinkingDefault, strings.Join(allThinkingLevels, ", "))
	}
	return normalized, nil
}

func normalizeThinkingLevelIn(value string, allowed []string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == "" || slices.Contains(allowed, trimmed) {
		return trimmed, nil
	}
	return "", fmt.Errorf("thinkingLevel must be empty or one of: %s", strings.Join(allowed, ", "))
}

func OverlayThinkingLevel(agentValue, dispatchValue string) (string, bool) {
	dispatch, err := NormalizeDispatchThinkingLevel(dispatchValue)
	if err != nil || dispatch == "" {
		return agentValue, false
	}
	if dispatch == DispatchThinkingDefault {
		return "", true
	}
	return dispatch, true
}

func PromptNodeCommand(promptName, model, thinking string) string {
	command := fmt.Sprintf(
		`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
		promptName,
		ShellSingleQuote(model),
	)
	if thinking = strings.TrimSpace(thinking); thinking != "" {
		command += " " + ShellSingleQuote(thinking)
	}
	return command
}

func FollowUpLoopCommand(model, thinking string) string {
	command := fmt.Sprintf(`node "$SUPERPLANE_TASK_DIR/follow_up_loop.js" %s`, ShellSingleQuote(model))
	if thinking = strings.TrimSpace(thinking); thinking != "" {
		command += " " + ShellSingleQuote(thinking)
	}
	return command
}
