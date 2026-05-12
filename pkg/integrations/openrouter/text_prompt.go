package openrouter

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ResponsePayloadType = "openrouter.response"

type TextPrompt struct{}

type TextPromptSpec struct {
	Model       string  `mapstructure:"model"`
	Input       string  `mapstructure:"input"`
	Temperature float64 `mapstructure:"temperature"`
	MaxTokens   int     `mapstructure:"maxTokens"`
}

type ResponsePayload struct {
	ID     string `json:"id"`
	Model  string `json:"model"`
	Text   string `json:"text"`
	Usage  *Usage `json:"usage,omitempty"`
	Reason string `json:"reason,omitempty"`
}

func (c *TextPrompt) Name() string {
	return "openrouter.textPrompt"
}

func (c *TextPrompt) Label() string {
	return "Text Prompt"
}

func (c *TextPrompt) Description() string {
	return "Generate a text response using any model available on OpenRouter"
}

func (c *TextPrompt) Documentation() string {
	return `The Text Prompt component generates text responses using any model available on OpenRouter.

## Use Cases

- **Multi-provider AI**: Access models from Anthropic, OpenAI, Meta, Mistral, and many others through a single integration
- **Content generation**: Generate text content, summaries, or descriptions
- **Natural language processing**: Process and transform text using AI
- **Automated responses**: Generate responses to user queries or events
- **Model flexibility**: Easily switch between models without changing your workflow

## Configuration

- **Model**: Select the OpenRouter model to use (e.g., anthropic/claude-3.5-sonnet, openai/gpt-4o, meta-llama/llama-3-70b-instruct)
- **Prompt**: The text prompt to send to the model (supports expressions)
- **Temperature**: Controls randomness (0.0-2.0, default 0.7)
- **Max Tokens**: Maximum tokens in response (default 1024)

## Output

Returns the generated response including:
- **text**: The generated text response
- **model**: The model used for generation
- **usage**: Token usage information (prompt tokens, completion tokens, total tokens)
- **reason**: The finish reason (stop, length, etc.)
- **id**: Response ID for tracking

## Notes

- Requires a valid OpenRouter API key configured in the integration settings
- OpenRouter provides access to 100+ models from various providers
- Pricing and availability depend on the selected model
- Response quality and speed vary by model`
}

func (c *TextPrompt) Icon() string {
	return "globe"
}

func (c *TextPrompt) Color() string {
	return "purple"
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
			Default:     "anthropic/claude-3.5-sonnet",
			Placeholder: "e.g. anthropic/claude-3.5-sonnet",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "model",
				},
			},
		},
		{
			Name:        "input",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the prompt text",
			Description: "The prompt to send to the model (supports expressions)",
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     0.7,
			Description: "Controls randomness (0.0-2.0, lower is more deterministic)",
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Default:     1024,
			Description: "Maximum number of tokens in the response",
		},
	}
}

func (c *TextPrompt) Setup(ctx core.SetupContext) error {
	spec := TextPromptSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Input == "" {
		return fmt.Errorf("input is required")
	}

	return nil
}

func (c *TextPrompt) Execute(ctx core.ExecutionContext) error {
	spec := TextPromptSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Input == "" {
		return fmt.Errorf("input is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	chatReq := ChatCompletionRequest{
		Model: spec.Model,
		Messages: []ChatMessage{
			{
				Role:    "user",
				Content: spec.Input,
			},
		},
		Temperature: spec.Temperature,
		MaxTokens:   spec.MaxTokens,
	}

	if chatReq.MaxTokens == 0 {
		chatReq.MaxTokens = 1024
	}
	if chatReq.Temperature == 0 {
		chatReq.Temperature = 0.7
	}

	response, err := client.CreateChatCompletion(chatReq)
	if err != nil {
		return err
	}

	if len(response.Choices) == 0 {
		return fmt.Errorf("no response choices returned")
	}

	choice := response.Choices[0]
	payload := ResponsePayload{
		ID:     response.ID,
		Model:  response.Model,
		Text:   choice.Message.Content,
		Usage:  &response.Usage,
		Reason: choice.FinishReason,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ResponsePayloadType,
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

func (c *TextPrompt) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *TextPrompt) Cleanup(ctx core.SetupContext) error {
	return nil
}

func extractResponseText(response *ChatCompletionResponse) string {
	if response == nil {
		return ""
	}

	if len(response.Choices) == 0 {
		return ""
	}

	return strings.TrimSpace(response.Choices[0].Message.Content)
}
