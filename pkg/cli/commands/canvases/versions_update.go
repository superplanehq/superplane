package canvases

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type versionsUpdateCommand struct {
	canvas          *string
	file            *string
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *versionsUpdateCommand) Execute(ctx core.CommandContext) error {
	filePath := ""
	if c.file != nil {
		filePath = strings.TrimSpace(*c.file)
	}
	if filePath == "" {
		return fmt.Errorf("--file is required")
	}

	canvasIDFromFile, canvas, err := loadCanvasFromFile(filePath)
	if err != nil {
		return err
	}

	canvasRef := ""
	if c.canvas != nil {
		canvasRef = *c.canvas
	}

	canvasID := canvasIDFromFile
	if strings.TrimSpace(canvasRef) != "" {
		resolvedCanvasID, resolveErr := resolveCanvasIDFromArgOrActive(ctx, canvasRef)
		if resolveErr != nil {
			return resolveErr
		}
		if resolvedCanvasID != canvasIDFromFile {
			return fmt.Errorf("canvas id from --canvas (%s) does not match metadata.id in --file (%s)", resolvedCanvasID, canvasIDFromFile)
		}
		canvasID = resolvedCanvasID
	}

	versionRef := ""
	if len(ctx.Args) == 1 {
		versionRef = ctx.Args[0]
	}

	trimmedVersionRef := strings.TrimSpace(versionRef)
	versionID := ""
	targetLiveVersion := false
	switch {
	case strings.EqualFold(trimmedVersionRef, "live"):
		targetLiveVersion = true
	case trimmedVersionRef != "":
		versionID = trimmedVersionRef
	default:
		if ctx.Config != nil {
			activeVersion := strings.TrimSpace(ctx.Config.GetActiveCanvasVersion())
			if activeVersion != "" {
				versionID = activeVersion
				break
			}
		}

		resolvedVersionID, resolveErr := resolveOrCreateEditVersionID(ctx, canvasID)
		if resolveErr != nil {
			return resolveErr
		}
		versionID = resolvedVersionID
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

	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)

	if autoLayoutFlagsWereSet(ctx) {
		if autoLayoutValue == "" && (autoLayoutScopeValue != "" || len(autoLayoutNodeIDs) > 0) {
			return fmt.Errorf("--auto-layout is required when using --auto-layout-scope or --auto-layout-node")
		}
		if autoLayoutValue != "" {
			autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
			if parseErr != nil {
				return parseErr
			}
			body.SetAutoLayout(*autoLayout)
		}
	} else {
		currentCanvas, describeErr := describeCanvasByID(ctx, canvasID)
		if describeErr != nil {
			return describeErr
		}
		body.SetAutoLayout(buildDefaultAutoLayout(currentCanvas, canvas))
	}

	var response *openapi_client.CanvasesUpdateCanvasVersionResponse
	if targetLiveVersion {
		response, _, err = ctx.API.CanvasVersionAPI.
			CanvasesUpdateCanvasVersion2(ctx.Context, canvasID).
			Body(body).
			Execute()
	} else {
		response, _, err = ctx.API.CanvasVersionAPI.
			CanvasesUpdateCanvasVersion(ctx.Context, canvasID, versionID).
			Body(body).
			Execute()
	}
	if err != nil {
		return err
	}

	if response.Version == nil || response.Version.Metadata == nil {
		return fmt.Errorf("failed to update canvas version")
	}

	activeVersion := versionID
	if targetLiveVersion {
		activeVersion = ""
	}

	if err := setActiveCanvasAndVersion(ctx, canvasID, activeVersion); err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.Version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := response.Version.GetMetadata()
		_, _ = fmt.Fprintf(stdout, "Canvas: %s\n", canvasID)
		if targetLiveVersion {
			_, _ = fmt.Fprintf(stdout, "Updated live version: %s\n", metadata.GetId())
		} else {
			_, _ = fmt.Fprintf(stdout, "Updated edit version: %s\n", metadata.GetId())
		}
		_, _ = fmt.Fprintf(stdout, "Revision: %d\n", metadata.GetRevision())
		_, err = fmt.Fprintln(stdout, "Active context updated")
		return err
	})
}
