package openai

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/agentcli"
)

const (
	CodexAgentPayloadType = "openai.codexAgent.finished"

	CodexAgentOutputChannelSuccess = "success"
	CodexAgentOutputChannelFailed  = "failed"

	defaultCodexModel            = "gpt-5.1-codex-mini"
	defaultCodexSandbox          = "read-only"
	defaultCodexWorkingDirectory = "/app"
	defaultCodexTimeoutSeconds   = 600
	maxCodexTimeoutSeconds       = 86400
)

var codexSandboxOptions = []string{"read-only", "workspace-write", "danger-full-access"}

type RunCodexAgent struct {
	runner agentcli.Runner
}

type RunCodexAgentSpec struct {
	Model            string `json:"model" mapstructure:"model"`
	Prompt           string `json:"prompt" mapstructure:"prompt"`
	Sandbox          string `json:"sandbox" mapstructure:"sandbox"`
	WorkingDirectory string `json:"workingDirectory" mapstructure:"workingDirectory"`
	TimeoutSeconds   int    `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
}

type CodexAgentPayload struct {
	Text             string           `json:"text"`
	ExitCode         int              `json:"exitCode"`
	TimedOut         bool             `json:"timedOut"`
	Model            string           `json:"model"`
	Sandbox          string           `json:"sandbox"`
	WorkingDirectory string           `json:"workingDirectory"`
	DurationMs       int64            `json:"durationMs"`
	Events           []map[string]any `json:"events,omitempty"`
	Stdout           string           `json:"stdout,omitempty"`
	Stderr           string           `json:"stderr,omitempty"`
}

func (a *RunCodexAgent) Name() string {
	return "openai.runCodexAgent"
}

func (a *RunCodexAgent) Label() string {
	return "Run Codex Agent"
}

func (a *RunCodexAgent) Description() string {
	return "Run OpenAI Codex CLI non-interactively on a local workspace"
}

func (a *RunCodexAgent) Documentation() string {
	return `The Run Codex Agent component runs the OpenAI Codex CLI in non-interactive mode from the SuperPlane app container.

## Use Cases

- **Repository analysis**: Ask Codex to inspect a codebase and summarize findings
- **Automated code review**: Run targeted review prompts against a mounted repository
- **Local coding tasks**: Let Codex propose or make workspace changes when write-capable sandbox modes are explicitly selected
- **Workflow automation**: Convert upstream events into coding-agent tasks

## Configuration

- **Model**: Codex model to use. Defaults to ` + "`gpt-5.1-codex-mini`" + `.
- **Prompt**: The task sent to Codex (supports expressions).
- **Sandbox**: CLI sandbox mode. Defaults to read-only. Workspace write and full access modes should only be used in trusted local/dev environments.
- **Working Directory**: Directory passed to Codex as the workspace root. Defaults to ` + "`/app`" + `.
- **Timeout**: Maximum runtime in seconds. Defaults to 600 seconds.

## Output

Routes to one of two channels:
- **success**: Codex exits with code 0
- **failed**: Codex exits non-zero or times out

The payload includes the final Codex message, exit code, timeout flag, selected model, sandbox mode, working directory, duration, parsed JSONL events when available, and stderr/stdout for failures.

## Notes

- Requires the Codex CLI on the app container PATH.
- Requires a valid OpenAI API key configured on the OpenAI integration.
- The API key is passed only to the Codex subprocess and is never emitted in the payload.
- The component runs locally in the SuperPlane app container, so sandbox modes should be chosen carefully.`
}

func (a *RunCodexAgent) Icon() string {
	return "openai"
}

func (a *RunCodexAgent) Color() string {
	return "gray"
}

func (a *RunCodexAgent) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      CodexAgentPayloadType,
		"timestamp": "2026-05-30T12:00:00Z",
		"data": map[string]any{
			"text":             "I reviewed the repository and found no obvious regression in the requested files.",
			"exitCode":         0,
			"timedOut":         false,
			"model":            defaultCodexModel,
			"sandbox":          defaultCodexSandbox,
			"workingDirectory": "/app",
			"durationMs":       12420,
		},
	}
}

func (a *RunCodexAgent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: CodexAgentOutputChannelSuccess, Label: "Success"},
		{Name: CodexAgentOutputChannelFailed, Label: "Failed"},
	}
}

func (a *RunCodexAgent) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     defaultCodexModel,
			Placeholder: defaultCodexModel,
			Description: "Codex model to use",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "model"},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Review this repository for the issue described in {{ previous().data.title }}",
			Description: "Task sent to Codex. Supports expressions.",
		},
		{
			Name:        "sandbox",
			Label:       "Sandbox",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     defaultCodexSandbox,
			Description: "Sandbox mode for model-generated shell commands",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
					{Label: "Read-only", Value: "read-only", Description: "Codex can inspect files but cannot write to the workspace."},
					{Label: "Workspace write", Value: "workspace-write", Description: "Codex can write inside the workspace."},
					{Label: "Full access", Value: "danger-full-access", Description: "Dangerous: disables sandbox restrictions for local/dev use only."},
				}},
			},
		},
		{
			Name:        "workingDirectory",
			Label:       "Working Directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     defaultCodexWorkingDirectory,
			Placeholder: defaultCodexWorkingDirectory,
			Description: "Directory used as the Codex workspace root",
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultCodexTimeoutSeconds,
			Description: "Maximum runtime in seconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: intPtr(1),
					Max: intPtr(maxCodexTimeoutSeconds),
				},
			},
		},
	}
}

func (a *RunCodexAgent) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunCodexAgentSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateRunCodexAgentSpec(spec, true)
}

func (a *RunCodexAgent) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunCodexAgentSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunCodexAgentSpec(spec, true); err != nil {
		return err
	}

	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return fmt.Errorf("failed to read OpenAI API key: %w", err)
	}
	if len(apiKey) == 0 {
		return fmt.Errorf("apiKey is required")
	}

	lastMessageFile, cleanup, err := createLastMessageFile()
	if err != nil {
		return err
	}
	defer cleanup()

	command := buildCodexAgentCommand(spec, string(apiKey), lastMessageFile)
	result, err := a.commandRunner().Run(context.Background(), command)
	if err != nil {
		return fmt.Errorf("failed to run Codex CLI: %w", err)
	}

	events := parseCodexJSONLEvents(result.Stdout)
	text := readCodexLastMessage(lastMessageFile)
	if text == "" {
		text = extractCodexText(events)
	}

	payload := CodexAgentPayload{
		Text:             text,
		ExitCode:         result.ExitCode,
		TimedOut:         result.TimedOut,
		Model:            spec.Model,
		Sandbox:          spec.Sandbox,
		WorkingDirectory: spec.WorkingDirectory,
		DurationMs:       result.Duration.Milliseconds(),
		Events:           events,
	}

	channel := CodexAgentOutputChannelSuccess
	if result.ExitCode != 0 || result.TimedOut {
		channel = CodexAgentOutputChannelFailed
		payload.Stdout = result.Stdout
		payload.Stderr = result.Stderr
	}

	return ctx.ExecutionState.Emit(channel, CodexAgentPayloadType, []any{payload})
}

func (a *RunCodexAgent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (a *RunCodexAgent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (a *RunCodexAgent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (a *RunCodexAgent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (a *RunCodexAgent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (a *RunCodexAgent) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (a *RunCodexAgent) commandRunner() agentcli.Runner {
	if a.runner != nil {
		return a.runner
	}
	return agentcli.OSRunner{}
}

func decodeRunCodexAgentSpec(raw any) (RunCodexAgentSpec, error) {
	var spec RunCodexAgentSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return RunCodexAgentSpec{}, fmt.Errorf("codex agent spec decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return RunCodexAgentSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Model = strings.TrimSpace(spec.Model)
	if spec.Model == "" {
		spec.Model = defaultCodexModel
	}
	spec.Prompt = strings.TrimSpace(spec.Prompt)
	spec.Sandbox = strings.TrimSpace(spec.Sandbox)
	if spec.Sandbox == "" {
		spec.Sandbox = defaultCodexSandbox
	}
	spec.WorkingDirectory = strings.TrimSpace(spec.WorkingDirectory)
	if spec.WorkingDirectory == "" {
		spec.WorkingDirectory = defaultCodexWorkingDirectory
	}
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = defaultCodexTimeoutSeconds
	}

	return spec, nil
}

func validateRunCodexAgentSpec(spec RunCodexAgentSpec, checkWorkingDirectory bool) error {
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if !slices.Contains(codexSandboxOptions, spec.Sandbox) {
		return fmt.Errorf("sandbox must be one of: %s", strings.Join(codexSandboxOptions, ", "))
	}
	if spec.TimeoutSeconds < 1 || spec.TimeoutSeconds > maxCodexTimeoutSeconds {
		return fmt.Errorf("timeoutSeconds must be between 1 and %d", maxCodexTimeoutSeconds)
	}
	if checkWorkingDirectory {
		if err := validateDirectory(spec.WorkingDirectory); err != nil {
			return fmt.Errorf("workingDirectory: %w", err)
		}
	}
	return nil
}

func buildCodexAgentCommand(spec RunCodexAgentSpec, apiKey string, lastMessageFile string) agentcli.Command {
	args := []string{
		"exec",
		"--json",
		"--ephemeral",
		"--skip-git-repo-check",
		"--sandbox",
		spec.Sandbox,
		"--cd",
		spec.WorkingDirectory,
		"--model",
		spec.Model,
		"--output-last-message",
		lastMessageFile,
		spec.Prompt,
	}

	return agentcli.Command{
		Name:    "codex",
		Args:    args,
		Dir:     spec.WorkingDirectory,
		Timeout: time.Duration(spec.TimeoutSeconds) * time.Second,
		Env: map[string]string{
			"CODEX_API_KEY":  apiKey,
			"OPENAI_API_KEY": apiKey,
		},
	}
}

func createLastMessageFile() (string, func(), error) {
	file, err := os.CreateTemp("", "superplane-codex-last-message-*.txt")
	if err != nil {
		return "", nil, fmt.Errorf("failed to create Codex output file: %w", err)
	}
	path := file.Name()
	if err := file.Close(); err != nil {
		_ = os.Remove(path)
		return "", nil, fmt.Errorf("failed to close Codex output file: %w", err)
	}

	return path, func() { _ = os.Remove(path) }, nil
}

func readCodexLastMessage(path string) string {
	if path == "" {
		return ""
	}
	content, err := os.ReadFile(filepath.Clean(path))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(content))
}

func parseCodexJSONLEvents(stdout string) []map[string]any {
	lines := strings.Split(stdout, "\n")
	events := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var event map[string]any
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			continue
		}
		events = append(events, event)
	}
	return events
}

func extractCodexText(events []map[string]any) string {
	for i := len(events) - 1; i >= 0; i-- {
		if text := extractTextValue(events[i]); text != "" {
			return text
		}
	}
	return ""
}

func extractTextValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := []string{}
		for _, item := range v {
			if text := extractTextValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "message", "content", "output", "result", "final_output", "last_message"} {
			if text := extractTextValue(v[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func validateDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("is required")
	}
	info, err := os.Stat(path)
	if err != nil {
		return err
	}
	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", path)
	}
	return nil
}

func intPtr(v int) *int {
	return &v
}
