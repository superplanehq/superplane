package runner

import (
	"encoding/base64"
	"fmt"
	"strings"

	"github.com/mitchellh/mapstructure"

	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	claudePermissionAcceptEdits = "acceptEdits"
	claudeStepPrompt            = "prompt"
	claudeStepBash              = "bash"
	defaultClaudeAllowedTools   = "Bash,Read,Edit,Write"
	envAnthropicAPIKey          = "ANTHROPIC_API_KEY"
)

// ClaudeCodeStep is one ordered bash or prompt action in a Run Claude Code node.
type ClaudeCodeStep struct {
	Name    string  `mapstructure:"name"`
	Type    string  `mapstructure:"type"`
	Prompt  *string `mapstructure:"prompt,omitempty"`
	Command *string `mapstructure:"command,omitempty"`
}

// RunClaudeCodeSpec is persisted runnerClaudeCode node configuration.
type RunClaudeCodeSpec struct {
	MachineType             string                     `mapstructure:"machine_type"`
	Steps                   []ClaudeCodeStep           `mapstructure:"steps"`
	AnthropicAPIKey         configuration.SecretKeyRef `mapstructure:"anthropicApiKey"`
	Model                   string                     `mapstructure:"model"`
	WorkingDirectory        string                     `mapstructure:"workingDirectory"`
	Environment             []EnvironmentVariable      `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                        `mapstructure:"execution_timeout_seconds"` // 0 = DefaultExecutionTimeoutSeconds

	// Legacy fields — migrated into Steps when Steps is empty.
	Prompt              string `mapstructure:"prompt"`
	EnableSetupCommands bool   `mapstructure:"enable_setup_commands"`
	SetupCommands       string `mapstructure:"setup_commands"`
	EnableAfterCommands bool   `mapstructure:"enable_after_commands"`
	AfterCommands       string `mapstructure:"after_commands"`
}

func decodeRunClaudeCodeSpec(raw any) (RunClaudeCodeSpec, error) {
	var spec RunClaudeCodeSpec
	dec, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return RunClaudeCodeSpec{}, fmt.Errorf("runnerClaudeCode spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunClaudeCodeSpec{}, fmt.Errorf("decode runnerClaudeCode configuration: %w", err)
	}
	applyRunClaudeCodeSpecDefaults(&spec)
	return spec, nil
}

func applyRunClaudeCodeSpecDefaults(spec *RunClaudeCodeSpec) {
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = DefaultExecutionTimeoutSeconds
	}
	migrateLegacyClaudeCodeSteps(spec)
}

func migrateLegacyClaudeCodeSteps(spec *RunClaudeCodeSpec) {
	if len(spec.Steps) > 0 {
		return
	}

	var steps []ClaudeCodeStep
	if spec.EnableSetupCommands {
		if cmd := strings.TrimSpace(spec.SetupCommands); cmd != "" {
			steps = append(steps, ClaudeCodeStep{Name: "Setup", Type: claudeStepBash, Command: &cmd})
		}
	}
	if prompt := strings.TrimSpace(spec.Prompt); prompt != "" {
		steps = append(steps, ClaudeCodeStep{Name: "Prompt", Type: claudeStepPrompt, Prompt: &prompt})
	}
	if spec.EnableAfterCommands {
		if cmd := strings.TrimSpace(spec.AfterCommands); cmd != "" {
			steps = append(steps, ClaudeCodeStep{Name: "After", Type: claudeStepBash, Command: &cmd})
		}
	}
	spec.Steps = steps
}

func normalizeClaudeStepType(stepType string) string {
	switch strings.TrimSpace(stepType) {
	case claudeStepBash:
		return claudeStepBash
	default:
		return claudeStepPrompt
	}
}

func validateRunClaudeCodeSpec(spec RunClaudeCodeSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if err := validateClaudeCodeSteps(spec.Steps); err != nil {
		return err
	}
	if !spec.AnthropicAPIKey.IsSet() {
		return fmt.Errorf("anthropic API key is required")
	}
	if err := validateEnvironment(spec.Environment); err != nil {
		return err
	}
	for i, variable := range spec.Environment {
		if strings.TrimSpace(variable.Name) == envAnthropicAPIKey {
			return fmt.Errorf("environment[%d].name cannot be %s; use the Anthropic API Key field", i, envAnthropicAPIKey)
		}
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > maxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", maxExecutionTimeoutSecondsRequest, DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

func validateClaudeCodeSteps(steps []ClaudeCodeStep) error {
	if len(steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	promptCount := 0
	for i, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("steps[%d].name is required", i)
		}
		switch normalizeClaudeStepType(step.Type) {
		case claudeStepBash:
			if step.Command == nil || strings.TrimSpace(*step.Command) == "" {
				return fmt.Errorf("steps[%d].command is required for bash steps", i)
			}
		case claudeStepPrompt:
			if step.Prompt == nil || strings.TrimSpace(*step.Prompt) == "" {
				return fmt.Errorf("steps[%d].prompt is required for prompt steps", i)
			}
			promptCount++
		}
	}
	if promptCount == 0 {
		return fmt.Errorf("at least one prompt step is required")
	}
	return nil
}

// buildClaudeCodeScript returns a Bash script that runs ordered bash/prompt steps
// using the preinstalled `claude` CLI.
//
// Prompt steps use --output-format stream-json; a small Python formatter turns the
// NDJSON into readable live logs. The final stream "result" event is written to
// SUPERPLANE_RESULT_FILE.
func buildClaudeCodeScript(spec RunClaudeCodeSpec) string {
	model := strings.TrimSpace(spec.Model)
	workdir := strings.TrimSpace(spec.WorkingDirectory)

	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n\n")

	if workdir != "" {
		fmt.Fprintf(&b, "cd %s\n\n", shellSingleQuote(workdir))
	}

	b.WriteString("if ! command -v claude >/dev/null 2>&1; then\n")
	b.WriteString("  echo \"claude CLI not found on PATH; install Claude Code on the runner\" >&2\n")
	b.WriteString("  exit 127\n")
	b.WriteString("fi\n")
	b.WriteString("if ! command -v python3 >/dev/null 2>&1; then\n")
	b.WriteString("  echo \"python3 not found on PATH; required to format Claude Code live logs\" >&2\n")
	b.WriteString("  exit 127\n")
	b.WriteString("fi\n\n")

	// Prefer line-buffered stdout so NDJSON events reach the formatter immediately.
	b.WriteString("claude_bin=(claude)\n")
	b.WriteString("if command -v stdbuf >/dev/null 2>&1; then\n")
	b.WriteString("  claude_bin=(stdbuf -oL -eL claude)\n")
	b.WriteString("fi\n\n")

	// stream-json emits NDJSON as the agent works; formatter renders readable activity.
	b.WriteString("claude_args=(--bare -p --output-format stream-json --verbose --include-partial-messages)\n")
	fmt.Fprintf(&b, "claude_args+=(--permission-mode %s)\n", shellSingleQuote(claudePermissionAcceptEdits))
	if model != "" {
		fmt.Fprintf(&b, "claude_args+=(--model %s)\n", shellSingleQuote(model))
	}
	fmt.Fprintf(&b, "claude_args+=(--allowedTools %s)\n", shellSingleQuote(defaultClaudeAllowedTools))
	b.WriteString("\n")
	b.WriteString("stream_file=$(mktemp)\n")
	b.WriteString("format_script=$(mktemp)\n")
	b.WriteString("trap 'rm -f \"$stream_file\" \"$format_script\"' EXIT\n")
	formatterB64 := base64.StdEncoding.EncodeToString([]byte(claudeStreamFormatPython))
	fmt.Fprintf(&b, "printf '%%s' %s | base64 -d >\"$format_script\"\n", shellSingleQuote(formatterB64))
	b.WriteString(": >\"$stream_file\"\n")
	b.WriteString("prompt_count=0\n\n")

	// Extract the last stream-json "result" event into SUPERPLANE_RESULT_FILE.
	b.WriteString("write_claude_result() {\n")
	b.WriteString("  local stream=$1\n")
	b.WriteString("  local out=$2\n")
	b.WriteString("  local last=\"\" found=\"\" line\n")
	b.WriteString("  while IFS= read -r line || [[ -n \"$line\" ]]; do\n")
	b.WriteString("    [[ -z \"$line\" ]] && continue\n")
	b.WriteString("    last=$line\n")
	b.WriteString("    case \"$line\" in\n")
	b.WriteString("      *'\"type\":\"result\"'*|*'\"type\": \"result\"'*) found=$line ;;\n")
	b.WriteString("    esac\n")
	b.WriteString("  done <\"$stream\"\n")
	b.WriteString("  if [[ -n \"$found\" ]]; then\n")
	b.WriteString("    printf '%s\\n' \"$found\" >\"$out\"\n")
	b.WriteString("  elif [[ -n \"$last\" ]]; then\n")
	b.WriteString("    printf '%s\\n' \"$last\" >\"$out\"\n")
	b.WriteString("  else\n")
	b.WriteString("    printf '%s\\n' '{}' >\"$out\"\n")
	b.WriteString("  fi\n")
	b.WriteString("}\n\n")

	for i, step := range spec.Steps {
		switch normalizeClaudeStepType(step.Type) {
		case claudeStepBash:
			command := ""
			if step.Command != nil {
				command = *step.Command
			}
			writeClaudeBashStep(&b, i+1, step.Name, command)
		case claudeStepPrompt:
			prompt := ""
			if step.Prompt != nil {
				prompt = *step.Prompt
			}
			writeClaudePromptStep(&b, i+1, step.Name, prompt)
		}
	}

	b.WriteString("write_claude_result \"$stream_file\" \"$SUPERPLANE_RESULT_FILE\"\n")

	return b.String()
}

func writeClaudeBashStep(b *strings.Builder, stepNumber int, name, command string) {
	fmt.Fprintf(b, "echo %s >&2\n", shellSingleQuote(claudeStepLogLabel(stepNumber, claudeStepBash, name)))
	fmt.Fprintf(b, "bash -c %s\n\n", shellSingleQuote(command))
}

func writeClaudePromptStep(b *strings.Builder, stepNumber int, name, prompt string) {
	promptB64 := base64.StdEncoding.EncodeToString([]byte(prompt))
	fmt.Fprintf(b, "echo %s >&2\n", shellSingleQuote(claudeStepLogLabel(stepNumber, claudeStepPrompt, name)))
	fmt.Fprintf(b, "PROMPT=$(printf '%%s' %s | base64 -d)\n", shellSingleQuote(promptB64))
	b.WriteString("step_args=(\"${claude_args[@]}\")\n")
	b.WriteString("if [[ \"$prompt_count\" -gt 0 ]]; then\n")
	b.WriteString("  step_args+=(--continue)\n")
	b.WriteString("fi\n")
	// Keep raw NDJSON for the finished result; show a human-readable stream in logs.
	b.WriteString("\"${claude_bin[@]}\" \"${step_args[@]}\" -- \"$PROMPT\" \\\n")
	b.WriteString("  | tee -a \"$stream_file\" \\\n")
	b.WriteString("  | python3 -u \"$format_script\"\n")
	b.WriteString("prompt_count=$((prompt_count + 1))\n\n")
}

func claudeStepLogLabel(stepNumber int, stepType, name string) string {
	label := fmt.Sprintf("==> Step %d: %s", stepNumber, stepType)
	if trimmed := strings.TrimSpace(name); trimmed != "" {
		return label + " (" + trimmed + ")"
	}
	return label
}

func shellSingleQuote(value string) string {
	// Wrap in single quotes, escaping embedded single quotes as: '\''
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}

func defaultClaudeCodeSteps() []map[string]any {
	return []map[string]any{
		{"name": "Prompt", "type": claudeStepPrompt, "prompt": ""},
	}
}
