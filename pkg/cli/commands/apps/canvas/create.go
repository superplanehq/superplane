package canvas

import (
	"fmt"
	"io"
	"net/http"
	"strings"

	"github.com/spf13/cobra"
	"github.com/superplanehq/superplane/pkg/cli/commands/apps/canvas/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/cli/layout"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type createCommand struct {
	file            *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *createCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = *c.file
	}
	autoLayoutValue := ""
	if c.autoLayout != nil {
		autoLayoutValue = strings.TrimSpace(*c.autoLayout)
	}
	autoLayoutScopeValue := ""
	if c.autoLayoutScope != nil {
		autoLayoutScopeValue = strings.TrimSpace(*c.autoLayoutScope)
	}
	autoLayoutNodeIDs := []string{}
	if c.autoLayoutNodes != nil {
		autoLayoutNodeIDs = append(autoLayoutNodeIDs, *c.autoLayoutNodes...)
	}

	if filePath != "" {
		if len(ctx.Args) > 0 {
			return fmt.Errorf("cannot use <app-name> together with --file")
		}
		return c.createFromFile(ctx, filePath, autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
	}

	if len(ctx.Args) != 1 {
		return fmt.Errorf("either --file or <app-name> is required")
	}

	name := ctx.Args[0]
	resource := models.Canvas{
		APIVersion: core.APIVersion,
		Kind:       models.CanvasKind,
		Metadata:   &openapi_client.CanvasesCanvasMetadata{Name: &name},
		Spec:       models.EmptyCanvasSpec(),
	}

	request := models.CreateCanvasRequestFromCanvas(resource)
	if layout.HasFlags(ctx) {
		autoLayout, parseErr := layout.ParseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if autoLayout != nil {
			request.SetAutoLayout(*autoLayout)
		}
	} else {
		request.SetAutoLayout(buildDefaultAutoLayout())
	}

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
	return validateAndPrintCreateResponse(ctx, resp, httpResp, err)
}

func (c *createCommand) createFromFile(
	ctx core.CommandContext,
	path string,
	autoLayoutValue string,
	autoLayoutScopeValue string,
	autoLayoutNodeIDs []string,
) error {
	canvas, fileAutoLayout, err := loadCanvasForCreateFromFile(path)
	if err != nil {
		return err
	}

	request := openapi_client.CanvasesCreateCanvasRequest{}
	request.SetCanvas(canvas)

	if layout.HasFlags(ctx) {
		autoLayout, parseErr := layout.ParseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if autoLayout != nil {
			if fileAutoLayout != nil {
				return fmt.Errorf("cannot use auto-layout flags with --file when file already defines autoLayout")
			}
			request.SetAutoLayout(*autoLayout)
		}
	} else {
		if fileAutoLayout != nil {
			request.SetAutoLayout(*fileAutoLayout)
		} else {
			request.SetAutoLayout(buildDefaultAutoLayout())
		}
	}

	resp, httpResp, err := ctx.API.CanvasAPI.CanvasesCreateCanvas(ctx.Context).Body(request).Execute()
	return validateAndPrintCreateResponse(ctx, resp, httpResp, err)
}

func validateAndPrintCreateResponse(
	ctx core.CommandContext,
	resp *openapi_client.CanvasesCreateCanvasResponse,
	httpResp *http.Response,
	err error,
) error {
	if err != nil {
		return err
	}

	if httpResp != nil && (httpResp.StatusCode < 200 || httpResp.StatusCode >= 300) {
		return fmt.Errorf("unexpected response status: %s", httpResp.Status)
	}

	if resp == nil || resp.Canvas == nil || resp.Canvas.Metadata == nil || resp.Canvas.Metadata.GetId() == "" {
		return fmt.Errorf("failed to create canvas: the server returned an empty response")
	}

	canvas := *resp.Canvas
	resource := models.CanvasResourceFromCanvas(canvas)
	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(resource)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		if _, err := fmt.Fprintf(stdout, "App %q created (ID: %s)\n", canvas.Metadata.GetName(), canvas.Metadata.GetId()); err != nil {
			return err
		}
		if url := BuildCanvasURL(ctx, canvas.Metadata.GetOrganizationId(), canvas.Metadata.GetId()); url != "" {
			if _, err := fmt.Fprintf(stdout, "App URL: %s\n", url); err != nil {
				return err
			}
		}
		return nil
	})
}

// NewCreateCommand registers app creation under `apps create`.
func NewCreateCommand(options core.BindOptions) *cobra.Command {
	var createFile string
	var createAutoLayout string
	var createAutoLayoutScope string
	var createAutoLayoutNodes []string
	createCmd := &cobra.Command{
		Use:   "create [app-name]",
		Short: "Create an app",
		Long: `Create an app by name or from a canvas YAML file.

AI agents: for canonical canvas YAML shapes and wiring rules, install skills:
- ` + core.SkillsInstallCommand("superplane-app-builder") + `
- ` + core.SkillsInstallCommand("superplane-cli"),
		Args: cobra.MaximumNArgs(1),
	}
	createCmd.Flags().StringVarP(&createFile, "file", "f", "", "filename, directory, or URL to files to use to create the resource")
	createCmd.Flags().StringVar(&createAutoLayout, "auto-layout", "", "automatically arrange the canvas (supported: horizontal, disable)")
	createCmd.Flags().StringVar(&createAutoLayoutScope, "auto-layout-scope", "", "scope for auto layout (full-canvas, connected-component)")
	createCmd.Flags().StringArrayVar(&createAutoLayoutNodes, "auto-layout-node", nil, "node id seed for auto layout (repeatable)")
	core.Bind(createCmd, &createCommand{
		file:            &createFile,
		autoLayout:      &createAutoLayout,
		autoLayoutScope: &createAutoLayoutScope,
		autoLayoutNodes: &createAutoLayoutNodes,
	}, options)

	return createCmd
}
