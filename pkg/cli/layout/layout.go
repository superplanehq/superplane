package layout

import (
	"fmt"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

func ParseAutoLayout(value string, scopeValue string, nodeIDs []string) (*openapi_client.CanvasesCanvasAutoLayout, error) {
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

func HasFlags(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("auto-layout") || flags.Changed("auto-layout-scope") || flags.Changed("auto-layout-node")
}

func HasCanvasFlags(ctx core.CommandContext) bool {
	if ctx.Cmd == nil {
		return false
	}

	flags := ctx.Cmd.Flags()
	if flags == nil {
		return false
	}

	return flags.Changed("canvas-auto-layout") || flags.Changed("canvas-auto-layout-scope") || flags.Changed("canvas-auto-layout-node")
}

func DefaultAutoLayout() openapi_client.CanvasesCanvasAutoLayout {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	return autoLayout
}
