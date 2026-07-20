package claude

import (
	"encoding/base64"
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/mitchellh/mapstructure"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	claudePermissionAcceptEdits = "acceptEdits"
	claudeStepPrompt            = "prompt"
	claudeStepBash              = "bash"
	defaultClaudeAllowedTools   = "Bash,Read,Edit,Write"
	envAnthropicAPIKey          = "ANTHROPIC_API_KEY"

	// Prefer plain terminal text in live logs (no Markdown chrome).
	claudeTerminalOutputSystemPrompt = "Write all assistant messages as plain terminal text. " +
		"Do not use Markdown: no bold/italic markers, headings, links, tables, or fenced code blocks. " +
		"Prefer plain paths, shell commands, and simple indentation."
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
	MachineType             string                       `mapstructure:"machine_type"`
	Steps                   []ClaudeCodeStep             `mapstructure:"steps"`
	AnthropicAPIKey         configuration.SecretKeyRef   `mapstructure:"anthropicApiKey"`
	Model                   string                       `mapstructure:"model"`
	WorkingDirectory        string                       `mapstructure:"workingDirectory"`
	Environment             []runner.EnvironmentVariable `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                          `mapstructure:"execution_timeout_seconds"` // 0 = runner.DefaultExecutionTimeoutSeconds

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
// Prepare only checks claude/node and initializes mutable run state.
func buildClaudeCodeBrokerTask(spec RunClaudeCodeSpec) ClaudeCodeBrokerTask {
	model := strings.TrimSpace(spec.Model)
	workdir := strings.TrimSpace(spec.WorkingDirectory)

	files := []runner.BrokerTaskFile{
		{Path: "format.js", Content: streamFormatJS, Mode: "0644"},
		{Path: "write-result.sh", Content: claudeWriteResultScript(), Mode: "0755"},
	}

	stepCommands := make([]runner.BrokerCommand, 0, len(spec.Steps))
	for i, step := range spec.Steps {
		scriptName := claudeStepScriptName(i+1, step.Name)
		var stepScript string
		switch normalizeClaudeStepType(step.Type) {
		case claudeStepBash:
			command := ""
			if step.Command != nil {
				command = *step.Command
			}
			stepScript = buildClaudeBashStepScript(command)
		case claudeStepPrompt:
			prompt := ""
			if step.Prompt != nil {
				prompt = *step.Prompt
			}
			stepScript = buildClaudePromptStepScript(prompt, model)
		}
		files = append(files, runner.BrokerTaskFile{
			Path:    "steps/" + scriptName,
			Content: stepScript,
			Mode:    "0755",
		})
		stepCommands = append(stepCommands, claudeStepBrokerCommand(step.Name, scriptName))
	}

	prepareCommand := runner.BrokerCommand{
		Name:    "Prepare Claude Code",
		Command: "bash -c " + shellSingleQuote(claudePrepareScript(workdir)),
	}
	return ClaudeCodeBrokerTask{
		Commands: append([]runner.BrokerCommand{prepareCommand}, stepCommands...),
		Files:    files,
	}
}

func claudePrepareScript(workdir string) string {
	var prepare strings.Builder
	prepare.WriteString("set -euo pipefail\n")
	prepare.WriteString(claudeCodeStateDirAssignment())
	prepare.WriteString("if ! command -v claude >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"claude CLI not found on PATH; install Claude Code on the runner\" >&2\n")
	prepare.WriteString("  exit 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString("if ! command -v node >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"node not found on PATH; required to format Claude Code live logs\" >&2\n")
	prepare.WriteString("  exit 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString(": >\"$SP/stream.jsonl\"\n")
	prepare.WriteString("printf '0\\n' >\"$SP/prompt_count\"\n")
	if workdir != "" {
		fmt.Fprintf(&prepare, "printf '%%s\\n' %s >\"$SP/workdir\"\n", shellSingleQuote(workdir))
	} else {
		prepare.WriteString("rm -f \"$SP/workdir\"\n")
	}
	return prepare.String()
}

func claudeWriteResultScript() string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString("stream=$1\n")
	b.WriteString("out=$2\n")
	b.WriteString("last=\"\"\n")
	b.WriteString("found=\"\"\n")
	b.WriteString("while IFS= read -r line || [[ -n \"$line\" ]]; do\n")
	b.WriteString("  [[ -z \"$line\" ]] && continue\n")
	b.WriteString("  last=$line\n")
	b.WriteString("  case \"$line\" in\n")
	b.WriteString("    *'\"type\":\"result\"'*|*'\"type\": \"result\"'*) found=$line ;;\n")
	b.WriteString("  esac\n")
	b.WriteString("done <\"$stream\"\n")
	b.WriteString("if [[ -n \"$found\" ]]; then\n")
	b.WriteString("  printf '%s\\n' \"$found\" >\"$out\"\n")
	b.WriteString("elif [[ -n \"$last\" ]]; then\n")
	b.WriteString("  printf '%s\\n' \"$last\" >\"$out\"\n")
	b.WriteString("else\n")
	b.WriteString("  printf '%s\\n' '{}' >\"$out\"\n")
	b.WriteString("fi\n")
	return b.String()
}

func buildClaudeBashStepScript(command string) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString(claudeCodeStateDirAssignment())
	b.WriteString("if [[ -f \"$SP/workdir\" ]]; then\n")
	b.WriteString("  cd \"$(cat \"$SP/workdir\")\"\n")
	b.WriteString("fi\n")
	// Run in this shell (not a nested bash -c) so cd persists, then share
	// the ending directory with later steps via $SP/workdir.
	b.WriteString(strings.TrimRight(command, "\n"))
	b.WriteString("\n")
	b.WriteString("pwd -P >\"$SP/workdir\"\n")
	return b.String()
}

func buildClaudePromptStepScript(prompt, model string) string {
	var b strings.Builder
	b.WriteString("#!/usr/bin/env bash\n")
	b.WriteString("set -euo pipefail\n")
	b.WriteString(claudeCodeStateDirAssignment())
	b.WriteString("if [[ -f \"$SP/workdir\" ]]; then\n")
	b.WriteString("  cd \"$(cat \"$SP/workdir\")\"\n")
	b.WriteString("fi\n")

	promptB64 := base64.StdEncoding.EncodeToString([]byte(prompt))
	fmt.Fprintf(&b, "PROMPT=$(printf '%%s' %s | base64 -d)\n", shellSingleQuote(promptB64))

	b.WriteString("claude_bin=(claude)\n")
	b.WriteString("if command -v stdbuf >/dev/null 2>&1; then\n")
	b.WriteString("  claude_bin=(stdbuf -oL -eL claude)\n")
	b.WriteString("fi\n")

	b.WriteString("claude_args=(--bare -p --output-format stream-json --verbose --include-partial-messages)\n")
	fmt.Fprintf(&b, "claude_args+=(--permission-mode %s)\n", shellSingleQuote(claudePermissionAcceptEdits))
	fmt.Fprintf(&b, "claude_args+=(--append-system-prompt %s)\n", shellSingleQuote(claudeTerminalOutputSystemPrompt))
	if model != "" {
		fmt.Fprintf(&b, "claude_args+=(--model %s)\n", shellSingleQuote(model))
	}
	fmt.Fprintf(&b, "claude_args+=(--allowedTools %s)\n", shellSingleQuote(defaultClaudeAllowedTools))
	b.WriteString("if [[ \"$(cat \"$SP/prompt_count\")\" -gt 0 ]]; then\n")
	b.WriteString("  claude_args+=(--continue)\n")
	b.WriteString("fi\n")
	b.WriteString("\"${claude_bin[@]}\" \"${claude_args[@]}\" -- \"$PROMPT\" \\\n")
	b.WriteString("  | tee -a \"$SP/stream.jsonl\" \\\n")
	b.WriteString("  | node \"$SP/format.js\"\n")
	b.WriteString("printf '%s\\n' \"$(($(cat \"$SP/prompt_count\") + 1))\" >\"$SP/prompt_count\"\n")
	b.WriteString("bash \"$SP/write-result.sh\" \"$SP/stream.jsonl\" \"$SUPERPLANE_RESULT_FILE\"\n")
	return b.String()
}

func claudeCodeStateDirAssignment() string {
	return ": \"${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}\"\nSP=\"$SUPERPLANE_TASK_DIR\"\n"
}

func claudeStepBrokerCommand(stepName, scriptName string) runner.BrokerCommand {
	label := strings.TrimSpace(stepName)
	if label == "" {
		label = scriptName
	}
	return runner.BrokerCommand{
		Name:    label,
		Command: fmt.Sprintf(`bash "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName),
	}
}

func claudeStepScriptName(stepNumber int, name string) string {
	slug := slugifyClaudeStepName(name)
	if slug == "" {
		slug = "step"
	}
	return fmt.Sprintf("%02d-%s.sh", stepNumber, slug)
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
