package atlascloud

import (
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ResponsePayloadType = "atlascloud.response"

type CreateResponse struct{}

type CreateResponseSpec struct {
	Model string `json:"model"`
	Input string `json:"input"`
}

type ResponsePayload struct {
	ID       string         `json:"id"`
	Model    string         `json:"model"`
	Text     string         `json:"text"`
	Usage    *ResponseUsage `json:"usage,omitempty"`
	Response *Response      `json:"response"`
}

func (c *CreateResponse) Name() string {
	return "atlascloud.textPrompt"
}

func (c *CreateResponse) Label() string {
	return "Text Prompt"
}

func (c *CreateResponse) Description() string {
	return "Generate a text response using Atlas Cloud"
}

func (c *CreateResponse) Documentation() string {
	return `The Text Prompt component generates text responses using Atlas Cloud's language models.

## Use Cases

- **Content generation**: Generate text content, summaries, or descriptions
- **Natural language processing**: Process and transform text using AI
- **Automated responses**: Generate responses to user queries or events
- **Data transformation**: Convert structured data into natural language

## Configuration

- **Model**: Select the Atlas Cloud model to use (e.g., deepseek-ai/DeepSeek-V3-0324)
- **Prompt**: The text prompt to send to the model (supports expressions)

## Output

Returns the generated response including:
- **text**: The generated text response
- **model**: The model used for generation
- **usage**: Token usage information (input tokens, output tokens, total tokens)
- **id**: Response ID for tracking

## Notes

- Requires a valid Atlas Cloud API key configured in the application settings
- Atlas Cloud exposes a single OpenAI-compatible API for 300+ models across LLM, image, and video modalities
- Response quality and speed depend on the selected model
- Token usage is tracked and may incur costs based on your Atlas Cloud plan`
}

func (c *CreateResponse) Icon() string {
	return "sparkles"
}

func (c *CreateResponse) Color() string {
	return "gray"
}

func (c *CreateResponse) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *CreateResponse) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Default:     "deepseek-ai/DeepSeek-V3-0324",
			Placeholder: "e.g. deepseek-ai/DeepSeek-V3-0324",
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
		},
	}
}

func (c *CreateResponse) Setup(ctx core.SetupContext) error {
	spec := CreateResponseSpec{}
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

func (c *CreateResponse) Execute(ctx core.ExecutionContext) error {
	spec := CreateResponseSpec{}
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

	response, err := client.CreateResponse(spec.Model, spec.Input)
	if err != nil {
		return err
	}

	text := extractResponseText(response)
	payload := ResponsePayload{
		ID:       response.ID,
		Model:    response.Model,
		Text:     text,
		Usage:    response.Usage,
		Response: response,
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ResponsePayloadType,
		[]any{payload},
	)
}

func (c *CreateResponse) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *CreateResponse) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *CreateResponse) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func extractResponseText(response *Response) string {
	if response == nil {
		return ""
	}

	if response.OutputText != "" {
		return response.OutputText
	}

	var builder strings.Builder
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type != "" && content.Type != "output_text" && content.Type != "text" {
				continue
			}

			if content.Text == "" {
				continue
			}

			if builder.Len() > 0 {
				builder.WriteString("\n")
			}
			builder.WriteString(content.Text)
		}
	}

	return builder.String()
}

func (c *CreateResponse) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateResponse) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateResponse) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
