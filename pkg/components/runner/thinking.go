package runner

import (
	"fmt"
	"strings"
)

const (
	ThinkingLevelKey     = "thinkingLevel"
	ThinkingLevelMinimal = "minimal"
	ThinkingLevelLow     = "low"
	ThinkingLevelMedium  = "medium"
	ThinkingLevelHigh    = "high"
	ThinkingLevelXHigh   = "xhigh"
	// DispatchThinkingDefault is the Start override that uses runner Default
	// instead of the saved agent thinking level. Empty dispatch thinking means Auto.
	DispatchThinkingDefault = "default"
)

func NormalizeThinkingLevel(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "", ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh:
		return trimmed, nil
	default:
		return "", fmt.Errorf("thinkingLevel must be empty, low, medium, or high")
	}
}

func NormalizeCodexThinkingLevel(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "", ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh:
		return trimmed, nil
	default:
		return "", fmt.Errorf("thinkingLevel must be empty, minimal, low, medium, high, or xhigh")
	}
}

func NormalizeDispatchThinkingLevel(value string) (string, error) {
	trimmed := strings.ToLower(strings.TrimSpace(value))
	switch trimmed {
	case "", DispatchThinkingDefault, ThinkingLevelMinimal, ThinkingLevelLow, ThinkingLevelMedium, ThinkingLevelHigh, ThinkingLevelXHigh:
		return trimmed, nil
	default:
		return "", fmt.Errorf("thinking_level must be empty, default, minimal, low, medium, high, or xhigh")
	}
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
