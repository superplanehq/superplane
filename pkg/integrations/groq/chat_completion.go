package groq

import (
	"fmt"
	"net/http"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/models"
)

const ChatCompletionPayloadType = "groq.chatCompletion.result"

type ChatCompletion struct{}

type ChatCompletionSpec struct {
	Model        string  `json:"model" mapstructure:"model"`
	Input        string  `json:"input" mapstructure:"input"`
	SystemPrompt *string `json:"systemPrompt,omitempty" mapstructure:"systemPrompt"`
}

type ChatCompletionPayload struct {
	ID       string                  `json:"id"`
	Model    string                  `json:"model"`
	Text     string                  `json:"text"`
	Usage    *ChatCompletionUsage    `json:"usage,omitempty"`
	Response *ChatCompletionResponse `json:"response"`
}

type ChatCompletionNodeMetadata struct {
	Model string `json:"model" mapstructure:"model"`
}

func (c *ChatCompletion) Name() string {
	return "groq.chatCompletion"
}

func (c *ChatCompletion) Label() string {
	return "Chat Completion"
}

func (c *ChatCompletion) Description() string {
	return "Generate a response with a Groq-hosted chat model"
}

func (c *ChatCompletion) Documentation() string {
	return `The Chat Completion component sends a prompt to a selected Groq model and returns its text response.

## Use Cases

- **Content generation**: Generate text for a workflow step
- **Summarization**: Summarize input from an earlier workflow step
- **Classification**: Ask a model to classify text or select an outcome
- **Data transformation**: Convert input into a different text format

## Configuration

- **Model**: Select an active Groq-hosted chat model. SuperPlane filters known non-chat models.
- **Input**: Enter the user message to send to the model. This field supports expressions.
- **System Prompt**: Add optional instructions. SuperPlane sends these instructions before the user message.

## Output

The component returns:

- **text**: The generated text response
- **model**: The model used for the response
- **usage**: Prompt, completion, and total token counts
- **id**: The Groq response ID
- **response**: The chat completion response

## Notes

- Requires a valid Groq API key in the Groq integration.
- Available models depend on your Groq account and Groq's current model catalog.
- Token usage is recorded for the run when Groq returns usage data.`
}

func (c *ChatCompletion) Icon() string {
	return "message-square"
}

func (c *ChatCompletion) Color() string {
	return "orange"
}

func (c *ChatCompletion) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ChatCompletion) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Placeholder: "openai/gpt-oss-120b",
			Description: "Select a Groq model",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{Type: "model"},
			},
		},
		{
			Name:        "input",
			Label:       "Input",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the prompt",
			Description: "The user message (supports expressions)",
		},
		{
			Name:        "systemPrompt",
			Label:       "System Prompt",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Placeholder: "Optional system-level prompt",
			Description: "System-level instructions sent before the user prompt",
		},
	}
}

func (c *ChatCompletion) Setup(ctx core.SetupContext) error {
	spec, err := decodeChatCompletionSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateChatCompletionSpec(spec); err != nil {
		return err
	}

	if ctx.Metadata != nil {
		_ = ctx.Metadata.Set(ChatCompletionNodeMetadata{Model: spec.Model})
	}

	return nil
}

func (c *ChatCompletion) Execute(ctx core.ExecutionContext) error {
	spec, err := decodeChatCompletionSpec(ctx.Configuration)
	if err != nil {
		return err
	}
	if err := validateChatCompletionSpec(spec); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	response, err := client.CreateChatCompletion(ChatCompletionRequest{
		Model:    spec.Model,
		Messages: buildMessages(spec.SystemPrompt, spec.Input),
	})
	if err != nil {
		return err
	}

	payload, err := buildChatCompletionPayload(response)
	if err != nil {
		return err
	}

	recordGroqUsage(ctx, response)
	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ChatCompletionPayloadType,
		[]any{*payload},
	)
}

func decodeChatCompletionSpec(configuration any) (ChatCompletionSpec, error) {
	spec := ChatCompletionSpec{}
	if err := mapstructure.Decode(configuration, &spec); err != nil {
		return ChatCompletionSpec{}, fmt.Errorf("failed to decode Groq chat completion configuration: %w", err)
	}
	return spec, nil
}

func validateChatCompletionSpec(spec ChatCompletionSpec) error {
	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}
	if spec.Input == "" {
		return fmt.Errorf("input is required")
	}
	return nil
}

func buildMessages(systemPrompt *string, input string) []ChatMessage {
	messages := make([]ChatMessage, 0, 2)
	if systemPrompt != nil && *systemPrompt != "" {
		messages = append(messages, ChatMessage{Role: "system", Content: *systemPrompt})
	}
	return append(messages, ChatMessage{Role: "user", Content: input})
}

func buildChatCompletionPayload(response *ChatCompletionResponse) (*ChatCompletionPayload, error) {
	if response == nil || len(response.Choices) == 0 {
		return nil, fmt.Errorf("Groq chat completion returned no choices")
	}
	choice := response.Choices[0]
	return &ChatCompletionPayload{
		ID:       response.ID,
		Model:    response.Model,
		Text:     choice.Message.Content,
		Usage:    response.Usage,
		Response: response,
	}, nil
}

func recordGroqUsage(ctx core.ExecutionContext, response *ChatCompletionResponse) {
	if response == nil || response.Usage == nil {
		return
	}
	inputTokens := int64(response.Usage.PromptTokens)
	outputTokens := int64(response.Usage.CompletionTokens)
	totalTokens := int64(response.Usage.TotalTokens)
	if totalTokens == 0 {
		totalTokens = inputTokens + outputTokens
	}
	ctx.RecordUsageBestEffort(core.UsageRecord{
		Provider:     models.UsageProviderGroq,
		Model:        response.Model,
		InputTokens:  inputTokens,
		OutputTokens: outputTokens,
		TotalTokens:  totalTokens,
	})
}

func (c *ChatCompletion) ExampleOutput() map[string]any {
	return map[string]any{
		"type":      ChatCompletionPayloadType,
		"timestamp": "2026-08-25T12:00:00Z",
		"data": map[string]any{
			"id":    "chatcmpl-example",
			"model": "llama-3.3-70b-versatile",
			"text":  "Response from the model",
			"usage": map[string]any{
				"prompt_tokens":     10,
				"completion_tokens": 10,
				"total_tokens":      20,
			},
			"response": map[string]any{
				"id":      "chatcmpl-example",
				"object":  "chat.completion",
				"created": 1756123200,
				"model":   "llama-3.3-70b-versatile",
				"choices": []map[string]any{{
					"index": 0,
					"message": map[string]any{
						"role":    "assistant",
						"content": "Response from the model",
					},
					"finish_reason": "stop",
				}},
				"usage": map[string]any{
					"prompt_tokens":     10,
					"completion_tokens": 10,
					"total_tokens":      20,
				},
			},
		},
	}
}

func (c *ChatCompletion) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *ChatCompletion) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *ChatCompletion) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *ChatCompletion) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *ChatCompletion) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
