package claude

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/agentcli"
)

const (
	ClaudeCodePayloadType = "claude.codeAgent.finished"

	ClaudeCodeOutputChannelSuccess = "success"
	ClaudeCodeOutputChannelFailed  = "failed"

	defaultClaudeCodeModel            = "sonnet"
	defaultClaudeCodePermissionMode   = "plan"
	defaultClaudeCodeWorkingDirectory = "/app"
	defaultClaudeCodeTimeoutSeconds   = 600
	defaultClaudeCodeMaxTurns         = 3
	maxClaudeCodeTimeoutSeconds       = 86400
	maxClaudeCodeTurns                = 100
)

var claudeCodePermissionModes = []string{"plan", "default", "acceptEdits", "auto", "dontAsk", "bypassPermissions"}

type RunClaudeCode struct {
	runner agentcli.Runner
}

type RunClaudeCodeSpec struct {
	Model            string `json:"model" mapstructure:"model"`
	Prompt           string `json:"prompt" mapstructure:"prompt"`
	PermissionMode   string `json:"permissionMode" mapstructure:"permissionMode"`
	WorkingDirectory string `json:"workingDirectory" mapstructure:"workingDirectory"`
	TimeoutSeconds   int    `json:"timeoutSeconds" mapstructure:"timeoutSeconds"`
	MaxTurns         int    `json:"maxTurns" mapstructure:"maxTurns"`
}

type ClaudeCodePayload struct {
	Text             string         `json:"text"`
	ExitCode         int            `json:"exitCode"`
	TimedOut         bool           `json:"timedOut"`
	IsError          bool           `json:"isError"`
	Model            string         `json:"model"`
	PermissionMode   string         `json:"permissionMode"`
	WorkingDirectory string         `json:"workingDirectory"`
	MaxTurns         int            `json:"maxTurns"`
	DurationMs       int64          `json:"durationMs"`
	Response         map[string]any `json:"response,omitempty"`
	Stdout           string         `json:"stdout,omitempty"`
	Stderr           string         `json:"stderr,omitempty"`
}

func (c *RunClaudeCode) Name() string {
	return "claude.runClaudeCode"
}

func (c *RunClaudeCode) Label() string {
	return "Run Claude Code"
}

func (c *RunClaudeCode) Description() string {
	return "Run Claude Code non-interactively on a local workspace"
}

func (c *RunClaudeCode) Documentation() string {
	return `The Run Claude Code component runs the Claude Code CLI in non-interactive print mode from the SuperPlane app container.

## Use Cases

- **Repository analysis**: Ask Claude Code to inspect code and summarize risks
- **Automated implementation tasks**: Run coding tasks from upstream workflow events
- **Code review**: Generate review feedback or remediation guidance
- **Local workflow automation**: Use SuperPlane events to start Claude Code tasks in a mounted workspace

## Configuration

- **Model**: Claude Code model alias or full model name. Defaults to ` + "`sonnet`" + `.
- **Prompt**: The task sent to Claude Code (supports expressions).
- **Permission Mode**: Claude Code permission mode. Defaults to plan mode for read-only behavior. Write-capable and bypass modes should only be used in trusted local/dev environments.
- **Working Directory**: Directory used as the Claude Code process working directory. Defaults to ` + "`/app`" + `.
- **Max Turns**: Maximum number of agentic turns in non-interactive mode. Defaults to 3.
- **Timeout**: Maximum runtime in seconds. Defaults to 600 seconds.

## Output

Routes to one of two channels:
- **success**: Claude Code exits with code 0 and does not report an error result
- **failed**: Claude Code exits non-zero, times out, or reports an error result

The payload includes the final Claude Code result text, exit code, timeout flag, selected model, permission mode, working directory, max turns, duration, parsed JSON response when available, and stderr/stdout for failures.

## Notes

- Requires the Claude Code CLI on the app container PATH.
- Requires a valid Claude API key configured on the Claude integration.
- The API key is passed only to the Claude Code subprocess and is never emitted in the payload.
- The component uses Claude Code ` + "`--bare`" + ` and ` + "`--no-session-persistence`" + ` modes to reduce local side effects.`
}

func (c *RunClaudeCode) Icon() string {
	return "claude"
}

func (c *RunClaudeCode) Color() string {
	return "#C9784D"
}

func (c *RunClaudeCode) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      ClaudeCodePayloadType,
		"timestamp": "2026-05-30T12:00:00Z",
		"data": map[string]any{
			"text":             "I inspected the requested files and drafted the implementation plan.",
			"exitCode":         0,
			"timedOut":         false,
			"isError":          false,
			"model":            defaultClaudeCodeModel,
			"permissionMode":   defaultClaudeCodePermissionMode,
			"workingDirectory": "/app",
			"maxTurns":         defaultClaudeCodeMaxTurns,
			"durationMs":       10180,
		},
	}
}

func (c *RunClaudeCode) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{
		{Name: ClaudeCodeOutputChannelSuccess, Label: "Success"},
		{Name: ClaudeCodeOutputChannelFailed, Label: "Failed"},
	}
}

func (c *RunClaudeCode) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    false,
			Default:     defaultClaudeCodeModel,
			Placeholder: defaultClaudeCodeModel,
			Description: "Claude model alias or full model name",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "model"},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Review the deployment failure in {{ previous().data.summary }}",
			Description: "Task sent to Claude Code. Supports expressions.",
		},
		{
			Name:        "permissionMode",
			Label:       "Permission Mode",
			Type:        configuration.FieldTypeSelect,
			Required:    false,
			Default:     defaultClaudeCodePermissionMode,
			Description: "Claude Code permission mode",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{Options: []configuration.FieldOption{
					{Label: "Plan", Value: "plan", Description: "Read-only planning mode."},
					{Label: "Default", Value: "default", Description: "Claude Code default permission behavior."},
					{Label: "Accept edits", Value: "acceptEdits", Description: "Allow Claude Code to apply edits."},
					{Label: "Auto", Value: "auto", Description: "Allow Claude Code to classify permissions automatically."},
					{Label: "Don't ask", Value: "dontAsk", Description: "Run without interactive permission prompts where supported."},
					{Label: "Bypass permissions", Value: "bypassPermissions", Description: "Dangerous: bypass permission checks for local/dev use only."},
				}},
			},
		},
		{
			Name:        "workingDirectory",
			Label:       "Working Directory",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Default:     defaultClaudeCodeWorkingDirectory,
			Placeholder: defaultClaudeCodeWorkingDirectory,
			Description: "Directory used as the Claude Code process working directory",
		},
		{
			Name:        "maxTurns",
			Label:       "Max Turns",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultClaudeCodeMaxTurns,
			Description: "Maximum number of agentic turns in non-interactive mode",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: claudeCodeIntPtr(1),
					Max: claudeCodeIntPtr(maxClaudeCodeTurns),
				},
			},
		},
		{
			Name:        "timeoutSeconds",
			Label:       "Timeout (seconds)",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     defaultClaudeCodeTimeoutSeconds,
			Description: "Maximum runtime in seconds",
			TypeOptions: &configuration.TypeOptions{
				Number: &configuration.NumberTypeOptions{
					Min: claudeCodeIntPtr(1),
					Max: claudeCodeIntPtr(maxClaudeCodeTimeoutSeconds),
				},
			},
		},
	}
}

func (c *RunClaudeCode) Setup(ctx core.SetupContext) error {
	spec, err := decodeRunClaudeCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	return validateRunClaudeCodeSpec(spec, true)
}

func (c *RunClaudeCode) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeRunClaudeCodeSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateRunClaudeCodeSpec(spec, true); err != nil {
		return err
	}

	apiKey, err := ctx.Integration.GetConfig("apiKey")
	if err != nil {
		return fmt.Errorf("failed to read Claude API key: %w", err)
	}
	if len(apiKey) == 0 {
		return fmt.Errorf("apiKey is required")
	}

	command := buildClaudeCodeCommand(spec, string(apiKey))
	result, err := c.commandRunner().Run(context.Background(), command)
	if err != nil {
		return fmt.Errorf("failed to run Claude Code CLI: %w", err)
	}

	response := parseClaudeCodeResponse(result.Stdout)
	text := extractClaudeCodeText(response)
	isError := claudeCodeResponseIsError(response)

	payload := ClaudeCodePayload{
		Text:             text,
		ExitCode:         result.ExitCode,
		TimedOut:         result.TimedOut,
		IsError:          isError,
		Model:            spec.Model,
		PermissionMode:   spec.PermissionMode,
		WorkingDirectory: spec.WorkingDirectory,
		MaxTurns:         spec.MaxTurns,
		DurationMs:       result.Duration.Milliseconds(),
		Response:         response,
	}

	channel := ClaudeCodeOutputChannelSuccess
	if result.ExitCode != 0 || result.TimedOut || isError {
		channel = ClaudeCodeOutputChannelFailed
		payload.Stdout = result.Stdout
		payload.Stderr = result.Stderr
	}

	return ctx.ExecutionState.Emit(channel, ClaudeCodePayloadType, []any{payload})
}

func (c *RunClaudeCode) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *RunClaudeCode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *RunClaudeCode) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *RunClaudeCode) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *RunClaudeCode) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *RunClaudeCode) HandleHook(ctx core.ActionHookContext) error {
	return nil
}

func (c *RunClaudeCode) commandRunner() agentcli.Runner {
	if c.runner != nil {
		return c.runner
	}
	return agentcli.OSRunner{}
}

func decodeRunClaudeCodeSpec(raw any) (RunClaudeCodeSpec, error) {
	var spec RunClaudeCodeSpec
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		Result:           &spec,
		WeaklyTypedInput: true,
		TagName:          "mapstructure",
	})
	if err != nil {
		return RunClaudeCodeSpec{}, fmt.Errorf("claude code spec decoder: %w", err)
	}
	if err := decoder.Decode(raw); err != nil {
		return RunClaudeCodeSpec{}, fmt.Errorf("failed to decode configuration: %w", err)
	}

	spec.Model = strings.TrimSpace(spec.Model)
	if spec.Model == "" {
		spec.Model = defaultClaudeCodeModel
	}
	spec.Prompt = strings.TrimSpace(spec.Prompt)
	spec.PermissionMode = strings.TrimSpace(spec.PermissionMode)
	if spec.PermissionMode == "" {
		spec.PermissionMode = defaultClaudeCodePermissionMode
	}
	spec.WorkingDirectory = strings.TrimSpace(spec.WorkingDirectory)
	if spec.WorkingDirectory == "" {
		spec.WorkingDirectory = defaultClaudeCodeWorkingDirectory
	}
	if spec.TimeoutSeconds <= 0 {
		spec.TimeoutSeconds = defaultClaudeCodeTimeoutSeconds
	}
	if spec.MaxTurns <= 0 {
		spec.MaxTurns = defaultClaudeCodeMaxTurns
	}

	return spec, nil
}

func validateRunClaudeCodeSpec(spec RunClaudeCodeSpec, checkWorkingDirectory bool) error {
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if !slices.Contains(claudeCodePermissionModes, spec.PermissionMode) {
		return fmt.Errorf("permissionMode must be one of: %s", strings.Join(claudeCodePermissionModes, ", "))
	}
	if spec.MaxTurns < 1 || spec.MaxTurns > maxClaudeCodeTurns {
		return fmt.Errorf("maxTurns must be between 1 and %d", maxClaudeCodeTurns)
	}
	if spec.TimeoutSeconds < 1 || spec.TimeoutSeconds > maxClaudeCodeTimeoutSeconds {
		return fmt.Errorf("timeoutSeconds must be between 1 and %d", maxClaudeCodeTimeoutSeconds)
	}
	if checkWorkingDirectory {
		if err := validateClaudeCodeDirectory(spec.WorkingDirectory); err != nil {
			return fmt.Errorf("workingDirectory: %w", err)
		}
	}
	return nil
}

func buildClaudeCodeCommand(spec RunClaudeCodeSpec, apiKey string) agentcli.Command {
	args := []string{
		"--bare",
		"-p",
		"--output-format",
		"json",
		"--no-session-persistence",
		"--permission-mode",
		spec.PermissionMode,
		"--model",
		spec.Model,
		"--max-turns",
		strconv.Itoa(spec.MaxTurns),
		spec.Prompt,
	}

	return agentcli.Command{
		Name:    "claude",
		Args:    args,
		Dir:     spec.WorkingDirectory,
		Timeout: time.Duration(spec.TimeoutSeconds) * time.Second,
		Env: map[string]string{
			"ANTHROPIC_API_KEY":   apiKey,
			"DISABLE_AUTOUPDATER": "1",
		},
	}
}

func parseClaudeCodeResponse(stdout string) map[string]any {
	trimmed := strings.TrimSpace(stdout)
	if trimmed == "" {
		return nil
	}

	var response map[string]any
	if err := json.Unmarshal([]byte(trimmed), &response); err == nil {
		return response
	}

	lines := strings.Split(trimmed, "\n")
	for i := len(lines) - 1; i >= 0; i-- {
		line := strings.TrimSpace(lines[i])
		if line == "" {
			continue
		}
		if err := json.Unmarshal([]byte(line), &response); err == nil {
			return response
		}
	}
	return nil
}

func extractClaudeCodeText(response map[string]any) string {
	if response == nil {
		return ""
	}
	for _, key := range []string{"result", "text", "message", "content", "output"} {
		if text := extractClaudeCodeTextValue(response[key]); text != "" {
			return text
		}
	}
	return ""
}

func extractClaudeCodeTextValue(value any) string {
	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v)
	case []any:
		parts := []string{}
		for _, item := range v {
			if text := extractClaudeCodeTextValue(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.Join(parts, "\n")
	case map[string]any:
		for _, key := range []string{"text", "result", "message", "content", "output"} {
			if text := extractClaudeCodeTextValue(v[key]); text != "" {
				return text
			}
		}
	}
	return ""
}

func claudeCodeResponseIsError(response map[string]any) bool {
	if response == nil {
		return false
	}
	if isError, ok := response["is_error"].(bool); ok {
		return isError
	}
	if subtype, ok := response["subtype"].(string); ok && strings.Contains(strings.ToLower(subtype), "error") {
		return true
	}
	if typ, ok := response["type"].(string); ok && strings.Contains(strings.ToLower(typ), "error") {
		return true
	}
	return false
}

func validateClaudeCodeDirectory(path string) error {
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

func claudeCodeIntPtr(v int) *int {
	return &v
}
