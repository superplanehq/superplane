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
	defaultClaudeAllowedTools   = "Bash,Read,Edit,Write"
	envAnthropicAPIKey          = "ANTHROPIC_API_KEY"
)

// RunClaudeCodeSpec is persisted runnerClaudeCode node configuration.
type RunClaudeCodeSpec struct {
	MachineType             string                     `mapstructure:"machine_type"`
	Prompt                  string                     `mapstructure:"prompt"`
	AnthropicAPIKey         configuration.SecretKeyRef `mapstructure:"anthropicApiKey"`
	Model                   string                     `mapstructure:"model"`
	WorkingDirectory        string                     `mapstructure:"workingDirectory"`
	EnableSetupCommands     bool                       `mapstructure:"enable_setup_commands"`
	SetupCommands           string                     `mapstructure:"setup_commands"`
	EnableAfterCommands     bool                       `mapstructure:"enable_after_commands"`
	AfterCommands           string                     `mapstructure:"after_commands"`
	Environment             []EnvironmentVariable      `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                        `mapstructure:"execution_timeout_seconds"` // 0 = DefaultExecutionTimeoutSeconds
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
}

func validateRunClaudeCodeSpec(spec RunClaudeCodeSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if strings.TrimSpace(spec.Prompt) == "" {
		return fmt.Errorf("prompt is required")
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
	if spec.EnableSetupCommands {
		if err := validateCommands(spec.SetupCommands); err != nil {
			return fmt.Errorf("setup commands: %w", err)
		}
	}
	if spec.EnableAfterCommands {
		if err := validateCommands(spec.AfterCommands); err != nil {
			return fmt.Errorf("after commands: %w", err)
		}
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > maxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", maxExecutionTimeoutSecondsRequest, DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

// buildClaudeCodeScript returns a Bash script that invokes the preinstalled `claude` CLI
// in headless mode and writes JSON output to SUPERPLANE_RESULT_FILE.
// Optional after-commands run in the same shell after Claude succeeds.
func buildClaudeCodeScript(spec RunClaudeCodeSpec) string {
	promptB64 := base64.StdEncoding.EncodeToString([]byte(spec.Prompt))
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
	b.WriteString("fi\n\n")

	fmt.Fprintf(&b, "PROMPT=$(printf '%%s' %s | base64 -d)\n\n", shellSingleQuote(promptB64))

	b.WriteString("args=(--bare -p --output-format json)\n")
	fmt.Fprintf(&b, "args+=(--permission-mode %s)\n", shellSingleQuote(claudePermissionAcceptEdits))

	if model != "" {
		fmt.Fprintf(&b, "args+=(--model %s)\n", shellSingleQuote(model))
	}
	fmt.Fprintf(&b, "args+=(--allowedTools %s)\n", shellSingleQuote(defaultClaudeAllowedTools))

	b.WriteString("\n")
	b.WriteString("result_file=$(mktemp)\n")
	b.WriteString("trap 'rm -f \"$result_file\"' EXIT\n\n")
	b.WriteString("claude \"${args[@]}\" -- \"$PROMPT\" | tee \"$result_file\"\n\n")
	b.WriteString("if [[ -s \"$result_file\" ]]; then\n")
	b.WriteString("  cp \"$result_file\" \"$SUPERPLANE_RESULT_FILE\"\n")
	b.WriteString("else\n")
	b.WriteString("  printf '%s\\n' '{}' >\"$SUPERPLANE_RESULT_FILE\"\n")
	b.WriteString("fi\n")

	if spec.EnableAfterCommands {
		for _, command := range normalizeCommands(spec.AfterCommands) {
			b.WriteString("\n")
			b.WriteString(command)
			b.WriteString("\n")
		}
	}

	return b.String()
}

func shellSingleQuote(value string) string {
	// Wrap in single quotes, escaping embedded single quotes as: '\''
	return "'" + strings.ReplaceAll(value, "'", `'\''`) + "'"
}
