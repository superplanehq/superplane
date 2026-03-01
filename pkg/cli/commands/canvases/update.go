package canvases

import (
	"fmt"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	file            *string
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

	body := openapi_client.CanvasesUpdateCanvasBody{}
	body.SetCanvas(canvas)

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

	_, _, err = ctx.API.CanvasAPI.
		CanvasesUpdateCanvas(ctx.Context, canvasID).
		Body(body).
		Execute()
	return err
}

func loadCanvasFromFile(filePath string) (string, openapi_client.CanvasesCanvas, error) {
	// #nosec
	data, err := os.ReadFile(filePath)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("failed to read resource file: %w", err)
	}

	_, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if kind != models.CanvasKind {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("unsupported resource kind %q for update", kind)
	}

	resource, err := models.ParseCanvas(data)
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}
	if resource.Metadata == nil || resource.Metadata.Id == nil || resource.Metadata.GetId() == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata.id is required for update")
	}

	return resource.Metadata.GetId(), models.CanvasFromCanvas(*resource), nil
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

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}
	if response.Canvas == nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", target)
	}

	return canvasID, *response.Canvas, nil
}

func parseAutoLayout(value string, scopeValue string, nodeIDs []string) (*openapi_client.CanvasesCanvasAutoLayout, error) {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}

	switch strings.ToLower(strings.TrimSpace(value)) {
	case "horizontal":
		autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	default:
		return nil, fmt.Errorf("unsupported auto layout %q (supported: horizontal)", value)
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
	case "exact-set", "exact_set", "exact":
		autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_EXACT_SET)
	default:
		return nil, fmt.Errorf("unsupported auto layout scope %q (supported: full-canvas, connected-component, exact-set)", scopeValue)
	}

	return &autoLayout, nil
}
