package openai

import (
	"bytes"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
	"github.com/superplanehq/superplane/pkg/integrations/llmattach"
)

const ResponsePayloadType = "openai.response"

type CreateResponse struct{}

type CreateResponseSpec struct {
	Model string   `json:"model"`
	Input string   `json:"input"`
	Files []string `json:"files"`
}

type ResponsePayload struct {
	ID       string          `json:"id"`
	Model    string          `json:"model"`
	Text     string          `json:"text"`
	Usage    *ResponseUsage  `json:"usage,omitempty"`
	Response *OpenAIResponse `json:"response"`
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
- **Files**: (Optional) Attach files from the Files tab (images, PDFs, or text). They are uploaded to the OpenAI Files API and sent alongside the prompt.

## Output

Returns the generated response including:
- **text**: The generated text response
- **model**: The model used for generation
- **usage**: Token usage information (prompt tokens, completion tokens, total tokens)
- **id**: Response ID for tracking

## Notes

- Requires a valid OpenAI API key configured in the application settings
- Response quality and speed depend on the selected model
- Token usage is tracked and may incur costs based on your OpenAI plan
- Supports OpenAI-compatible providers by setting a custom Base URL in the integration settings (e.g., Azure OpenAI, Ollama, vLLM)`
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
			Name:        "files",
			Label:       "Files",
			Type:        configuration.FieldTypeList,
			Required:    false,
			Description: "Files from the Files tab to attach to the prompt (images, PDFs, or text)",
			TypeOptions: &configuration.TypeOptions{
				List: &configuration.ListTypeOptions{
					ItemLabel: "File path",
					ItemDefinition: &configuration.ListItemDefinition{
						Type: configuration.FieldTypeRepositoryFile,
					},
				},
			},
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

	attachments, err := llmattach.Read(ctx.Files, spec.Files)
	if err != nil {
		return fmt.Errorf("failed to read attachments: %v", err)
	}
	input, fileIDs, err := buildInput(client, attachments, spec.Input)
	if err != nil {
		return err
	}
	defer cleanupFiles(client, fileIDs)

	response, err := client.CreateResponse(CreateResponseRequest{Model: spec.Model, Input: input})
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

// buildInput uploads each attachment to the OpenAI Files API and builds the
// Responses API input: a plain string when there are no attachments, otherwise
// a user message carrying input_text + input_image/input_file parts referenced
// by file_id. The returned file IDs should be cleaned up after the request.
func buildInput(client *Client, attachments []llmattach.Attachment, prompt string) (any, []string, error) {
	if len(attachments) == 0 {
		return prompt, nil, nil
	}

	parts := make([]InputPart, 0, len(attachments)+1)
	parts = append(parts, InputPart{Type: "input_text", Text: prompt})
	fileIDs := make([]string, 0, len(attachments))
	for _, att := range attachments {
		purpose := "user_data"
		if att.IsImage() {
			purpose = "vision"
		}
		fileID, err := client.UploadFile(bytes.NewReader(att.Data), att.Name, purpose, att.Mime)
		if err != nil {
			cleanupFiles(client, fileIDs)
			return nil, nil, fmt.Errorf("upload file %q: %w", att.Name, err)
		}
		fileIDs = append(fileIDs, fileID)

		if att.IsImage() {
			parts = append(parts, InputPart{Type: "input_image", FileID: fileID})
		} else {
			parts = append(parts, InputPart{Type: "input_file", FileID: fileID})
		}
	}

	return []InputMessage{{Role: "user", Content: parts}}, fileIDs, nil
}

// cleanupFiles best-effort deletes uploaded files after the request completes.
func cleanupFiles(client *Client, fileIDs []string) {
	for _, id := range fileIDs {
		_ = client.DeleteFile(id)
	}
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

func (c *CreateResponse) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *CreateResponse) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *CreateResponse) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
