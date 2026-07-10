package apps

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/common"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"github.com/superplanehq/superplane/pkg/yaml"
)

type createCommand struct {
	name        *string
	description *string
	files       *[]string
	message     *string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	name := ""
	if c.name != nil {
		name = strings.TrimSpace(*c.name)
	}
	if name == "" {
		return fmt.Errorf("--name is required")
	}

	description := ""
	if c.description != nil {
		description = strings.TrimSpace(*c.description)
	}

	localFiles := []string{}
	if c.files != nil {
		localFiles = append(localFiles, *c.files...)
	}

	request := openapi_client.NewCanvasesCreateCanvasRequest()
	canvas := openapi_client.NewCanvasesCanvas()
	metadata := openapi_client.NewCanvasesCanvasMetadata()
	metadata.SetName(name)
	if description != "" {
		metadata.SetDescription(description)
	}
	canvas.SetMetadata(*metadata)
	request.SetCanvas(*canvas)

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(*request).Execute()
	if err != nil {
		return err
	}

	if httpResp != nil && (httpResp.StatusCode < 200 || httpResp.StatusCode >= 300) {
		return fmt.Errorf("unexpected response status: %s", httpResp.Status)
	}

	if resp == nil || resp.Canvas == nil || resp.Canvas.Metadata == nil || resp.Canvas.Metadata.GetId() == "" {
		return fmt.Errorf("failed to create canvas: the server returned an empty response")
	}

	canvasID := resp.Canvas.Metadata.GetId()
	if len(localFiles) == 0 {
		return printCreateResponse(ctx, *resp.Canvas, nil)
	}

	commitMessage, err := resolveCreateCommitMessage(c.message, name)
	if err != nil {
		return err
	}

	stagedFiles, err := prepareCreateRepositoryFiles(localFiles)
	if err != nil {
		return err
	}

	if err := common.StageRepositoryFiles(ctx, canvasID, stagedFiles); err != nil {
		return fmt.Errorf("app was created but staging failed: %w", err)
	}

	commitResponse, err := common.CommitCanvasStaging(ctx, canvasID, commitMessage)
	if err != nil {
		return fmt.Errorf("app was created but commit failed: %w", err)
	}

	return printCreateResponse(ctx, *resp.Canvas, commitResponse)
}

func resolveCreateCommitMessage(message *string, name string) (string, error) {
	if message != nil {
		trimmed := strings.TrimSpace(*message)
		if trimmed != "" {
			return trimmed, nil
		}
	}
	return fmt.Sprintf("Create %q", name), nil
}

func prepareCreateRepositoryFiles(localFiles []string) ([]common.RepositoryFileStaging, error) {
	if len(localFiles) == 0 {
		return nil, fmt.Errorf("at least one --file is required")
	}

	stagedFiles := make([]common.RepositoryFileStaging, 0, len(localFiles))
	seenPaths := make(map[string]struct{}, len(localFiles))

	for _, localPath := range localFiles {
		trimmedPath := strings.TrimSpace(localPath)
		if trimmedPath == "" {
			return nil, fmt.Errorf("file path is required")
		}

		content, err := os.ReadFile(trimmedPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read %s: %w", trimmedPath, err)
		}

		repositoryPath := common.RepositoryPathFromLocalFile(trimmedPath)
		if _, exists := seenPaths[repositoryPath]; exists {
			return nil, fmt.Errorf("duplicate repository file %q", repositoryPath)
		}
		seenPaths[repositoryPath] = struct{}{}

		switch repositoryPath {
		case common.CanvasYAMLRepositoryPath:
			_, err := yaml.CanvasFromYAML(content)
			if err != nil {
				return nil, fmt.Errorf("invalid canvas yaml: %w", err)
			}
		case common.ConsoleYAMLRepositoryPath:
			_, err := yaml.ConsoleFromYML(content)
			if err != nil {
				return nil, fmt.Errorf("invalid console yaml: %w", err)
			}
		}

		stagedFiles = append(stagedFiles, common.RepositoryFileStaging{
			Path:    repositoryPath,
			Content: content,
		})
	}

	return stagedFiles, nil
}

func printCreateResponse(
	ctx core.CommandContext,
	canvas openapi_client.CanvasesCanvas,
	commitResponse *openapi_client.CanvasesCommitCanvasStagingResponse,
) error {
	if !ctx.Renderer.IsText() {
		output := map[string]any{
			"canvas": canvas,
		}
		if commitResponse != nil && commitResponse.Version != nil {
			output["version"] = commitResponse.Version
		}
		return ctx.Renderer.Render(output)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if _, err := fmt.Fprintf(stdout, "App %q created (ID: %s)\n", canvas.Metadata.GetName(), canvas.Metadata.GetId()); err != nil {
			return err
		}
		if url := common.BuildAppURL(ctx, canvas.Metadata.GetOrganizationId(), canvas.Metadata.GetId()); url != "" {
			if _, err := fmt.Fprintf(stdout, "App URL: %s\n", url); err != nil {
				return err
			}
		}
		if commitResponse != nil && commitResponse.Version != nil && commitResponse.Version.Metadata != nil {
			if _, err := fmt.Fprintf(stdout, "Committed version: %s\n", commitResponse.Version.Metadata.GetId()); err != nil {
				return err
			}
		}
		return nil
	})
}

// NewCreateCommand registers app creation under `apps create`.
func NewCreateCommand(options core.BindOptions) *cobra.Command {
	var name string
	var description string
	var files []string
	var message string
	createCmd := &cobra.Command{
		Use:   "create",
		Short: "Create an app",
		Long: `Create an app by name and optionally commit repository files.

Examples:
  superplane apps create --name "My App"
  superplane apps create --name "My App" --file canvas.yaml --file console.yaml --file README.md

When --file is provided, the command creates an empty app, stages the files, and
commits them in one step. canvas.yaml and console.yaml do not need metadata.id
or metadata.canvasId beforehand; those fields are filled in automatically.

AI agents: for canonical canvas YAML shapes and wiring rules, install skills:
- ` + core.SkillsInstallCommand("superplane-app-builder") + `
- ` + core.SkillsInstallCommand("superplane-cli"),
	}
	createCmd.Flags().StringVar(&name, "name", "", "app name (required)")
	createCmd.Flags().StringVar(&description, "description", "", "app description")
	createCmd.Flags().StringArrayVar(&files, "file", nil, "local file to stage and commit (repeatable)")
	createCmd.Flags().StringVar(&message, "message", "", "commit message when --file is used (default: Create \"<name>\")")
	_ = createCmd.MarkFlagRequired("name")
	core.Bind(createCmd, &createCommand{
		name:        &name,
		description: &description,
		files:       &files,
		message:     &message,
	}, options)

	return createCmd
}
