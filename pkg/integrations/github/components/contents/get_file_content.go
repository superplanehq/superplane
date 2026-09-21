package contents

import (
	"context"
	"fmt"

	"github.com/google/go-github/v84/github"
	"github.com/google/uuid"
	"github.com/mitchellh/mapstructure"
	"github.com/superplanehq/superplane/pkg/configuration"
	"github.com/superplanehq/superplane/pkg/core"
	"github.com/superplanehq/superplane/pkg/integrations/github/common"
)

type GetFileContent struct{}

type GetFileContentConfiguration struct {
	Repository string `json:"repository" mapstructure:"repository"`
	Path       string `json:"path" mapstructure:"path"`
	Ref        string `json:"ref,omitempty" mapstructure:"ref"`
}

type GetFileContentOutput struct {
	Content     string `json:"content"`
	Name        string `json:"name"`
	Path        string `json:"path"`
	SHA         string `json:"sha"`
	Size        int    `json:"size"`
	Ref         string `json:"ref,omitempty"`
	URL         string `json:"url"`
	HTMLURL     string `json:"html_url"`
	DownloadURL string `json:"download_url"`
}

func (c *GetFileContent) Name() string {
	return "github.getFileContent"
}

func (c *GetFileContent) Label() string {
	return "Get File Content"
}

func (c *GetFileContent) Description() string {
	return "Read a file from a GitHub repository"
}

func (c *GetFileContent) Documentation() string {
	return `The Get File Content component reads a file from a GitHub repository without cloning the repository.

## Use Cases

- **Configuration checks**: Read configuration files before making workflow decisions
- **Documentation automation**: Retrieve README files or other text documents
- **Repository comparisons**: Read files from branches, tags, or commits for comparison steps
- **Agent context**: Provide small repository files to downstream agents or actions

## Configuration

- **Repository**: Select the GitHub repository containing the file
- **Path**: Path to the file from the repository root (supports expressions)
- **Ref**: Optional branch name, tag, or commit SHA. When omitted, GitHub uses the repository's default branch.

## Output

Returns the decoded file content together with its name, path, SHA, size, URLs, and the requested ref.

This component reads files only. Directory paths are rejected; use a path that points to a single file.`
}

func (c *GetFileContent) Icon() string {
	return "github"
}

func (c *GetFileContent) Color() string {
	return "gray"
}

func (c *GetFileContent) OutputChannels(configuration any) []core.OutputChannel {
	return []core.OutputChannel{core.DefaultOutputChannel}
}

func (c *GetFileContent) Configuration() []configuration.Field {
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
			Label:       "File Path",
			Type:        configuration.FieldTypeString,
			Required:    true,
			Placeholder: "e.g., README.md or config/app.yaml",
			Description: "Path to the file from the repository root",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g., main, v1.2.3, or a commit SHA",
			Description: "Optional branch, tag, or commit SHA. Defaults to the repository's default branch.",
		},
	}
}

func (c *GetFileContent) Setup(ctx core.SetupContext) error {
	var config GetFileContentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateGetFileContentConfiguration(config); err != nil {
		return err
	}

	return common.EnsureRepoInMetadata(
		ctx.Metadata,
		ctx.Integration,
		ctx.HTTP,
		ctx.Configuration,
	)
}

func (c *GetFileContent) Execute(ctx core.ExecutionContext) error {
	var config GetFileContentConfiguration
	if err := mapstructure.Decode(ctx.Configuration, &config); err != nil {
		return fmt.Errorf("failed to decode configuration: %w", err)
	}

	if err := validateGetFileContentConfiguration(config); err != nil {
		return err
	}

	client, err := common.NewClient(ctx.Integration, ctx.HTTP)
	if err != nil {
		return fmt.Errorf("failed to initialize GitHub client: %w", err)
	}

	file, directory, _, err := client.GetContents(
		context.Background(),
		config.Repository,
		config.Path,
		&github.RepositoryContentGetOptions{Ref: config.Ref},
	)
	if err != nil {
		return fmt.Errorf("failed to get file content: %w", err)
	}

	if file == nil {
		if directory != nil {
			return fmt.Errorf("path %q points to a directory; expected a file", config.Path)
		}
		return fmt.Errorf("GitHub returned no file content for path %q", config.Path)
	}

	content, err := file.GetContent()
	if err != nil {
		return fmt.Errorf("failed to decode file content: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.fileContent",
		[]any{GetFileContentOutput{
			Content:     content,
			Name:        file.GetName(),
			Path:        file.GetPath(),
			SHA:         file.GetSHA(),
			Size:        file.GetSize(),
			Ref:         config.Ref,
			URL:         file.GetURL(),
			HTMLURL:     file.GetHTMLURL(),
			DownloadURL: file.GetDownloadURL(),
		}},
	)
}

func validateGetFileContentConfiguration(config GetFileContentConfiguration) error {
	if config.Repository == "" {
		return fmt.Errorf("repository is required")
	}
	if config.Path == "" {
		return fmt.Errorf("file path is required")
	}
	return nil
}

func (c *GetFileContent) ProcessQueueItem(ctx core.ProcessQueueContext) (*uuid.UUID, error) {
	return ctx.DefaultProcessing()
}

func (c *GetFileContent) HandleWebhook(ctx core.WebhookRequestContext) (int, *core.WebhookResponseBody, error) {
	return 200, nil, nil
}

func (c *GetFileContent) Cancel(ctx core.ExecutionContext) error {
	return nil
}

func (c *GetFileContent) Cleanup(ctx core.SetupContext) error {
	return nil
}

func (c *GetFileContent) Hooks() []core.Hook {
	return []core.Hook{}
}

func (c *GetFileContent) HandleHook(ctx core.ActionHookContext) error {
	return nil
}
