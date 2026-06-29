package openai

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const ResponsePayloadType = "openai.response"

type CreateResponse struct{}

type CreateResponseSpec struct {
	Model        string `json:"model"`
	Input        string `json:"input"`
	OutputSchema any    `json:"outputSchema"`
}

type ResponsePayload struct {
	ID       string          `json:"id"`
	Model    string          `json:"model"`
	Text     string          `json:"text"`
	Parsed   any             `json:"parsed,omitempty"`
	Usage    *ResponseUsage  `json:"usage,omitempty"`
	Response *OpenAIResponse `json:"response"`
}

// ResponseNodeMetadata is node-level metadata surfaced in the UI so the
// configured model and options are visible on the node without opening it.
type ResponseNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

func (c *CreateResponse) Name() string {
	return "openai.textPrompt"
}

func (c *CreateResponse) Label() string {
	return "Text Prompt"
}

func (c *CreateResponse) Description() string {
	return "Generate a text response using OpenAI"
}

func (c *CreateResponse) Documentation() string {
	return `The Text Prompt component generates text responses using OpenAI's language models.

## Use Cases

- **Content generation**: Generate text content, summaries, or descriptions
- **Natural language processing**: Process and transform text using AI
- **Automated responses**: Generate responses to user queries or events
- **Data transformation**: Convert structured data into natural language

## Configuration

- **Model**: Select the OpenAI model to use (e.g., gpt-4, gpt-3.5-turbo)
- **Prompt**: The text prompt to send to the model (supports expressions)
- **Structured Output Schema**: (Optional) A JSON Schema that constrains the response to JSON, returned on the parsed output. OpenAI strict mode requires the root to be an object, every object to set additionalProperties:false, and every property to be listed in required (use a nullable type for optional fields).

## Output

Returns the generated response including:
- **text**: The generated text response
- **model**: The model used for generation
- **usage**: Token usage information (prompt tokens, completion tokens, total tokens)
- **id**: Response ID for tracking
- **parsed**: When a Structured Output Schema is set, the response parsed into an object.

## Notes

- Requires a valid OpenAI API key configured in the application settings
- Response quality and speed depend on the selected model
- Token usage is tracked and may incur costs based on your OpenAI plan
- Supports OpenAI-compatible providers by setting a custom Base URL in the integration settings (e.g., Azure OpenAI, Ollama, vLLM). Note: structured output uses the OpenAI Responses API text.format parameter and may not be supported by all compatible providers.`
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
			Default:     "gpt-5.2",
			Placeholder: "e.g. gpt-5.2",
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
		{
			Name:        "outputSchema",
			Label:       "Structured Output Schema",
			Type:        configuration.FieldTypeObject,
			Required:    false,
			Togglable:   true,
			Default:     map[string]any{},
			Description: "JSON Schema for the response, returned on the `parsed` output field. OpenAI strict mode requires the root to be an object, every object to set `additionalProperties: false`, and every property to be listed in `required` (use a nullable type like [\"string\",\"null\"] for optional fields).",
			Placeholder: `{"type":"object","properties":{},"required":[],"additionalProperties":false}`,
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

	schema, err := decodeOutputSchema(spec.OutputSchema)
	if err != nil {
		return err
	}
	if schema != nil {
		if err := validateStrictJSONSchema(schema, ""); err != nil {
			return err
		}
	}

	if ctx.Metadata != nil {
		_ = ctx.Metadata.Set(ResponseNodeMetadata{
			Model:            spec.Model,
			StructuredOutput: schema != nil,
		})
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

	schema, err := decodeOutputSchema(spec.OutputSchema)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	req := CreateResponseRequest{Model: spec.Model, Input: spec.Input}
	if schema != nil {
		req.Text = &ResponseTextConfig{
			Format: &ResponseFormat{
				Type:   "json_schema",
				Name:   "structured_output",
				Schema: schema,
				Strict: true,
			},
		}
	}

	response, err := client.CreateResponse(req)
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

	// When a schema is configured, surface a refusal as text (it arrives on a
	// dedicated content item that extractResponseText skips) and otherwise parse
	// the JSON response into a structured object.
	if schema != nil {
		if refusal := extractRefusal(response); refusal != "" {
			if payload.Text == "" {
				payload.Text = refusal
			}
		} else if text != "" {
			var parsed any
			if err := json.Unmarshal([]byte(text), &parsed); err == nil {
				payload.Parsed = parsed
			}
		}
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

func extractResponseText(response *OpenAIResponse) string {
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

// extractRefusal returns the refusal message from a Responses API output, if any.
// Refusals arrive as a dedicated content item (type "refusal") rather than JSON.
func extractRefusal(response *OpenAIResponse) string {
	if response == nil {
		return ""
	}
	for _, output := range response.Output {
		for _, content := range output.Content {
			if content.Type == "refusal" && content.Refusal != "" {
				return content.Refusal
			}
		}
	}
	return ""
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
