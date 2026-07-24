package opencode

import (
	"fmt"
	"regexp"
	"strings"
	"unicode"

	"github.com/superplanehq/superplane/pkg/components/runner"
	"github.com/superplanehq/superplane/pkg/configuration"
)

const (
	openCodeStepPrompt = "prompt"
	openCodeStepBash   = "bash"
)

var nonSlugChars = regexp.MustCompile(`[^a-z0-9]+`)

// providerCredential describes one curated OpenCode provider and the
// environment variable that OpenCode reads its API key from.
type providerCredential struct {
	Value  string // stored provider identifier
	Label  string // UI label
	EnvVar string // environment variable OpenCode reads
}

// openCodeProviders is the curated list of provider API keys, in display order.
// OpenCode reads provider keys from these well-known environment variables.
var openCodeProviders = []providerCredential{
	{Value: "anthropic", Label: "Anthropic", EnvVar: "ANTHROPIC_API_KEY"},
	{Value: "openai", Label: "OpenAI", EnvVar: "OPENAI_API_KEY"},
	{Value: "google", Label: "Google (Gemini)", EnvVar: "GOOGLE_GENERATIVE_AI_API_KEY"},
	{Value: "openrouter", Label: "OpenRouter", EnvVar: "OPENROUTER_API_KEY"},
	{Value: "groq", Label: "Groq", EnvVar: "GROQ_API_KEY"},
	{Value: "xai", Label: "xAI (Grok)", EnvVar: "XAI_API_KEY"},
	{Value: "deepseek", Label: "DeepSeek", EnvVar: "DEEPSEEK_API_KEY"},
	{Value: "mistral", Label: "Mistral", EnvVar: "MISTRAL_API_KEY"},
}

func providerByValue(value string) (providerCredential, bool) {
	for _, provider := range openCodeProviders {
		if provider.Value == value {
			return provider, true
		}
	}
	return providerCredential{}, false
}

func providerFieldOptions() []configuration.FieldOption {
	options := make([]configuration.FieldOption, 0, len(openCodeProviders))
	for _, provider := range openCodeProviders {
		options = append(options, configuration.FieldOption{
			Label:       provider.Label,
			Value:       provider.Value,
			Description: "Sets " + provider.EnvVar,
		})
	}
	return options
}

// OpenCodeStep is one ordered bash or prompt action in a Run OpenCode node.
type OpenCodeStep struct {
	Name    string  `mapstructure:"name"`
	Type    string  `mapstructure:"type"`
	Prompt  *string `mapstructure:"prompt,omitempty"`
	Command *string `mapstructure:"command,omitempty"`
}

// RunOpenCodeSpec is persisted runnerOpenCode node configuration.
type RunOpenCodeSpec struct {
	MachineType             string                        `mapstructure:"machineType"`
	Provider                string                        `mapstructure:"provider"`
	Secret                  configuration.SecretKeyRef    `mapstructure:"secret"`
	Model                   string                        `mapstructure:"model"`
	Steps                   []OpenCodeStep                `mapstructure:"steps"`
	WorkingDirectory        string                        `mapstructure:"workingDirectory"`
	EnvironmentFrom         []runner.EnvironmentFromEntry `mapstructure:"environmentFrom"`
	Environment             []runner.EnvironmentVariable  `mapstructure:"environment"`
	ExecutionTimeoutSeconds int                           `mapstructure:"executionTimeoutSeconds"` // 0 = runner.DefaultExecutionTimeoutSeconds
}

// modelRef builds the OpenCode `provider/model` identifier from the configured
// provider and bare model name (for example anthropic + claude-sonnet-4-5).
func (s RunOpenCodeSpec) modelRef() string {
	return openCodeModelRef(s.Provider, s.Model)
}

func openCodeModelRef(provider, model string) string {
	return strings.TrimSpace(provider) + "/" + strings.TrimSpace(model)
}

// OpenCodeBrokerTask is the ordered broker commands and task files for a run.
// Helpers (formatter, step scripts) ship via files under SUPERPLANE_TASK_DIR;
// the first command only checks prerequisites and initializes mutable state.
type OpenCodeBrokerTask struct {
	Commands []runner.BrokerCommand
	Files    []runner.BrokerTaskFile
}

func decodeRunOpenCodeSpec(raw any) (RunOpenCodeSpec, error) {
	var spec RunOpenCodeSpec
	dec, err := runner.NewSpecDecoder(&spec)
	if err != nil {
		return RunOpenCodeSpec{}, fmt.Errorf("runnerOpenCode spec decoder: %w", err)
	}
	if err := dec.Decode(raw); err != nil {
		return RunOpenCodeSpec{}, fmt.Errorf("decode runnerOpenCode configuration: %w", err)
	}
	applyRunOpenCodeSpecDefaults(&spec)
	return spec, nil
}

func applyRunOpenCodeSpecDefaults(spec *RunOpenCodeSpec) {
	if spec.ExecutionTimeoutSeconds <= 0 {
		spec.ExecutionTimeoutSeconds = runner.DefaultExecutionTimeoutSeconds
	}
}

func normalizeOpenCodeStepType(stepType string) string {
	switch strings.TrimSpace(stepType) {
	case openCodeStepBash:
		return openCodeStepBash
	default:
		return openCodeStepPrompt
	}
}

func validateRunOpenCodeSpec(spec RunOpenCodeSpec) error {
	if strings.TrimSpace(spec.MachineType) == "" {
		return fmt.Errorf("machine type is required")
	}
	provider, err := validateOpenCodeProvider(spec.Provider)
	if err != nil {
		return err
	}
	if !spec.Secret.IsSet() {
		return fmt.Errorf("API key is required")
	}
	if err := validateOpenCodeModel(spec.Model); err != nil {
		return err
	}
	if err := validateOpenCodeSteps(spec.Steps); err != nil {
		return err
	}
	if err := runner.ValidateEnvironmentFrom(spec.EnvironmentFrom); err != nil {
		return err
	}
	if err := runner.ValidateEnvironment(spec.Environment); err != nil {
		return err
	}
	for i, variable := range spec.Environment {
		if strings.TrimSpace(variable.Name) == provider.EnvVar {
			return fmt.Errorf("environment[%d].name cannot be %s; use the API key field", i, provider.EnvVar)
		}
	}
	if spec.ExecutionTimeoutSeconds != 0 {
		if spec.ExecutionTimeoutSeconds < 1 || spec.ExecutionTimeoutSeconds > runner.MaxExecutionTimeoutSecondsRequest {
			return fmt.Errorf("execution timeout must be between 1 and %d seconds, or 0 to use the default (%d seconds)", runner.MaxExecutionTimeoutSecondsRequest, runner.DefaultExecutionTimeoutSeconds)
		}
	}
	return nil
}

func validateOpenCodeProvider(value string) (providerCredential, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return providerCredential{}, fmt.Errorf("provider is required")
	}
	provider, ok := providerByValue(trimmed)
	if !ok {
		return providerCredential{}, fmt.Errorf("provider is not supported: %s", trimmed)
	}
	return provider, nil
}

func validateOpenCodeModel(model string) error {
	if strings.TrimSpace(model) == "" {
		return fmt.Errorf("model is required (the model name on the selected provider, for example claude-sonnet-4-5)")
	}
	return nil
}

func validateOpenCodeSteps(steps []OpenCodeStep) error {
	if len(steps) == 0 {
		return fmt.Errorf("at least one step is required")
	}

	promptCount := 0
	for i, step := range steps {
		if strings.TrimSpace(step.Name) == "" {
			return fmt.Errorf("steps[%d].name is required", i)
		}
		switch normalizeOpenCodeStepType(step.Type) {
		case openCodeStepBash:
			if step.Command == nil || strings.TrimSpace(*step.Command) == "" {
				return fmt.Errorf("steps[%d].command is required for bash steps", i)
			}
		case openCodeStepPrompt:
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

// buildOpenCodeBrokerTask builds broker commands plus task files.
// Static helpers ship via `files` (materialized under SUPERPLANE_TASK_DIR).
// Bash steps are sourced into the runner's shared shell so cwd persists across steps.
func buildOpenCodeBrokerTask(spec RunOpenCodeSpec) OpenCodeBrokerTask {
	model := spec.modelRef()
	workdir := strings.TrimSpace(spec.WorkingDirectory)

	files := []runner.BrokerTaskFile{
		{Path: "run.js", Content: runScript, Mode: "0644"},
		{Path: "prepare.sh", Content: openCodePrepareScript(workdir), Mode: "0644"},
	}

	stepCommands := make([]runner.BrokerCommand, 0, len(spec.Steps))
	for i, step := range spec.Steps {
		file, command := buildOpenCodeStep(i+1, step, model)
		files = append(files, file)
		stepCommands = append(stepCommands, command)
	}

	prepareCommand := runner.BrokerCommand{
		Name:    "Prepare OpenCode",
		Command: `source "$SUPERPLANE_TASK_DIR/prepare.sh"`,
	}
	return OpenCodeBrokerTask{
		Commands: append([]runner.BrokerCommand{prepareCommand}, stepCommands...),
		Files:    files,
	}
}

func buildOpenCodeStep(stepNumber int, step OpenCodeStep, model string) (runner.BrokerTaskFile, runner.BrokerCommand) {
	stepSlug := openCodeStepSlug(stepNumber, step.Name)
	switch normalizeOpenCodeStepType(step.Type) {
	case openCodeStepBash:
		command := ""
		if step.Command != nil {
			command = *step.Command
		}
		scriptName := stepSlug + ".sh"
		return runner.BrokerTaskFile{
			Path:    "steps/" + scriptName,
			Content: command,
			Mode:    "0644",
		}, openCodeBashStepBrokerCommand(step.Name, scriptName)
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
		}, openCodePromptStepBrokerCommand(step.Name, promptName, model)
	}
}

func openCodePrepareScript(workdir string) string {
	var prepare strings.Builder
	prepare.WriteString("set -euo pipefail\n")
	prepare.WriteString(": \"${SUPERPLANE_TASK_DIR:?SUPERPLANE_TASK_DIR is required}\"\n")
	prepare.WriteString("if ! command -v opencode >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"opencode CLI not found on PATH; install OpenCode on the runner\" >&2\n")
	prepare.WriteString("  return 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString("if ! command -v node >/dev/null 2>&1; then\n")
	prepare.WriteString("  echo \"node not found on PATH; required to format OpenCode live logs\" >&2\n")
	prepare.WriteString("  return 127\n")
	prepare.WriteString("fi\n")
	prepare.WriteString("rm -f \"$SUPERPLANE_TASK_DIR/session_id\"\n")
	if workdir != "" {
		fmt.Fprintf(&prepare, "cd %s\n", shellSingleQuote(workdir))
	}
	prepare.WriteString("echo \"OpenCode ready\"\n")
	prepare.WriteString("echo \"opencode=$(opencode --version 2>/dev/null | head -n1)\"\n")
	prepare.WriteString("echo \"node=$(node --version 2>/dev/null)\"\n")
	prepare.WriteString("echo \"cwd=$(pwd -P)\"\n")
	return prepare.String()
}

func openCodeBashStepBrokerCommand(stepName, scriptName string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name:    openCodeStepLabel(stepName, scriptName),
		Command: fmt.Sprintf(`source "$SUPERPLANE_TASK_DIR/steps/%s"`, scriptName),
	}
}

func openCodePromptStepBrokerCommand(stepName, promptName, model string) runner.BrokerCommand {
	return runner.BrokerCommand{
		Name: openCodeStepLabel(stepName, promptName),
		Command: fmt.Sprintf(
			`node "$SUPERPLANE_TASK_DIR/run.js" "$SUPERPLANE_TASK_DIR/prompts/%s" %s`,
			promptName,
			shellSingleQuote(model),
		),
	}
}

func openCodeStepLabel(stepName, fallback string) string {
	if label := strings.TrimSpace(stepName); label != "" {
		return label
	}
	return fallback
}

func openCodeStepSlug(stepNumber int, name string) string {
	slug := slugifyOpenCodeStepName(name)
	if slug == "" {
		slug = "step"
	}
	return fmt.Sprintf("%02d-%s", stepNumber, slug)
}

func slugifyOpenCodeStepName(name string) string {
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

func defaultOpenCodeSteps() []map[string]any {
	return []map[string]any{
		{"name": "Prompt", "type": openCodeStepPrompt, "prompt": ""},
	}
}
