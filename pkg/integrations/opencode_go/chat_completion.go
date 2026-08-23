package opencodego

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/attachments"
	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

const ChatCompletionPayloadType = "opencodego.chatCompletion.result"

// maxAttachmentBytes caps the combined size of attachments on one request.
// OpenCode Go has no Files API, so attachments are inlined into the request
// body as base64 data URLs — roughly a third larger again on the wire.
const maxAttachmentBytes = 8 * 1024 * 1024

type ChatCompletion struct{}

type ChatCompletionSpec struct {
	Model        string   `json:"model" mapstructure:"model"`
	Prompt       string   `json:"prompt" mapstructure:"prompt"`
	SystemPrompt string   `json:"systemPrompt" mapstructure:"systemPrompt"`
	Files        []string `json:"files" mapstructure:"files"`
	MaxTokens    *int     `json:"maxTokens" mapstructure:"maxTokens"`
	Temperature  *float64 `json:"temperature" mapstructure:"temperature"`
	OutputSchema string   `json:"outputSchema" mapstructure:"outputSchema"`
}

type ChatCompletionPayload struct {
	ID           string `json:"id"`
	Model        string `json:"model"`
	Text         string `json:"text"`
	Parsed       any    `json:"parsed,omitempty"`
	FinishReason string `json:"finishReason"`
	Usage        *Usage `json:"usage,omitempty"`
	Response     any    `json:"response"`
}

// ChatCompletionNodeMetadata is node-level metadata surfaced in the UI so the
// configured model is visible without opening the node.
type ChatCompletionNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
}

func (c *ChatCompletion) Name() string {
	return "opencodego.chatCompletion"
}

func (c *ChatCompletion) Label() string {
	return "Chat Completion"
}

func (c *ChatCompletion) Description() string {
	return "Generate a response with a curated open coding model on OpenCode Go"
}

func (c *ChatCompletion) Documentation() string {
	return `The Chat Completion component sends a prompt to an OpenCode Go model and returns the generated response.

## Use Cases

- **Workflow automation**: Run prompts on flat-rate open coding models from a workflow
- **Classification**: Sort incoming events into categories before a human reads them
- **Triage**: Read reports and assign a priority
- **Enrichment**: Add summaries or labels to data as it moves through the workflow
- **Document analysis**: Attach images, PDFs, or text files from the Files tab alongside the prompt

## Configuration

- **Model**: The model to prompt. Picked from the models OpenCode Go currently lists.
- **Prompt**: The user message (supports expressions)
- **System Prompt**: (Optional) System-level instructions sent ahead of the prompt
- **Files**: (Optional) Files from the Files tab to attach. Text files are added to the prompt directly. Images require a vision-capable model.
- **Max Tokens**: (Optional) Upper bound on generated tokens
- **Temperature**: (Optional) Sampling temperature
- **Structured Output**: (Optional) A JSON Schema for the response. The reply is requested in strict JSON mode and is also available parsed on the ` + "`parsed`" + ` output. Support depends on the model. Models without support ignore this setting and return plain text.

## Output

Returns the completion including:
- **text**: The generated text
- **parsed**: When Structured Output is configured, the response parsed into an object
- **model**: The model that served the request
- **finishReason**: Why generation stopped
- **usage**: Token counts for the request
- **response**: The full API response

## Notes

	- The action selects the correct endpoint for the selected model: Chat Completions for GLM, Kimi, DeepSeek, MiMo, Hy3, and Ox Alpha Free; Messages for MiniMax and Qwen; and Responses for Grok 4.5, GPT 5.6 Luna, and Muse Spark 1.2 Contributor.
- Attachments are inlined into the request body rather than uploaded, so the combined size is capped at 8MB.
- Some models need an opt-in in your OpenCode Go workspace settings (China region or training data policy). See the integration instructions.
- Only PDFs and images are sent as attachments. Text files become part of the prompt, so they cost prompt tokens only.
- The schema is validated before the request and sent in strict mode. Strict mode marks every property required, so express optional fields by making their type nullable.
`
}

func (c *ChatCompletion) Icon() string {
	return "sparkles"
}

func (c *ChatCompletion) Color() string {
	return "gray"
}

func (c *ChatCompletion) OutputChannels(config any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *ChatCompletion) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "model",
			Label:       "Model",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "Model to prompt",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: ResourceTypeModel,
				},
			},
		},
		{
			Name:        "prompt",
			Label:       "Prompt",
			Type:        configuration.FieldTypeText,
			Required:    true,
			Placeholder: "Enter the prompt text",
			Description: "The user message (supports expressions)",
		},
		{
			Name:        "systemPrompt",
			Label:       "System Prompt",
			Type:        configuration.FieldTypeText,
			Required:    false,
			Togglable:   true,
			Description: "System-level instructions sent ahead of the prompt",
		},
		{
			Name:        "files",
			Label:       "Files",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Togglable:   true,
			Description: "Files from the Files tab to attach to the prompt (images, PDFs, or text). Images require a vision-capable model.",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File path",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeRepositoryFile,
					},
				},
			},
		},
		{
			Name:        "maxTokens",
			Label:       "Max Tokens",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Upper bound on generated tokens",
		},
		{
			Name:        "temperature",
			Label:       "Temperature",
			Type:        configuration.FieldTypeNumber,
			Required:    false,
			Togglable:   true,
			Description: "Sampling temperature",
		},
		structuredoutput.ConfigField(
			"outputSchema",
			"Structured Output",
			"A JSON Schema describing the response. Support depends on the model. Models without support ignore this setting and return plain text.",
		),
	}
}

func (c *ChatCompletion) Setup(ctx core.SetupContext) error {
	spec := ChatCompletionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if err := validateTemperature(spec.Temperature); err != nil {
		return err
	}

	if len(spec.Files) > 0 {
		if ctx.Files == nil {
			return fmt.Errorf("files configured but file access is not available")
		}
		available, err := ctx.Files.List()
		if err != nil {
			return fmt.Errorf("failed to list repository files: %v", err)
		}
		fileSet := make(map[string]bool, len(available))
		for _, f := range available {
			if norm, err := gitprovider.NormalizePath(f); err == nil {
				fileSet[norm] = true
			}
		}
		for _, f := range spec.Files {
			norm, err := gitprovider.ValidateUserPath(f)
			if err != nil {
				return fmt.Errorf("invalid file path %q: %v", f, err)
			}
			if !fileSet[norm] {
				return fmt.Errorf("file %q not found in app repository", f)
			}
		}

		// Read the files now so unsupported types, empty files, and the inline
		// size limit are caught at config time rather than on every execution.
		atts, err := attachments.Read(ctx.Files, spec.Files)
		if err != nil {
			return err
		}
		if err := checkAttachmentSize(atts); err != nil {
			return err
		}
	}

	// The schema supports expressions (like the prompt), which resolve only at
	// execution. Validate it as JSON when it has no unresolved expression;
	// Execute re-parses the resolved value.
	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
	if hasSchema && !strings.Contains(spec.OutputSchema, "{{") {
		if _, err := structuredoutput.Parse(spec.OutputSchema); err != nil {
			return err
		}
	}

	if ctx.Metadata != nil {
		_ = ctx.Metadata.Set(ChatCompletionNodeMetadata{
			Model:            spec.Model,
			StructuredOutput: hasSchema,
		})
	}

	return nil
}

func validateTemperature(temperature *float64) error {
	if temperature == nil {
		return nil
	}

	if *temperature < 0 || *temperature > 2 {
		return fmt.Errorf("temperature must be between 0 and 2")
	}

	return nil
}

func (c *ChatCompletion) Execute(ctx core.ExecutionContext) error {
	spec := ChatCompletionSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	if spec.Model == "" {
		return fmt.Errorf("model is required")
	}

	if spec.Prompt == "" {
		return fmt.Errorf("prompt is required")
	}

	if err := validateTemperature(spec.Temperature); err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	atts, err := attachments.Read(ctx.Files, spec.Files)
	if err != nil {
		return fmt.Errorf("failed to read attachments: %v", err)
	}
	if err := checkAttachmentSize(atts); err != nil {
		return err
	}

	schema, err := structuredoutput.Parse(spec.OutputSchema)
	if err != nil {
		return err
	}

	req := ChatCompletionRequest{
		Model:       spec.Model,
		Messages:    buildMessages(spec.SystemPrompt, spec.Prompt, atts),
		MaxTokens:   spec.MaxTokens,
		Temperature: spec.Temperature,
	}

	if schema != nil {
		req.ResponseFormat = &ResponseFormat{
			Type: "json_schema",
			JSONSchema: &JSONSchema{
				Name:   "structured_output",
				Strict: true,
				Schema: structuredoutput.Prepare(schema, true),
			},
		}
	}

	ctx.Logger.Infof("Running OpenCode Go chat completion with model %s", spec.Model)

	payload, err := executeModel(client, spec, atts, schema, req)
	if err != nil {
		return err
	}

	// The schema is a request, not a guarantee: a gateway or model that ignores
	// response_format returns prose. Parse only when the text is actually JSON
	// so downstream nodes can rely on parsed being set whenever it is present.
	if schema != nil && payload.Text != "" {
		var parsed any
		if err := json.Unmarshal([]byte(payload.Text), &parsed); err == nil {
			payload.Parsed = parsed
		}
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		ChatCompletionPayloadType,
		[]any{payload},
	)
}

func executeModel(client *Client, spec ChatCompletionSpec, atts []attachments.Attachment, schema map[string]any, chatReq ChatCompletionRequest) (*ChatCompletionPayload, error) {
	if supportedChatModels[spec.Model] {
		response, err := client.CreateChatCompletion(chatReq)
		if err != nil {
			return nil, err
		}
		return buildPayload(response)
	}

	if supportedMessagesModels[spec.Model] {
		maxTokens := 4096
		if spec.MaxTokens != nil {
			maxTokens = *spec.MaxTokens
		}
		messageReq := MessagesRequest{
			Model: spec.Model, Messages: []Message{{Role: "user", Content: buildMessagesContent(spec.Prompt, atts)}},
			System: spec.SystemPrompt, MaxTokens: maxTokens, Temperature: spec.Temperature,
		}
		if schema != nil {
			messageReq.OutputConfig = &MessagesOutputConfig{Format: &MessagesOutputFormat{Type: "json_schema", Schema: structuredoutput.Prepare(schema, false)}}
		}
		response, err := client.CreateMessage(messageReq)
		if err != nil {
			return nil, err
		}
		return buildMessagesPayload(response), nil
	}

	if supportedResponsesModels[spec.Model] {
		responseReq := ResponsesRequest{Model: spec.Model, Input: buildResponsesInput(spec.SystemPrompt, spec.Prompt, atts), MaxOutputTokens: spec.MaxTokens, Temperature: spec.Temperature}
		if schema != nil {
			responseReq.Text = &ResponsesTextConfig{Format: &ResponsesFormat{Type: "json_schema", Name: "structured_output", Schema: structuredoutput.Prepare(schema, true), Strict: true}}
		}
		response, err := client.CreateResponse(responseReq)
		if err != nil {
			return nil, err
		}
		return buildResponsesPayload(response), nil
	}

	return nil, fmt.Errorf("model %q is not supported by OpenCode Go", spec.Model)
}

// checkAttachmentSize caps the combined inline size of one request.
func checkAttachmentSize(atts []attachments.Attachment) error {
	total := 0
	for _, att := range atts {
		total += len(att.Data)
	}
	if total > maxAttachmentBytes {
		return fmt.Errorf("attachments total %d bytes, which exceeds the %d byte limit for inlined files", total, maxAttachmentBytes)
	}
	return nil
}

// buildMessages assembles the chat messages. Attachments are inlined into the
// user message as base64 data URLs, since OpenCode Go has no Files API. The
// system prompt goes first so it frames every request.
func buildMessages(systemPrompt, prompt string, atts []attachments.Attachment) []Message {
	messages := make([]Message, 0, 2)

	if systemPrompt != "" {
		messages = append(messages, Message{Role: "system", Content: systemPrompt})
	}

	if len(atts) == 0 {
		return append(messages, Message{Role: "user", Content: prompt})
	}

	parts := make([]ContentPart, 0, len(atts)+1)
	parts = append(parts, ContentPart{Type: "text", Text: prompt})
	for _, att := range atts {
		if att.IsImage() {
			parts = append(parts, ContentPart{Type: "image_url", ImageURL: &ImageURL{URL: dataURL(att)}})
			continue
		}

		if att.IsPDF() {
			parts = append(parts, ContentPart{
				Type: "file",
				File: &FilePart{Filename: att.Name, FileData: dataURL(att)},
			})
			continue
		}

		// Text goes in as prompt text rather than a file part. It needs no
		// document parsing on the provider side and costs prompt tokens only.
		parts = append(parts, ContentPart{
			Type: "text",
			Text: fmt.Sprintf("--- %s ---\n%s", att.Name, att.Data),
		})
	}

	return append(messages, Message{Role: "user", Content: parts})
}

// dataURL inlines an attachment, since OpenCode Go has no Files API to upload to.
func dataURL(att attachments.Attachment) string {
	return "data:" + att.UploadMIME() + ";base64," + base64.StdEncoding.EncodeToString(att.Data)
}

// buildPayload flattens the first choice into the node output. Content is null
// rather than empty when generation produced no text, so a run that produced
// nothing still emits an empty text field instead of failing here.
func buildPayload(response *ChatCompletionResponse) (*ChatCompletionPayload, error) {
	if response == nil || len(response.Choices) == 0 {
		return nil, fmt.Errorf("chat completion returned no choices")
	}

	choice := response.Choices[0]

	text := ""
	if choice.Message.Content != nil {
		text = *choice.Message.Content
	}

	return &ChatCompletionPayload{
		ID:           response.ID,
		Model:        response.Model,
		Text:         text,
		FinishReason: choice.FinishReason,
		Usage:        response.Usage,
		Response:     response,
	}, nil
}

func buildMessagesContent(prompt string, atts []attachments.Attachment) any {
	if len(atts) == 0 {
		return prompt
	}
	blocks := make([]MessageContentBlock, 0, len(atts)+1)
	for _, att := range atts {
		if att.IsImage() || att.IsPDF() {
			blockType := "document"
			if att.IsImage() {
				blockType = "image"
			}
			blocks = append(blocks, MessageContentBlock{Type: blockType, Source: &MessageContentSource{Type: "base64", MediaType: att.UploadMIME(), Data: base64.StdEncoding.EncodeToString(att.Data)}})
			continue
		}
		blocks = append(blocks, MessageContentBlock{Type: "text", Text: fmt.Sprintf("--- %s ---\n%s", att.Name, att.Data)})
	}
	blocks = append(blocks, MessageContentBlock{Type: "text", Text: prompt})
	return blocks
}

func buildResponsesInput(systemPrompt, prompt string, atts []attachments.Attachment) any {
	if len(atts) == 0 && systemPrompt == "" {
		return prompt
	}
	messages := make([]map[string]any, 0, 2)
	if systemPrompt != "" {
		messages = append(messages, map[string]any{"role": "system", "content": systemPrompt})
	}
	parts := []map[string]any{{"type": "input_text", "text": prompt}}
	for _, att := range atts {
		if att.IsImage() {
			parts = append(parts, map[string]any{"type": "input_image", "image_url": dataURL(att)})
			continue
		}
		if att.IsPDF() {
			parts = append(parts, map[string]any{"type": "input_file", "file_data": dataURL(att), "filename": att.Name})
			continue
		}
		parts = append(parts, map[string]any{"type": "input_text", "text": fmt.Sprintf("--- %s ---\n%s", att.Name, att.Data)})
	}
	messages = append(messages, map[string]any{"role": "user", "content": parts})
	return messages
}

func buildMessagesPayload(response *MessagesResponse) *ChatCompletionPayload {
	return &ChatCompletionPayload{ID: response.ID, Model: response.Model, Text: extractMessagesText(response), FinishReason: response.StopReason, Usage: &Usage{PromptTokens: response.Usage.InputTokens, CompletionTokens: response.Usage.OutputTokens, TotalTokens: response.Usage.InputTokens + response.Usage.OutputTokens}, Response: response}
}

func extractMessagesText(response *MessagesResponse) string {
	if response == nil {
		return ""
	}
	var out strings.Builder
	for _, block := range response.Content {
		if block.Type == "text" {
			if out.Len() > 0 {
				out.WriteString("\n")
			}
			out.WriteString(block.Text)
		}
	}
	return out.String()
}

func buildResponsesPayload(response *ResponsesResponse) *ChatCompletionPayload {
	usage := (*Usage)(nil)
	if response.Usage != nil {
		usage = &Usage{PromptTokens: response.Usage.InputTokens, CompletionTokens: response.Usage.OutputTokens, TotalTokens: response.Usage.TotalTokens}
	}
	return &ChatCompletionPayload{ID: response.ID, Model: response.Model, Text: extractResponsesText(response), Usage: usage, Response: response}
}

func extractResponsesText(response *ResponsesResponse) string {
	if response == nil {
		return ""
	}
	if response.OutputText != "" {
		return response.OutputText
	}
	var out strings.Builder
	for _, item := range response.Output {
		for _, content := range item.Content {
			if content.Type == "output_text" || content.Type == "text" {
				if out.Len() > 0 {
					out.WriteString("\n")
				}
				out.WriteString(content.Text)
			}
		}
	}
	return out.String()
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
