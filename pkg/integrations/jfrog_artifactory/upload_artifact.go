package jfrog_artifactory

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	UploadArtifactPayloadType = "jfrogArtifactory.artifact.uploaded"
)

type UploadArtifact struct{}

type UploadArtifactSpec struct {
	Repository  string `json:"repository" mapstructure:"repository"`
	Path        string `json:"path" mapstructure:"path"`
	Content     string `json:"content" mapstructure:"content"`
	ContentType string `json:"contentType" mapstructure:"contentType"`
}

type UploadArtifactNodeMetadata struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

func (u *UploadArtifact) Name() string {
	return "jfrogArtifactory.uploadArtifact"
}

func (u *UploadArtifact) Label() string {
	return "Upload Artifact"
}

func (u *UploadArtifact) Description() string {
	return "Upload an artifact to JFrog Artifactory"
}

func (u *UploadArtifact) Documentation() string {
	return `The Upload Artifact component deploys an artifact to a JFrog Artifactory repository.

## Use Cases

- **CI/CD publishing**: Upload build artifacts to Artifactory as part of a pipeline
- **Release management**: Deploy versioned artifacts to release repositories
- **Configuration distribution**: Push configuration files to shared repositories

## Configuration

- **Repository**: Select the target Artifactory repository
- **Path**: The destination path for the artifact (supports expressions)
- **Content**: The artifact content to upload (supports expressions for piping data from previous nodes)
- **Content Type**: Optional MIME type (defaults to application/octet-stream)

## Output

Returns deploy metadata including repository, path, size, checksums, and download URI.`
}

func (u *UploadArtifact) Icon() string {
	return "jfrogArtifactory"
}

func (u *UploadArtifact) Color() string {
	return "gray"
}

func (u *UploadArtifact) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (u *UploadArtifact) Configuration() []configuration.Field {
	return []configuration.Field{
		{
			Name:     "repository",
			Label:    "Repository",
			Type:     configuration.FieldTypeIntegrationResource,
			Required: true,
			TypeOptions: &configuration.TypeOptions{
				Resource: &configuration.ResourceTypeOptions{
					Type:           "repository",
					UseNameAsValue: true,
				},
			},
		},
		{
			Name:        "path",
			Label:       "Path",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Destination path for the artifact",
			Placeholder: "e.g. path/to/file.jar",
		},
		{
			Name:        "content",
			Label:       "Content",
			Type:        configuration.FieldTypeExpression,
			Required:    true,
			Description: "Artifact content to upload",
		},
		{
			Name:        "contentType",
			Label:       "Content Type",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Description: "MIME type of the artifact",
			Placeholder: "application/octet-stream",
		},
	}
}

func (u *UploadArtifact) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (u *UploadArtifact) Setup(ctx core.SetupContext) error {
	spec := UploadArtifactSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	return ctx.Metadata.Set(UploadArtifactNodeMetadata{
		Repository: spec.Repository,
	})
}

func (u *UploadArtifact) Execute(ctx core.ExecutionContext) error {
	spec := UploadArtifactSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	resp, err := client.DeployArtifact(spec.Repository, spec.Path, strings.NewReader(spec.Content), spec.ContentType)
	if err != nil {
		return fmt.Errorf("error uploading artifact: %v", err)
	}

	payload := map[string]any{
		"repo":        resp.Repo,
		"path":        resp.Path,
		"created":     resp.Created,
		"createdBy":   resp.CreatedBy,
		"downloadUri": resp.DownloadURI,
		"mimeType":    resp.MimeType,
		"size":        resp.Size,
		"uri":         resp.URI,
	}

	if resp.Checksums != nil {
		payload["checksums"] = map[string]any{
			"sha1":   resp.Checksums.SHA1,
			"md5":    resp.Checksums.MD5,
			"sha256": resp.Checksums.SHA256,
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, UploadArtifactPayloadType, []any{payload})
}

func (u *UploadArtifact) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (u *UploadArtifact) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (u *UploadArtifact) Actions() []core.Action {
	return []core.Action{}
}

func (u *UploadArtifact) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (u *UploadArtifact) Cleanup(ctx core.SetupContext) error {
	return nil
}
