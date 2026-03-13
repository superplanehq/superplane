package canvases

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file            *string
	draft           *bool
	autoLayout      *string
	autoLayoutScope *string
	autoLayoutNodes *[]string
}

func (c *updateCommand) Execute(ctx core.CommandContext) error {
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
	draftMode := c.draft != nil && *c.draft

	var (
		canvasID string
		canvas   openapi_client.CanvasesCanvas
		err      error
	)

	if filePath != "" {
		canvasID, canvas, err = loadCanvasFromFile(filePath)
		if err != nil {
			return err
		}
	} else {
		canvasID, canvas, err = loadCanvasFromExisting(ctx)
		if err != nil {
			return err
		}
	}

	versioningContext, err := resolveCanvasVersioningContext(ctx, canvasID)
	if err != nil {
		return err
	}

	targetVersionID := ""
	if !versioningContext.versioningEnabled {
		if draftMode {
			return fmt.Errorf("--draft cannot be used when effective canvas versioning is disabled")
		}
	} else {
		if !draftMode {
			return fmt.Errorf("effective canvas versioning is enabled for this canvas; use --draft")
		}

		targetVersionID, err = ensureCurrentUserDraftVersionID(ctx, canvasID)
		if err != nil {
			return err
		}
	}

	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)
	if targetVersionID != "" {
		body.SetVersionId(targetVersionID)
	}

	if autoLayoutFlagsWereSet(ctx) {
		autoLayout, parseErr := parseAutoLayout(autoLayoutValue, autoLayoutScopeValue, autoLayoutNodeIDs)
		if parseErr != nil {
			return parseErr
		}
		if autoLayout != nil {
			body.SetAutoLayout(*autoLayout)
		}
	} else {
		body.SetAutoLayout(buildDefaultAutoLayout())
	}

	_, _, err = ctx.API.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx.Context, canvasID).
		Body(body).
		Execute()
	return err
}

func loadCanvasFromExisting(ctx core.CommandContext) (string, openapi_client.CanvasesCanvas, error) {
	if len(ctx.Args) > 1 {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("update accepts at most one positional argument")
	}

	target := ""
	if len(ctx.Args) == 1 {
		target = ctx.Args[0]
	} else if ctx.Config != nil {
		target = strings.TrimSpace(ctx.Config.GetActiveCanvas())
	}

	if target == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("either --file or <name-or-id> (or an active canvas) is required")
	}

	canvasID, err := findCanvasID(ctx, ctx.API, target)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	canvas, err := describeCanvasByID(ctx, canvasID)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	return canvasID, canvas, nil
}

func parseAutoLayout(value string, scopeValue string, nodeIDs []string) (*openapi_client.CanvasesCanvasAutoLayout, error) {
	normalizedValue := strings.ToLower(strings.TrimSpace(value))
	switch normalizedValue {
	case "disable", "disabled", "none", "off":
		if strings.TrimSpace(scopeValue) != "" || len(nodeIDs) > 0 {
			return nil, fmt.Errorf("--auto-layout-scope and --auto-layout-node cannot be used when --auto-layout disables layout")
		}
		return nil, nil
	}

	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}

	switch normalizedValue {
	case "", "horizontal":
		autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	default:
		return nil, fmt.Errorf("unsupported auto layout %q (supported: horizontal, disable)", value)
	}

	normalizedNodeIDs := make([]string, 0, len(nodeIDs))
	seen := make(map[string]struct{}, len(nodeIDs))
	for _, nodeID := range nodeIDs {
		trimmed := strings.TrimSpace(nodeID)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		normalizedNodeIDs = append(normalizedNodeIDs, trimmed)
	}
	if len(normalizedNodeIDs) > 0 {
		autoLayout.SetNodeIds(normalizedNodeIDs)
	}

	if strings.TrimSpace(scopeValue) == "" {
		return &autoLayout, nil
	}

	switch strings.ToLower(strings.TrimSpace(scopeValue)) {
	case "full-canvas", "full_canvas", "full":
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	case "connected-component", "connected_component", "connected":
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_CONNECTED_COMPONENT)
	default:
		return nil, fmt.Errorf("unsupported auto layout scope %q (supported: full-canvas, connected-component)", scopeValue)
	}

	return &autoLayout, nil
}

func autoLayoutFlagsWereSet(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("auto-layout") || flags.Changed("auto-layout-scope") || flags.Changed("auto-layout-node")
}

func describeCanvasByID(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if response.Canvas == nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", canvasID)
	}

	return *response.Canvas, nil
}

func buildDefaultAutoLayout() openapi_client.CanvasesCanvasAutoLayout {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	return autoLayout
}
