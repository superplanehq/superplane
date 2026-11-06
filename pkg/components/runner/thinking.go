package runner

import (
	"fmt"
	"regexp"
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
	// CodexThinkingLevels are fallback efforts for API credentials.
	// Subscriptions use the efforts reported by the native Codex catalog.
	CodexThinkingLevels = []string{
		ThinkingLevelNone, ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh, ThinkingLevelMax,
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

var catalogThinkingLevelPattern = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)

// NormalizeCatalogThinkingLevel normalizes an effort identifier. The selected
// model's catalog determines whether that effort is available.
func NormalizeCatalogThinkingLevel(value string) (string, error) {
	level := strings.ToLower(strings.TrimSpace(value))
	if level == "" {
		return "", nil
	}
	if level == "auto" || level == DispatchThinkingDefault || !catalogThinkingLevelPattern.MatchString(level) {
		return "", fmt.Errorf("thinkingLevel must be a reasoning effort identifier")
	}
	return level, nil
}

func NormalizeOpenCodeThinkingLevel(value string) (string, error) {
	return normalizeThinkingLevelIn(value, OpenCodeThinkingLevels)
}

func NormalizeDispatchThinkingLevel(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	if trimmed == DispatchThinkingDefault {
		return trimmed, nil
	}
	normalized, err := NormalizeCatalogThinkingLevel(trimmed)
	if err != nil {
		return "", fmt.Errorf("thinking_level must be empty, %s, or a reasoning effort identifier", DispatchThinkingDefault)
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
