package contents

import (
	"context"
	"errors"
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
	Repository string `mapstructure:"repository"`
	Path       string `mapstructure:"path"`
	Ref        string `mapstructure:"ref"`
}

type GetFileContentOutput struct {
	Content string `json:"content"`
	SHA     string `json:"sha"`
	Path    string `json:"path"`
	Ref     string `json:"ref"`
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
	return `The Get File Content component reads a file from a GitHub repository.

## Use Cases

- **Configuration checks**: Read configuration before making a workflow decision
- **File comparisons**: Compare repository content with generated or remote content
- **Metadata extraction**: Read small source or documentation files without cloning a repository

## Configuration

- **Repository**: Select the GitHub repository
- **Path**: Repository-relative path to the file
- **Ref**: Optional branch, tag, or commit SHA. When omitted, GitHub uses the repository's default branch.

## Output

Returns the decoded file content together with its path, blob SHA, and the requested ref.`
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
			Placeholder: "e.g. config/app.yaml",
			Description: "Repository-relative path to the file. Supports template variables from previous steps.",
		},
		{
			Name:        "ref",
			Label:       "Ref",
			Type:        configuration.FieldTypeString,
			Required:    false,
			Placeholder: "e.g. main, v1.0.0, or a commit SHA",
			Description: "Optional branch, tag, or commit SHA. Defaults to the repository's default branch.",
		},
	}
}

func (c *GetFileContent) Setup(ctx core.SetupContext) error {
	config, err := decodeGetFileContentConfiguration(ctx.Configuration)
	if err != nil {
		return err
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
	config, err := decodeGetFileContentConfiguration(ctx.Configuration)
	if err != nil {
		return err
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

	if file == nil && len(directory) > 0 {
		return errors.New("path points to a directory, not a file")
	}

	if file == nil {
		return errors.New("GitHub returned no file content")
	}

	content, err := file.GetContent()
	if err != nil {
		return fmt.Errorf("failed to decode file content: %w", err)
	}

	return ctx.ExecutionState.Emit(
		core.DefaultOutputChannel.Name,
		"github.fileContent",
		[]any{GetFileContentOutput{
			Content: content,
			SHA:     file.GetSHA(),
			Path:    file.GetPath(),
			Ref:     config.Ref,
		}},
	)
}

func decodeGetFileContentConfiguration(value any) (GetFileContentConfiguration, error) {
	var config GetFileContentConfiguration
	if err := mapstructure.Decode(value, &config); err != nil {
		return config, fmt.Errorf("failed to decode configuration: %w", err)
	}

	return config, nil
}

func validateGetFileContentConfiguration(config GetFileContentConfiguration) error {
	if config.Repository == "" {
		return errors.New("repository is required")
	}

	if config.Path == "" {
		return errors.New("path is required")
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
