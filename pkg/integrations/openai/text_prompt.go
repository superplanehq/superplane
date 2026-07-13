package openai

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"path"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/configuration/attachments"
	"github.com/superplanehq/superplane/pkg/configuration/structuredoutput"
	"github.com/superplanehq/superplane/pkg/core"
	gitprovider "github.com/superplanehq/superplane/pkg/git/provider"
)

const ResponsePayloadType = "openai.response"

type CreateResponse struct{}

type CreateResponseSpec struct {
	Model           string   `json:"model"`
	Input           string   `json:"input"`
	Files           []string `json:"files"`
	CodeInterpreter bool     `json:"codeInterpreter"`
	OutputSchema    string   `json:"outputSchema"`
}

type ResponsePayload struct {
	ID        string          `json:"id"`
	Model     string          `json:"model"`
	Text      string          `json:"text"`
	Parsed    any             `json:"parsed,omitempty"`
	Usage     *ResponseUsage  `json:"usage,omitempty"`
	Artifacts []Artifact      `json:"artifacts,omitempty"`
	Response  *OpenAIResponse `json:"response"`
}

// Artifact is a file the code interpreter generated in its container. Its
// content is embedded in the payload — text files as a plain string,
// everything else base64-encoded — because containers (and their files)
// expire ~20 minutes after their last activity. Files over the inline size
// cap carry metadata and the API download URL only.
type Artifact struct {
	FileID      string `json:"fileId"`
	ContainerID string `json:"containerId"`
	Filename    string `json:"filename"`
	Bytes       int64  `json:"bytes,omitempty"`
	Encoding    string `json:"encoding,omitempty"`
	Content     string `json:"content,omitempty"`
	DownloadURL string `json:"downloadUrl"` // {BaseURL}/containers/{cid}/files/{fid}/content
}

// maxInlineArtifactSizeBytes caps how large a generated file can be for its
// content to be embedded in the output payload. Larger files keep their
// metadata and download URL only.
const maxInlineArtifactSizeBytes = 10 * 1024 * 1024

// ResponseNodeMetadata is node-level metadata surfaced in the UI so the
// configured model and options are visible on the node without opening it.
type ResponseNodeMetadata struct {
	Model            string `json:"model" mapstructure:"model"`
	StructuredOutput bool   `json:"structuredOutput" mapstructure:"structuredOutput"`
	CodeInterpreter  bool   `json:"codeInterpreter" mapstructure:"codeInterpreter"`
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
- **Code Interpreter**: (Optional) Let the model write and run Python in a sandboxed container. Files it creates are emitted as artifacts.
- **Structured Output**: (Optional) Provide a JSON Schema for the response and the model returns JSON matching it, available on the parsed output. The schema is validated before the request and sent in OpenAI strict mode; strict mode marks every property required, so express optional fields by making their type nullable.

## Output

Returns the generated response including:
- **text**: The generated text response
- **model**: The model used for generation
- **usage**: Token usage information (prompt tokens, completion tokens, total tokens)
- **id**: Response ID for tracking
- **parsed**: When Structured Output is configured, the response parsed into an object.
- **artifacts**: When Code Interpreter is enabled, the files the model generated, with the file content included: text files as plain text (` + "`encoding: \"text\"`" + `), everything else base64-encoded (` + "`encoding: \"base64\"`" + `). Each artifact carries fileId, containerId, filename, bytes, and a downloadUrl; files over 10MB include metadata and the download URL only (note that containers expire about 20 minutes after their last activity).

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
		{
			Name:        "codeInterpreter",
			Label:       "Code Interpreter",
			Type:        configuration.FieldTypeBool,
			Required:    false,
			Default:     false,
			Description: "Let the model write and run Python in a sandboxed container. Files it creates are emitted as artifacts.",
		},
		structuredoutput.ConfigField(
			"outputSchema",
			"Structured Output",
			"A JSON Schema describing the response. The model is constrained to return JSON matching it (available on the `parsed` output). Edit the default schema; it is validated before the request. Strict mode requires every property to be listed in `required`, so all top-level and nested properties are marked required automatically.",
		),
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

		// Read the files now so unsupported types, empty files, and size limits
		// are caught at config time rather than on every execution.
		if _, err := attachments.Read(ctx.Files, spec.Files); err != nil {
			return err
		}
	}

	// The schema field supports expressions (like the prompt), which are only
	// resolved at execution. Validate it as JSON only when it has no unresolved
	// expression; Execute re-parses the resolved value.
	hasSchema := strings.TrimSpace(spec.OutputSchema) != ""
	if hasSchema && !strings.Contains(spec.OutputSchema, "{{") {
		if _, err := structuredoutput.Parse(spec.OutputSchema); err != nil {
			return err
		}
	}

	if ctx.Metadata != nil {
		_ = ctx.Metadata.Set(ResponseNodeMetadata{
			Model:            spec.Model,
			StructuredOutput: hasSchema,
			CodeInterpreter:  spec.CodeInterpreter,
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

	schema, err := structuredoutput.Parse(spec.OutputSchema)
	if err != nil {
		return err
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	// Read attached repository files and build the Responses API input: files are
	// uploaded to the Files API and referenced by file_id alongside the prompt.
	atts, err := attachments.Read(ctx.Files, spec.Files)
	if err != nil {
		return fmt.Errorf("failed to read attachments: %v", err)
	}
	input, fileIDs, err := buildInput(client, atts, spec.Input)
	if err != nil {
		return err
	}
	defer cleanupFiles(client, fileIDs)

	req := CreateResponseRequest{Model: spec.Model, Input: input}
	if spec.CodeInterpreter {
		req.Tools = []any{map[string]any{"type": "code_interpreter", "container": map[string]any{"type": "auto"}}}
	}
	if schema != nil {
		req.Text = &ResponseTextConfig{
			Format: &ResponseFormat{
				Type:   "json_schema",
				Name:   "structured_output",
				Schema: structuredoutput.Prepare(schema, true),
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
		ID:        response.ID,
		Model:     response.Model,
		Text:      text,
		Usage:     response.Usage,
		Artifacts: extractArtifacts(client, response, spec.CodeInterpreter),
		Response:  response,
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

// buildInput uploads each attachment to the OpenAI Files API and builds the
// Responses API input: a plain string when there are no attachments, otherwise
// a user message carrying input_text + input_image/input_file parts referenced
// by file_id. The returned file IDs should be cleaned up after the request.
func buildInput(client *Client, atts []attachments.Attachment, prompt string) (any, []string, error) {
	if len(atts) == 0 {
		return prompt, nil, nil
	}

	parts := make([]InputPart, 0, len(atts)+1)
	parts = append(parts, InputPart{Type: "input_text", Text: prompt})
	fileIDs := make([]string, 0, len(atts))
	for _, att := range atts {
		purpose := "user_data"
		if att.IsImage() {
			purpose = "vision"
		}
		fileID, err := client.UploadFile(bytes.NewReader(att.Data), att.Name, purpose, att.UploadMIME())
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

// extractArtifacts collects the files the code interpreter generated and
// embeds their content in the payload. Generated files normally arrive as
// container_file_citation annotations on the output text; when annotations
// are missing (a known gap when combined with structured output), it falls
// back to listing the container's assistant-generated files. Content fetches
// are best-effort: a failure degrades the artifact to metadata + download URL.
func extractArtifacts(client *Client, response *OpenAIResponse, codeInterpreter bool) []Artifact {
	if response == nil {
		return nil
	}

	var artifacts []Artifact
	seen := map[string]bool{}
	for _, output := range response.Output {
		for _, content := range output.Content {
			for _, annotation := range content.Annotations {
				if annotation.Type != "container_file_citation" || annotation.FileID == "" || seen[annotation.FileID] {
					continue
				}
				seen[annotation.FileID] = true
				artifact := Artifact{
					FileID:      annotation.FileID,
					ContainerID: annotation.ContainerID,
					Filename:    annotation.Filename,
					DownloadURL: client.ContainerFileContentURL(annotation.ContainerID, annotation.FileID),
				}
				if file, err := client.GetContainerFile(annotation.ContainerID, annotation.FileID); err == nil {
					artifact.Bytes = file.Bytes
					if artifact.Filename == "" {
						artifact.Filename = containerFileName(file.Path)
					}
					inlineArtifactContent(client, &artifact)
				}
				artifacts = append(artifacts, artifact)
			}
		}
	}

	if len(artifacts) > 0 || !codeInterpreter {
		return artifacts
	}

	for _, output := range response.Output {
		if output.Type != "code_interpreter_call" || output.ContainerID == "" {
			continue
		}
		files, err := client.ListContainerFiles(output.ContainerID)
		if err != nil {
			continue
		}
		for _, file := range files {
			if file.Source != "assistant" || file.ID == "" || seen[file.ID] {
				continue
			}
			seen[file.ID] = true
			artifact := Artifact{
				FileID:      file.ID,
				ContainerID: output.ContainerID,
				Filename:    containerFileName(file.Path),
				Bytes:       file.Bytes,
				DownloadURL: client.ContainerFileContentURL(output.ContainerID, file.ID),
			}
			inlineArtifactContent(client, &artifact)
			artifacts = append(artifacts, artifact)
		}
	}
	return artifacts
}

// inlineArtifactContent embeds the artifact's content when it is within the
// inline size cap, sniffing the content type to decide between plain text and
// base64. Failures leave the artifact with metadata and download URL only.
func inlineArtifactContent(client *Client, artifact *Artifact) {
	if artifact.Bytes > maxInlineArtifactSizeBytes {
		return
	}
	content, err := client.DownloadContainerFileContent(artifact.ContainerID, artifact.FileID)
	if err != nil {
		return
	}
	if isTextMIME(http.DetectContentType(content)) {
		artifact.Encoding, artifact.Content = "text", string(content)
		return
	}
	artifact.Encoding, artifact.Content = "base64", base64.StdEncoding.EncodeToString(content)
}

// containerFileName derives the file name from the container path
// (e.g. /mnt/data/plot.png -> plot.png).
func containerFileName(containerPath string) string {
	if containerPath == "" {
		return ""
	}
	return path.Base(containerPath)
}

// isTextMIME reports whether a MIME type is text-like and safe to emit as a
// plain string payload.
func isTextMIME(mimeType string) bool {
	mt := strings.ToLower(strings.TrimSpace(strings.Split(mimeType, ";")[0]))
	if strings.HasPrefix(mt, "text/") {
		return true
	}
	switch mt {
	case "application/json", "application/xml", "application/x-yaml", "application/yaml", "application/javascript":
		return true
	}
	return strings.HasSuffix(mt, "+json") || strings.HasSuffix(mt, "+xml")
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
