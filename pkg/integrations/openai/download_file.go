package openai

import (
	"encoding/base64"
	"errors"
	"fmt"
	"net/http"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const FileDownloadedPayloadType = "openai.file.downloaded"

// maxDownloadBytes caps downloads so huge files don't end up inlined in event
// payloads. The size is checked against metadata before fetching the content.
const maxDownloadBytes = 25 * 1024 * 1024

type DownloadFile struct{}

type DownloadFileSpec struct {
	File string `json:"file"`
}

// FileDownloadPayload carries the downloaded file content: text files as a
// plain string, everything else base64-encoded.
type FileDownloadPayload struct {
	ID       string `json:"id"`
	Filename string `json:"filename"`
	Purpose  string `json:"purpose"`
	Bytes    int64  `json:"bytes"`
	Encoding string `json:"encoding"`
	Content  string `json:"content"`
	URL      string `json:"url"`
}

func (c *DownloadFile) Name() string {
	return "openai.downloadFile"
}

func (c *DownloadFile) Label() string {
	return "Download File"
}

func (c *DownloadFile) Description() string {
	return "Download the content of a file stored in the OpenAI Files API"
}

func (c *DownloadFile) Documentation() string {
	return `The Download File component downloads the content of a file stored in the OpenAI Files API.

## Use Cases

- **Retrieve results**: Download batch output or fine-tuning result files
- **Workflow data**: Feed file content into downstream nodes for processing
- **Archiving**: Push file content to external storage systems

## Configuration

- **File**: The file to download

## Output

Returns the file content and metadata:
- **id**: The file ID
- **filename**: The file name
- **purpose**: The purpose the file was uploaded with
- **bytes**: The file size in bytes
- **encoding**: "text" for text content, "base64" for binary content
- **content**: The file content, base64-encoded when binary
- **url**: Link to the file in the OpenAI platform console

## Notes

- Files larger than 25MB are rejected; the size is checked before downloading
- OpenAI does not allow downloading files of some purposes (e.g. assistants, user_data, vision); the API error is surfaced as-is`
}

func (c *DownloadFile) Icon() string {
	return "file-down"
}

func (c *DownloadFile) Color() string {
	return "gray"
}

func (c *DownloadFile) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *DownloadFile) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "file",
			Label:       "File",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The file to download",
			Placeholder: "Select a file",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "file",
				},
			},
		},
	}
}

func (c *DownloadFile) Setup(ctx core.SetupContext) error {
	spec := DownloadFileSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	fileID := strings.TrimSpace(spec.File)
	if fileID == "" {
		return errors.New("file is required")
	}

	return resolveFileMetadata(ctx, fileID)
}

func (c *DownloadFile) Execute(ctx core.ExecutionContext) error {
	spec := DownloadFileSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	fileID := strings.TrimSpace(spec.File)
	if fileID == "" {
		return errors.New("file is required")
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	file, err := client.GetFile(fileID)
	if err != nil {
		return fmt.Errorf("failed to get file: %w", err)
	}

	if file.Bytes > maxDownloadBytes {
		return fmt.Errorf("file is %d bytes, which exceeds the %dMB download limit", file.Bytes, maxDownloadBytes/(1024*1024))
	}

	content, err := client.DownloadFileContent(fileID)
	if err != nil {
		return fmt.Errorf("failed to download file content: %w", err)
	}

	encoding, encoded := encodeFileContent(content)

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		FileDownloadedPayloadType,
		[]any{FileDownloadPayload{
			ID:       file.ID,
			Filename: file.Filename,
			Purpose:  file.Purpose,
			Bytes:    file.Bytes,
			Encoding: encoding,
			Content:  encoded,
			URL:      client.FileURL(file.ID),
		}},
	)
}

// encodeFileContent returns the payload encoding and content for downloaded
// bytes: text content is passed through as a string, everything else is
// base64-encoded. The content type is sniffed from the bytes because the Files
// API does not return a MIME type.
func encodeFileContent(content []byte) (string, string) {
	if isTextMIME(http.DetectContentType(content)) {
		return "text", string(content)
	}
	return "base64", base64.StdEncoding.EncodeToString(content)
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

func (c *DownloadFile) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *DownloadFile) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *DownloadFile) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *DownloadFile) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *DownloadFile) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *DownloadFile) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
