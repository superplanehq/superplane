package jfrogartifactory

import (
	"fmt"

	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
)

const (
	GetArtifactInfoPayloadType = "jfrogArtifactory.artifact.info"
)

type GetArtifactInfo struct{}

type GetArtifactInfoSpec struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Path       string `json:"path" mapstructure:"path"`
}

type GetArtifactInfoNodeMetadata struct {
	Repository string `json:"repository" mapstructure:"repository"`
}

func (g *GetArtifactInfo) Name() string {
	return "jfrogArtifactory.getArtifactInfo"
}

func (g *GetArtifactInfo) Label() string {
	return "Get Artifact Info"
}

func (g *GetArtifactInfo) Description() string {
	return "Get metadata about an artifact in JFrog Artifactory"
}

func (g *GetArtifactInfo) Documentation() string {
	return `The Get Artifact Info component retrieves metadata about an artifact stored in JFrog Artifactory.

## Use Cases

- **Artifact verification**: Check artifact existence and checksums before deployment
- **Pipeline metadata**: Retrieve artifact details for downstream workflow steps
- **Audit and tracking**: Get creation time, size, and author information

## Configuration

- **Repository**: Select the Artifactory repository containing the artifact
- **Path**: The path to the artifact within the repository (supports expressions)

## Output

Returns artifact metadata including repository, path, size, checksums, download URI, and timestamps.`
}

func (g *GetArtifactInfo) Icon() string {
	return "jfrogArtifactory"
}

func (g *GetArtifactInfo) Color() string {
	return "gray"
}

func (g *GetArtifactInfo) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (g *GetArtifactInfo) Configuration() []configuration.Field {
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
			Description: "Path to the artifact within the repository",
			Placeholder: "e.g. path/to/file.jar",
		},
	}
}

func (g *GetArtifactInfo) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (g *GetArtifactInfo) Setup(ctx core.SetupContext) error {
	spec := GetArtifactInfoSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if spec.Repository == "" {
		return fmt.Errorf("repository is required")
	}

	if spec.Path == "" {
		return fmt.Errorf("path is required")
	}

	return ctx.Metadata.Set(GetArtifactInfoNodeMetadata{
		Repository: spec.Repository,
	})
}

func (g *GetArtifactInfo) Execute(ctx core.ExecutionContext) error {
	spec := GetArtifactInfoSpec{}
	if err := mapstructure.Decode(ctx.Configuration, &spec); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	client, err := NewClient(ctx.HTTP, ctx.Integration)
	if err != nil {
		return err
	}

	info, err := client.GetArtifactInfo(spec.Repository, spec.Path)
	if err != nil {
		return fmt.Errorf("error getting artifact info: %v", err)
	}

	payload := map[string]any{
		"repo":         info.Repo,
		"path":         info.Path,
		"created":      info.Created,
		"createdBy":    info.CreatedBy,
		"lastModified": info.LastModified,
		"modifiedBy":   info.ModifiedBy,
		"downloadUri":  info.DownloadURI,
		"mimeType":     info.MimeType,
		"size":         info.Size,
		"uri":          info.URI,
	}

	if info.Checksums != nil {
		payload["checksums"] = map[string]any{
			"sha1":   info.Checksums.SHA1,
			"md5":    info.Checksums.MD5,
			"sha256": info.Checksums.SHA256,
		}
	}

	return ctx.ExecutionState.Emit(core.DefaultOutputChannel.Name, GetArtifactInfoPayloadType, []any{payload})
}

func (g *GetArtifactInfo) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (g *GetArtifactInfo) HandleWebhook(ctx core.WebhookRequestContext) (int, error) {
	return 0, nil
}

func (g *GetArtifactInfo) Actions() []core.Action {
	return []core.Action{}
}

func (g *GetArtifactInfo) HandleAction(ctx core.ActionContext) error {
	return nil
}

func (g *GetArtifactInfo) Cleanup(ctx core.SetupContext) error {
	return nil
}
