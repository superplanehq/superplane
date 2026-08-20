package claude

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/components/runner"
)

const (
	envAnthropicAPIKey = "ANTHROPIC_API_KEY"
)

// ClaudeCodeStep is one ordered bash or prompt action in a Run Claude Code node.
type ClaudeCodeStep = runner.AgentStep

// RunClaudeCodeSpec is persisted runnerClaudeCode node configuration.
type RunClaudeCodeSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Steps                   []ClaudeCodeStep              `mapstructure:"steps"`
	Credentials             runner.AgentCredentials       `mapstructure:"credentials"`
	Model                   string                        `mapstructure:"model"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"` // 0 = runner.DefaultExecutionTimeoutSeconds

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
	dec, err := runner.NewSpecDecoder(&spec)
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
			steps = append(steps, ClaudeCodeStep{Name: "Setup", Type: runner.AgentStepBash, Command: &cmd})
		}
	}
	if prompt := strings.TrimSpace(spec.Prompt); prompt != "" {
		steps = append(steps, ClaudeCodeStep{Name: "Prompt", Type: runner.AgentStepPrompt, Prompt: &prompt})
	}
	if spec.EnableAfterCommands {
		if cmd := strings.TrimSpace(spec.AfterCommands); cmd != "" {
			steps = append(steps, ClaudeCodeStep{Name: "After", Type: runner.AgentStepBash, Command: &cmd})
		}
	}
	spec.Steps = steps
}

func validateRunClaudeCodeSpec(spec RunClaudeCodeSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	if err := runner.ValidateAgentSteps(spec.Steps); err != nil {
		return err
	}
	if err := runner.ValidateAgentCredentials(spec.Credentials, true); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	if err := runner.ValidateReservedEnvironmentName(spec.Environment, envAnthropicAPIKey); err != nil {
		return err
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
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
	stepSlug := runner.AgentStepSlug(stepNumber, step.Name)
	switch runner.NormalizeAgentStepType(step.Type) {
	case runner.AgentStepBash:
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
		fmt.Fprintf(&prepare, "cd %s\n", runner.ShellSingleQuote(workdir))
	}
	prepare.WriteString("echo \"Claude Code ready\"\n")
	prepare.WriteString("echo \"claude=$(claude --version 2>/dev/null | head -n1)\"\n")
	prepare.WriteString("echo \"node=$(node --version 2>/dev/null)\"\n")
	prepare.WriteString("echo \"cwd=$(pwd -P)\"\n")
	return prepare.String()
}

func claudeBashStepBrokerCommand(stepName, scriptName string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name:    runner.AgentStepLabel(stepName, scriptName),
		Command: fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName),
	}
}

func claudePromptStepBrokerCommand(stepName, promptName, model string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name: runner.AgentStepLabel(stepName, promptName),
		Command: fmt.Sprintf(
			`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
			promptName,
			runner.ShellSingleQuote(model),
		),
	}
}
