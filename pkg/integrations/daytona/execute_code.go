package daytona

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ExecuteCodePayloadType = "daytona.execute.response"

type ExecuteCode struct{}

type ExecuteCodeSpec struct {
	SandboxID string `json:"sandboxId"`
	Code      string `json:"code"`
	Language  string `json:"language"`
	Timeout   int    `json:"timeout,omitempty"`
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

	req := &ExecuteCodeRequest{
		Code:     spec.Code,
		Language: spec.Language,
		Timeout:  spec.Timeout,
	}

	response, err := client.ExecuteCode(spec.SandboxID, req)
	if err != nil {
		return fmt.Errorf("failed to execute code: %v", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ExecuteCodePayloadType,
		[]any{response},
	)
}

func (e *ExecuteCode) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (e *ExecuteCode) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (e *ExecuteCode) Actions() []core.Action {
	return []core.Action{}
}

func (e *ExecuteCode) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (e *ExecuteCode) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}
