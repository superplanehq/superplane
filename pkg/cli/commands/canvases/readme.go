package canvases

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type readmeGetCommand struct {
	draft   *bool
	version *string
}

func (c *readmeGetCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("readme get accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	request := ctx.API.CanvasAPI.CanvasesGetCanvasReadme(ctx.Context, canvasID)

	draftRequested := c.draft != nil && *c.draft
	versionID := ""
	if c.version != nil {
		versionID = strings.TrimSpace(*c.version)
	}

	if draftRequested && versionID != "" {
		return fmt.Errorf("--draft and --version cannot be used together")
	}

	switch {
	case draftRequested:
		request = request.VersionId("draft")
	case versionID != "":
		request = request.VersionId(versionID)
	}

	response, _, err := request.Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprint(stdout, response.GetContent())
		return err
	})
}

type readmeUpdateCommand struct {
	file      *string
	content   *string
	message   *string
	draftOnly *bool
}

func (c *readmeUpdateCommand) Execute(ctx core.CommandContext) error {
	if len(ctx.Args) > 1 {
		return fmt.Errorf("readme update accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = strings.TrimSpace(ctx.Args[0])
	}

	canvasID, err := resolveCanvasTargetFromOptionalArg(ctx, target)
	if err != nil {
		return err
	}

	content, err := readReadmeContent(c.file, c.content)
	if err != nil {
		return err
	}

	// Ensure there's a draft we own; the backend would auto-create one on
	// update, but calling ensureCurrentUserDraftVersionID keeps the flow
	// explicit and returns a version id we can use for CR creation.
	versionID, err := ensureCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return err
	}

	body := openapi_client.NewCanvasesUpdateCanvasReadmeBody()
	body.SetVersionId(versionID)
	body.SetContent(content)

	_, _, err = ctx.API.CanvasAPI.
		CanvasesUpdateCanvasReadme(ctx.Context, canvasID).
		Body(*body).
		Execute()
	if err != nil {
		return err
	}

	draftOnly := c.draftOnly != nil && *c.draftOnly

	cmContext, err := resolveChangeManagementContext(ctx, canvasID)
	if err != nil {
		return err
	}

	if draftOnly {
		return renderReadmeUpdateResult(ctx, canvasID, versionID, "draft")
	}

	if cmContext.changeManagementEnabled {
		title := "Update README"
		if c.message != nil {
			trimmed := strings.TrimSpace(*c.message)
			if trimmed != "" {
				title = trimmed
			}
		}

		crBody := openapi_client.CanvasesCreateCanvasChangeRequestBody{}
		crBody.SetVersionId(versionID)
		crBody.SetTitle(title)

		_, _, crErr := ctx.API.CanvasChangeRequestAPI.
			CanvasesCreateCanvasChangeRequest(ctx.Context, canvasID).
			Body(crBody).
			Execute()
		if crErr != nil {
			return fmt.Errorf("readme was saved on the draft, but change request creation failed: %w", crErr)
		}

		return renderReadmeUpdateResult(ctx, canvasID, versionID, "change-request")
	}

	// Change management disabled: publish the draft so the readme goes live.
	_, _, publishErr := ctx.API.CanvasVersionAPI.
		CanvasesPublishCanvasVersion(ctx.Context, canvasID, versionID).
		Body(map[string]any{}).
		Execute()
	if publishErr != nil {
		return fmt.Errorf("readme was saved on the draft, but publish failed: %w", publishErr)
	}

	return renderReadmeUpdateResult(ctx, canvasID, versionID, "published")
}

func readReadmeContent(filePtr *string, contentPtr *string) (string, error) {
	file := ""
	if filePtr != nil {
		file = strings.TrimSpace(*filePtr)
	}
	content := ""
	hasContentFlag := contentPtr != nil
	if contentPtr != nil {
		content = *contentPtr
	}

	if file != "" && hasContentFlag && content != "" {
		return "", fmt.Errorf("--file and --content cannot be used together")
	}

	if file != "" {
		if file == "-" {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return "", fmt.Errorf("failed to read readme from stdin: %w", err)
			}
			return string(data), nil
		}

		// #nosec
		data, err := os.ReadFile(file)
		if err != nil {
			return "", fmt.Errorf("failed to read readme file %q: %w", file, err)
		}
		return string(data), nil
	}

	if hasContentFlag {
		return content, nil
	}

	return "", fmt.Errorf("either --file/-f or --content is required")
}

func renderReadmeUpdateResult(ctx core.CommandContext, canvasID, versionID, outcome string) error {
	if !ctx.Renderer.IsText() {
		summary := map[string]string{
			"canvasId":  canvasID,
			"versionId": versionID,
			"outcome":   outcome,
		}
		return ctx.Renderer.Render(summary)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		switch outcome {
		case "draft":
			_, err := fmt.Fprintf(stdout, "Readme saved to draft %s\n", versionID)
			return err
		case "change-request":
			_, err := fmt.Fprintf(stdout, "Readme saved to draft %s; change request created\n", versionID)
			return err
		case "published":
			_, err := fmt.Fprintf(stdout, "Readme published (version %s)\n", versionID)
			return err
		default:
			_, err := fmt.Fprintf(stdout, "Readme updated (version %s)\n", versionID)
			return err
		}
	})
}
