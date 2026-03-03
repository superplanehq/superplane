package openrouter

import (
	"fmt"
	"net/http"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const MessagePayloadType = "openrouter.message"

type TextPrompt struct{}

type TextPromptSpec struct {
	Model         string   `json:"model"`
	Prompt        string   `json:"prompt"`
	SystemMessage string   `json:"systemMessage"`
	MaxTokens     int      `json:"maxTokens"`
	Temperature   *float64 `json:"temperature"`
}

type MessagePayload struct {
	ID           string                   `json:"id"`
	Model        string                   `json:"model"`
	Text         string                   `json:"text"`
	Usage        *ResponseUsage           `json:"usage,omitempty"`
	FinishReason string                   `json:"finishReason,omitempty"`
	Response     *ChatCompletionsResponse `json:"response,omitempty"`
}

func (c *TextPrompt) Name() string {
	return "openrouter.textPrompt"
}

func (c *TextPrompt) Label() string {
	return "Text Prompt"
}

func (c *TextPrompt) Description() string {
	return "Generate a response using OpenRouter's chat completions API (any supported model)"
}

func (c *TextPrompt) Documentation() string {
	return `The Text Prompt component uses OpenRouter's unified API to generate text with many models (OpenAI, Anthropic, Google, etc.).

## Use Cases

- **Summarization**: Summarize incidents, logs, or deployments.
- **Code analysis**: Code review or PR comments.
- **Content generation**: Documentation, communications, or drafts.
- **Model flexibility**: Switch models (e.g. openai/gpt-4o, anthropic/claude-3.5-sonnet) without changing integration.

## Configuration

- **Model**: The model ID (e.g. openai/gpt-4o, anthropic/claude-3.5-sonnet). Use the model selector for available models.
- **Prompt**: The main user message.
- **System Message**: (Optional) Context or system instruction.
- **Max Tokens**: (Optional) Maximum tokens to generate.
- **Temperature**: (Optional) Randomness (0 to 2).

## Output

Returns:
- **text**: Generated content.
- **usage**: Prompt and completion token counts.
- **finishReason**: Why generation stopped (e.g. stop, length).
- **model**: Model used (may differ if fallback occurred).

## Notes

- Requires a valid OpenRouter API key.
- Model IDs include provider prefix (e.g. openai/gpt-4o). See [openrouter.ai/models](https://openrouter.ai/models).
`
}

func (c *TextPrompt) Icon() string {
	return "message-square"
}

func (c *TextPrompt) Color() string {
	return "blue"
}

func (c *TextPrompt) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *TextPrompt) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "openai/gpt-4o",
			Placeholder: "Select a model",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the user prompt",
			Description: "The main instruction or question",
		},
		{
			Name:        "systemMessage",
			Label:       "System Message",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Placeholder: "e.g. You are a concise DevOps assistant",
			Description: "Optional context or behavior",
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "1024",
			Description: "Maximum tokens to generate",
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     "1.0",
			Description: "Randomness (0 to 2)",
		},
	}
}

func (c *TextPrompt) Setup(ctx core.SetupContext) error {
	spec := TextPromptSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	return nil
}

func (c *TextPrompt) Execute(ctx core.ExecutionContext) error {
	spec := TextPromptSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}
	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}
	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}
	if spec.MaxTokens == 0 {
		spec.MaxTokens = 1024
	}
	if spec.MaxTokens < 1 {
		return fmt.Errorf("maxTokens must be at least 1")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	messages := []Message{}
	if spec.SystemMessage != "" {
		messages = append(messages, Message{Role: "system", Content: spec.SystemMessage})
	}
	messages = append(messages, Message{Role: "user", Content: spec.Prompt})

	req := ChatCompletionsRequest{
		Model:       spec.Model,
		Messages:    messages,
		MaxTokens:   spec.MaxTokens,
		Temperature: spec.Temperature,
	}

	resp, err := client.ChatCompletions(req)
	if err != nil {
		return err
	}

	text := extractMessageText(resp)
	finishReason := ""
	if len(resp.Choices) > 0 {
		finishReason = resp.Choices[0].FinishReason
	}

	payload := MessagePayload{
		ID:           resp.ID,
		Model:        resp.Model,
		Text:         text,
		Usage:        resp.Usage,
		FinishReason: finishReason,
		Response:     resp,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		MessagePayloadType,
		[]any{payload},
	)
}

func (c *TextPrompt) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *TextPrompt) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *TextPrompt) Actions() []core.Action {
	return []core.Action{}
}

func (c *TextPrompt) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (c *TextPrompt) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return http.StatusOK, nil
}

func (c *TextPrompt) Cleanup(ctx core.SetupContext) error {
	return nil
}

func extractMessageText(resp *ChatCompletionsResponse) string {
	if resp == nil || len(resp.Choices) == 0 {
		return ""
	}
	return resp.Choices[0].Message.Content
}
