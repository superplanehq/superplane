package daytona

import (
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	ExecuteCodePayloadType  = "daytona.execute.response"
	ExecuteCodePollInterval = 5 * time.Second
)

type ExecuteCode struct{}

type ExecuteCodeSpec struct {
	SandboxID string `json:"sandboxId"`
	Code      string `json:"code"`
	Language  string `json:"language"`
	Timeout   int    `json:"timeout,omitempty"`
}

type ExecuteCodeMetadata struct {
	SandboxID string    `json:"sandboxId" mapstructure:"sandboxId"`
	SessionID string    `json:"sessionId" mapstructure:"sessionId"`
	CmdID     string    `json:"cmdId" mapstructure:"cmdId"`
	StartedAt time.Time `json:"startedAt" mapstructure:"startedAt"`
	Timeout   int       `json:"timeout" mapstructure:"timeout"`
}

func (e *ExecuteCode) Name() string {
	return "daytona.executeCode"
}

func (e *ExecuteCode) Label() string {
	return "Execute Code"
}

func (e *ExecuteCode) Description() string {
	return "Execute code in a sandbox environment"
}

func (e *ExecuteCode) Documentation() string {
	return `The Execute Code component runs code in an existing Daytona sandbox.

## Use Cases

- **AI code execution**: Run AI-generated code safely
- **Code testing**: Execute untrusted code in isolation
- **Script automation**: Run Python, TypeScript, or JavaScript scripts
- **Data processing**: Execute data transformation scripts

## Configuration

- **Sandbox ID**: The ID of the sandbox (from createSandbox output)
- **Code**: The code to execute (supports expressions)
- **Language**: The programming language (python, typescript, javascript)
- **Timeout**: Optional execution timeout in milliseconds

## Output

Returns the execution result including:
- **exitCode**: The process exit code (0 for success)
- **result**: The stdout/output from the code execution

## Notes

- The sandbox must be created first using createSandbox
- Code output is captured from stdout
- Non-zero exit codes indicate execution errors`
}

func (e *ExecuteCode) Icon() string {
	return "daytona"
}

func (e *ExecuteCode) Color() string {
	return "orange"
}

func (e *ExecuteCode) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (e *ExecuteCode) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "sandboxId",
			Label:       "Sandbox ID",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "The ID of the sandbox to execute code in",
			Placeholder: `{{ $["daytona.createSandbox"].data.id }}`,
		},
		{
			Name:        "code",
			Label:       "Code",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Description: "The code to execute",
			Placeholder: "print('Hello, World!')",
		},
		{
			Name:     "language",
			Label:    "Language",
			Type:     configuration.FieldTypeSelect,
			Required: true,
			Default:  "python",
			TypeOptions: &configuration.TypeOptions{
				Select: &configuration.SelectTypeOptions{
					Options: []configuration.FieldOption{
						{Label: "Python", Value: "python"},
						{Label: "TypeScript", Value: "typescript"},
						{Label: "JavaScript", Value: "javascript"},
					},
				},
			},
			Description: "The programming language of the code",
		},
		{
			Name:        "timeout",
			Label:       "Timeout",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Description: "Execution timeout in seconds",
			Default:     30,
		},
	}
}

func (e *ExecuteCode) Setup(ctx core.SetupContext) error {
	spec := ExecuteCodeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.SandboxID == "" {
		return fmt.Errorf("sandboxId is required")
	}

	if spec.Code == "" {
		return fmt.Errorf("code is required")
	}

	if spec.Language == "" {
		return fmt.Errorf("language is required")
	}

	validLanguages := map[string]bool{
		"python":     true,
		"typescript": true,
		"javascript": true,
	}
	if !validLanguages[spec.Language] {
		return fmt.Errorf("invalid language: %s (must be python, typescript, or javascript)", spec.Language)
	}

	return nil
}

func (e *ExecuteCode) Execute(ctx core.ExecutionContext) error {
	spec := ExecuteCodeSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return fmt.Errorf("failed to create client: %v", err)
	}

	var command string
	switch spec.Language {
	case "python":
		command = fmt.Sprintf("python3 -c %q", spec.Code)
	case "javascript":
		command = fmt.Sprintf("node -e %q", spec.Code)
	case "typescript":
		command = fmt.Sprintf("npx ts-node -e %q", spec.Code)
	default:
		command = fmt.Sprintf("python3 -c %q", spec.Code)
	}

	sessionID := uuid.New().String()
	if err := client.CreateSession(spec.SandboxID, sessionID); err != nil {
		return fmt.Errorf("failed to create session: %v", err)
	}

	response, err := client.ExecuteSessionCommand(spec.SandboxID, sessionID, command)
	if err != nil {
		return fmt.Errorf("failed to execute code: %v", err)
	}

	timeout := spec.Timeout
	if timeout == 0 {
		timeout = 30
	}

	metadata := ExecuteCodeMetadata{
		SandboxID: spec.SandboxID,
		SessionID: sessionID,
		CmdID:     response.CmdID,
		StartedAt: time.Now(),
		Timeout:   timeout,
	}

	if err := ctx.Metadata.Set(metadata); err != nil {
		return err
	}

	return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCodePollInterval)
}

func (e *ExecuteCode) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *ExecuteCode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExecuteCode) Actions() []core.Action {
	return []core.Action{
		{Name: "poll", UserAccessible: false},
	}
}

func (e *ExecuteCode) HandleAction(ctx core.ActionContext) error {
	if ctx.Name == "poll" {
		return e.poll(ctx)
	}
	return fmt.Errorf("unknown action: %s", ctx.Name)
}

func (e *ExecuteCode) poll(ctx core.ActionContext) error {
	if ctx.ExecutionState.IsFinished() {
		return nil
	}

	var metadata ExecuteCodeMetadata
	if err := mapstructure.Decode(ctx.Metadata.Get(), &metadata); err != nil {
		return fmt.Errorf("failed to decode metadata: %v", err)
	}

	if time.Since(metadata.StartedAt) > time.Duration(metadata.Timeout)*time.Second {
		return fmt.Errorf("code execution timed out after %d seconds", metadata.Timeout)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	session, err := client.GetSession(metadata.SandboxID, metadata.SessionID)
	if err != nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCodePollInterval)
	}

	cmd := session.FindCommand(metadata.CmdID)
	if cmd == nil || cmd.ExitCode == nil {
		return ctx.Requests.ScheduleActionCall("poll", map[string]any{}, ExecuteCodePollInterval)
	}

	logs, err := client.GetSessionCommandLogs(metadata.SandboxID, metadata.SessionID, metadata.CmdID)
	if err != nil {
		logs = ""
	}

	result := &ExecuteCodeResponse{
		ExitCode: *cmd.ExitCode,
		Result:   logs,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ExecuteCodePayloadType,
		[]any{result},
	)
}

func (e *ExecuteCode) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (e *ExecuteCode) Cleanup(ctx core.SetupContext) error {
	return nil
}
