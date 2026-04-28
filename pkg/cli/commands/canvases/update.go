package canvases

import (
	"fmt"
	"io"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/commands/canvases/models"
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

// resolveCanvasForFileUpdate parses --file and determines the target canvas id.
// If metadata.id is set in the file, it is authoritative; an optional positional
// name-or-id must resolve to the same id. If metadata.id is omitted, the canvas is
// resolved from the positional argument or the configured active canvas.
func resolveCanvasForFileUpdate(ctx core.CommandContext, filePath string) (string, openapi_client.CanvasesCanvas, error) {
	resource, err := parseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if len(ctx.Args) > 1 {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("update accepts at most one optional canvas name or id")
	}

	positional := ""
	if len(ctx.Args) == 1 {
		positional = strings.TrimSpace(ctx.Args[0])
	}

	fileID := ""
	if resource.Metadata != nil && resource.Metadata.Id != nil {
		fileID = strings.TrimSpace(resource.Metadata.GetId())
	}

	var canvasID string

	switch {
	case fileID != "":
		canvasID = fileID
		if positional != "" {
			resolved, ferr := findCanvasID(ctx, ctx.API, positional)
			if ferr != nil {
				return "", openapi_client.CanvasesCanvas{}, ferr
			}
			if !strings.EqualFold(strings.TrimSpace(resolved), strings.TrimSpace(fileID)) {
				return "", openapi_client.CanvasesCanvas{}, fmt.Errorf(
					"canvas file metadata.id %q does not match argument %q (resolved id %q)",
					fileID, positional, resolved,
				)
			}
		}
	case positional != "":
		var ferr error
		canvasID, ferr = findCanvasID(ctx, ctx.API, positional)
		if ferr != nil {
			return "", openapi_client.CanvasesCanvas{}, ferr
		}
	default:
		if ctx.Config == nil {
			return "", openapi_client.CanvasesCanvas{}, fmt.Errorf(
				"canvas metadata.id is empty: pass a canvas name or id, or set an active canvas with `superplane canvases active`",
			)
		}
		active := strings.TrimSpace(ctx.Config.GetActiveCanvas())
		if active == "" {
			return "", openapi_client.CanvasesCanvas{}, fmt.Errorf(
				"canvas metadata.id is empty: pass a canvas name or id, or set an active canvas with `superplane canvases active`",
			)
		}
		var ferr error
		canvasID, ferr = findCanvasID(ctx, ctx.API, active)
		if ferr != nil {
			return "", openapi_client.CanvasesCanvas{}, ferr
		}
	}

	if resource.Metadata == nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata is required")
	}
	resource.Metadata.SetId(canvasID)
	canvas := models.CanvasFromCanvas(*resource)
	return canvasID, canvas, nil
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

	canvasID, canvas, err := resolveCanvasForFileUpdate(ctx, filePath)
	if err != nil {
		return err
	}

	cmContext, err := resolveChangeManagementContext(ctx, canvasID)
	if err != nil {
		return err
	}

	if cmContext.changeManagementEnabled && !draftMode {
		return fmt.Errorf("change management is enabled for this canvas; use --draft to update your draft version, then publish with `superplane canvases change-requests create`")
	}

	targetVersionID, err := ensureCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return err
	}

	body := openapi_client.CanvasesUpdateCanvasVersionBody{}
	body.SetCanvas(canvas)
	body.SetVersionId(targetVersionID)

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

	response, _, err := ctx.API.CanvasVersionAPI.
		CanvasesUpdateCanvasVersion2(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	version := response.GetVersion()

	// When not in draft mode, auto-publish the updated draft version.
	if !draftMode {
		_, _, publishErr := ctx.API.CanvasVersionAPI.
			CanvasesPublishCanvasVersion(ctx.Context, canvasID, targetVersionID).
			Body(map[string]any{}).
			Execute()
		if publishErr != nil {
			return fmt.Errorf("draft was updated but publish failed: %w", publishErr)
		}
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(version)
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		metadata := version.GetMetadata()
		spec := version.GetSpec()

		_, _ = fmt.Fprintf(stdout, "Canvas version updated: %s\n", metadata.GetId())
		_, _ = fmt.Fprintf(stdout, "Canvas ID: %s\n", metadata.GetCanvasId())
		_, _ = fmt.Fprintf(stdout, "Nodes: %d\n", len(spec.GetNodes()))
		_, _ = fmt.Fprintf(stdout, "Edges: %d\n", len(spec.GetEdges()))

		integrations := make(map[string]struct{})
		for _, node := range spec.GetNodes() {
			if ref, ok := node.GetIntegrationOk(); ok && ref != nil {
				if id := ref.GetId(); id != "" {
					integrations[id] = struct{}{}
				}
			}
		}
		_, err := fmt.Fprintf(stdout, "Integrations: %d\n", len(integrations))
		return err
	})
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

func buildDefaultAutoLayout() openapi_client.CanvasesCanvasAutoLayout {
	autoLayout := openapi_client.CanvasesCanvasAutoLayout{}
	autoLayout.SetAlgorithm(openapi_client.CANVASAUTOLAYOUTALGORITHM_ALGORITHM_HORIZONTAL)
	autoLayout.SetScope(openapi_client.CANVASAUTOLAYOUTSCOPE_SCOPE_FULL_CANVAS)
	return autoLayout
}
