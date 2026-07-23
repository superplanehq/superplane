package claude

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mitchellh/mapstructure"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	claudeStepPrompt   = "prompt"
	claudeStepBash     = "bash"
	envAnthropicAPIKey = "ANTHROPIC_API_KEY"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// ClaudeCodeStep is one ordered bash or prompt action in a Run Claude Code node.
type ClaudeCodeStep struct {
	Name    string  `mapstructure:"name"`
	Type    string  `mapstructure:"type"`
	Prompt  *string `mapstructure:"prompt,omitempty"`
	Command *string `mapstructure:"command,omitempty"`
}

// RunClaudeCodeSpec is persisted runnerClaudeCode node configuration.
type RunClaudeCodeSpec struct {
	MachineType             string                       `mapstructure:"machineType"`
	Steps                   []ClaudeCodeStep             `mapstructure:"steps"`
	AnthropicAPIKey         configuration.SecretKeyRef   `mapstructure:"anthropicApiKey"`
	Model                   string                       `mapstructure:"model"`
	WorkingDirectory        string                       `mapstructure:"workingDirectory"`
	Environment             []runner.EnvironmentVariable `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                          `mapstructure:"executionTimeoutSeconds"` // 0 = runner.DefaultExecutionTimeoutSeconds

	// Legacy fields — migrated into Steps when Steps is empty.
	Prompt              string `mapstructure:"prompt"`
	EnableSetupCommands bool   `mapstructure:"enable_setup_commands"`
	SetupCommands       string `mapstructure:"setup_commands"`
	EnableAfterCommands bool   `mapstructure:"enable_after_commands"`
	AfterCommands       string `mapstructure:"after_commands"`
}

// ClaudeCodeBrokerTask is the ordered broker commands and task files for a run.
// Helpers (formatter, step scripts) ship via files under SUPERPLANE_TASK_DIR;
// the first command only checks prerequisites and initializes mutable state.
type ClaudeCodeBrokerTask struct {
	Commands []runner.BrokerCommand
	Files    []runner.BrokerTaskFile
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
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
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
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	for i, variable := range spec.Environment {
		if strings.TrimSpace(variable.Name) == envAnthropicAPIKey {
			return fmt.Errorf("environment[%d].name cannot be %s; use the Anthropic API Key field", i, envAnthropicAPIKey)
		}
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
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

// buildClaudeCodeBrokerTask builds broker commands plus task files.
// Static helpers ship via `files` (materialized under SUPERPLANE_TASK_DIR).
// Bash steps are sourced into the runner's shared shell so cwd persists across steps.
func buildClaudeCodeBrokerTask(spec RunClaudeCodeSpec) ClaudeCodeBrokerTask {
	model := strings.TrimSpace(spec.Model)
	workdir := strings.TrimSpace(spec.WorkingDirectory)

	files := []runner.BrokerTaskFile{
		{Path: "run.js", Content: runScript, Mode: "0644"},
		{Path: "prepare.sh", Content: claudePrepareScript(workdir), Mode: "0644"},
	}

	stepCommands := make([]runner.BrokerCommand, 0, len(spec.Steps))
	for i, step := range spec.Steps {
		file, command := buildClaudeCodeStep(i+1, step, model)
		files = append(files, file)
		stepCommands = append(stepCommands, command)
	}

	prepareCommand := runner.BrokerCommand{
		Name:    "Prepare Claude Code",
		Command: `source "$SUPERPLANE_TASK_DIR/prepare.sh"`,
	}
	return ClaudeCodeBrokerTask{
		Commands: append([]runner.BrokerCommand{prepareCommand}, stepCommands...),
		Files:    files,
	}
}

func buildClaudeCodeStep(stepNumber int, step ClaudeCodeStep, model string) (runner.BrokerTaskFile, runner.BrokerCommand) {
	stepSlug := claudeStepSlug(stepNumber, step.Name)
	switch normalizeClaudeStepType(step.Type) {
	case claudeStepBash:
		command := ""
		if step.Command != nil {
			command = *step.Command
		}
		scriptName := stepSlug + ".sh"
		return runner.BrokerTaskFile{
			Path:    "steps/" + scriptName,
			Content: command,
			Mode:    "0644",
		}, claudeBashStepBrokerCommand(step.Name, scriptName)
	default:
		prompt := ""
		if step.Prompt != nil {
			prompt = *step.Prompt
		}
		promptName := stepSlug + ".txt"
		return runner.BrokerTaskFile{
			Path:    "prompts/" + promptName,
			Content: prompt,
			Mode:    "0644",
		}, claudePromptStepBrokerCommand(step.Name, promptName, model)
	}
}

func claudePrepareScript(workdir string) string {
	var prepare strings.Builder
	prepare.WriteString("set -euo pipefail\n")
	prepare.WriteString(": \"${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}\"\n")
	prepare.WriteString("if ! command -v claude >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"claude CLI not found on PATH; install Claude Code on the runner\" >&2\n")
	prepare.WriteString("  return 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString("if ! command -v node >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"node not found on PATH; required to format Claude Code live logs\" >&2\n")
	prepare.WriteString("  return 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString("printf '0\\n' >\"$SUPERPLANE_TASK_DIR/prompt_count\"\n")
	if workdir != "" {
		fmt.Fprintf(&prepare, "cd %s\n", shellSingleQuote(workdir))
	}
	prepare.WriteString("echo \"Claude Code ready\"\n")
	prepare.WriteString("echo \"claude=$(claude --version 2>/dev/null | head -n1)\"\n")
	prepare.WriteString("echo \"node=$(node --version 2>/dev/null)\"\n")
	prepare.WriteString("echo \"cwd=$(pwd -P)\"\n")
	return prepare.String()
}

func claudeBashStepBrokerCommand(stepName, scriptName string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name:    claudeStepLabel(stepName, scriptName),
		Command: fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName),
	}
}

func claudePromptStepBrokerCommand(stepName, promptName, model string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name: claudeStepLabel(stepName, promptName),
		Command: fmt.Sprintf(
			`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
			promptName,
			shellSingleQuote(model),
		),
	}
}

func claudeStepLabel(stepName, fallback string) string {
	if label := strings.TrimSpace(stepName); label != "" {
		return label
	}
	return fallback
}

func claudeStepSlug(stepNumber int, name string) string {
	slug := slugifyClaudeStepName(name)
	if slug == "" {
		slug = "step"
	}
	return fmt.Sprintf("%02d-%s", stepNumber, slug)
}

func slugifyClaudeStepName(name string) string {
	trimmed := strings.TrimSpace(name)
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	for _, r := range strings.ToLower(trimmed) {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('-')
	}
	slug := nonSlugChars.ReplaceAllString(b.String(), "-")
	slug = strings.Trim(slug, "-")
	if len(slug) > 48 {
		slug = strings.Trim(slug[:48], "-")
	}
	return slug
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
