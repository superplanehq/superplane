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

func resolveCanvasForFileUpdate(filePath string) (string, openapi_client.CanvasesCanvas, error) {
	resource, err := parseCanvasResourceFromFile(filePath, "update")
	if err != nil {
		return "", openapi_client.CanvasesCanvas{}, err
	}

	if resource.Metadata == nil {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata is required")
	}

	fileID := ""
	if resource.Metadata.Id != nil {
		fileID = strings.TrimSpace(resource.Metadata.GetId())
	}

	if fileID == "" {
		return "", openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas metadata.id is required in the YAML file")
	}

	canvas := models.CanvasFromCanvas(*resource)
	return fileID, canvas, nil
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

	canvasID, canvas, err := resolveCanvasForFileUpdate(filePath)
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
	if errText := formatNodeSpecErrorsForCLI(version); errText != "" {
		return fmt.Errorf("%s", errText)
	}

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
		if err != nil {
			return err
		}
		if warnText := formatNodeSpecWarningsForCLI(version); warnText != "" {
			_, err = fmt.Fprint(stdout, warnText)
		}
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

// formatNodeSpecErrorsForCLI summarizes node error_message from the API response (blocks execution until fixed).
func formatNodeSpecErrorsForCLI(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasErrorMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetErrorMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("canvas was saved but the following nodes have configuration errors (error_message on each node):\n")
	for _, line := range lines {
		b.WriteString("  - ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}

func formatNodeSpecWarningsForCLI(version openapi_client.CanvasesCanvasVersion) string {
	spec, ok := version.GetSpecOk()
	if !ok || spec == nil {
		return ""
	}

	var lines []string
	for _, node := range spec.GetNodes() {
		if !node.HasWarningMessage() {
			continue
		}
		msg := strings.TrimSpace(node.GetWarningMessage())
		if msg == "" {
			continue
		}
		id := node.GetId()
		name := strings.TrimSpace(node.GetName())
		if name == "" {
			name = id
		}
		lines = append(lines, fmt.Sprintf("node %s (%s): %s", id, name, msg))
	}
	if len(lines) == 0 {
		return ""
	}

	var b strings.Builder
	b.WriteString("\nNode warnings (warning_message):\n")
	for _, line := range lines {
		b.WriteString("  - ")
		b.WriteString(line)
		b.WriteByte('\n')
	}
	return b.String()
}
