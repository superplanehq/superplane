package openai

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const FileFetchedPayloadType = "openai.file.fetched"

type GetFile struct{}

type GetFileSpec struct {
	File string `json:"file"`
}

// FilePayload is the emitted file metadata. Timestamps are converted from the
// API's unix seconds to RFC3339 strings.
type FilePayload struct {
	ID        string `json:"id"`
	Filename  string `json:"filename"`
	Purpose   string `json:"purpose"`
	Bytes     int64  `json:"bytes"`
	CreatedAt string `json:"createdAt"`
	ExpiresAt string `json:"expiresAt,omitempty"`
	URL       string `json:"url"`
}

// FileNodeMetadata stores a display label for file integration resources
// (the picker value is the file ID).
type FileNodeMetadata struct {
	Filename string `json:"filename"`
}

func (c *GetFile) Name() string {
	return "openai.getFile"
}

func (c *GetFile) Label() string {
	return "Get File"
}

func (c *GetFile) Description() string {
	return "Retrieve metadata for a file stored in the OpenAI Files API"
}

func (c *GetFile) Documentation() string {
	return `The Get File component fetches metadata for a file stored in the OpenAI Files API.

## Use Cases

- **Pre-flight validation**: Confirm a file exists and inspect its size before downloading it
- **Audit**: Capture file metadata (purpose, size, expiration) at a point in time
- **Workflow data**: Expose file metadata as payload data for downstream nodes

## Configuration

- **File**: The file to retrieve

## Output

Returns the file metadata including:
- **id**: The file ID
- **filename**: The file name
- **purpose**: The purpose the file was uploaded with (e.g. assistants, batch_output)
- **bytes**: The file size in bytes
- **createdAt**: When the file was created (RFC3339)
- **expiresAt**: When the file expires (RFC3339), when set
- **url**: Link to the file in the OpenAI platform console`
}

func (c *GetFile) Icon() string {
	return "file-text"
}

func (c *GetFile) Color() string {
	return "gray"
}

func (c *GetFile) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetFile) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:        "file",
			Label:       "File",
			Type:        configuration.FieldTypeIntegrationResource,
			Required:    true,
			Description: "The file to retrieve",
			Placeholder: "Select a file",
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type: "file",
				},
			},
		},
	}
}

func (c *GetFile) Setup(ctx core.SetupContext) error {
	spec := GetFileSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %v", err)
	}

	fileID := strings.TrimSpace(spec.File)
	if fileID == "" {
		return errors.New("file is required")
	}

	return resolveFileMetadata(ctx, fileID)
}

// resolveFileMetadata stores the file's name as node metadata so the UI can
// display it instead of the raw file ID. Resolution is best-effort: expressions
// are only resolved at execution time, and a failed metadata fetch must not
// block the node setup, so both fall back to the configured value.
func resolveFileMetadata(ctx core.SetupContext, fileID string) error {
	if ctx.Metadata == nil || ctx.Integration == nil || ctx.HTTP == nil {
		return nil
	}

	if strings.Contains(fileID, "{{") {
		return ctx.Metadata.Set(FileNodeMetadata{Filename: fileID})
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return ctx.Metadata.Set(FileNodeMetadata{Filename: fileID})
	}

	file, err := client.GetFile(fileID)
	if err != nil || file.Filename == "" {
		return ctx.Metadata.Set(FileNodeMetadata{Filename: fileID})
	}

	return ctx.Metadata.Set(FileNodeMetadata{Filename: file.Filename})
}

func (c *GetFile) Execute(ctx core.ExecutionContext) error {
	spec := GetFileSpec{}
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

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		FileFetchedPayloadType,
		[]any{FilePayload{
			ID:        file.ID,
			Filename:  file.Filename,
			Purpose:   file.Purpose,
			Bytes:     file.Bytes,
			CreatedAt: unixToRFC3339(file.CreatedAt),
			ExpiresAt: unixToRFC3339(file.ExpiresAt),
			URL:       client.FileURL(file.ID),
		}},
	)
}

// unixToRFC3339 converts a unix seconds timestamp to an RFC3339 string,
// returning "" for the zero value so optional timestamps can be omitted.
func unixToRFC3339(ts int64) string {
	if ts == 0 {
		return ""
	}
	return time.Unix(ts, 0).UTC().Format(time.RFC3339)
}

func (c *GetFile) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetFile) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetFile) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return http.StatusOK, nil, nil
}

func (c *GetFile) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetFile) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetFile) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
