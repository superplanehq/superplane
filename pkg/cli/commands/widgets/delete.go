package widgets

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type deleteCommand struct {
	canvasID *string
	yes      *bool
	draft    *bool
}

// Execute removes a widget node from the canvas spec, including any edges
// that point at it (none today since widget nodes are decorative, but the
// pruning is harmless and keeps the spec clean if that ever changes).
func (c *deleteCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	cmEnabled, err := changeManagementEnabled(ctx, canvasID)
	if err != nil {
		return err
	}
	draftMode := c.draft != nil && *c.draft
	if cmEnabled && !draftMode {
		return fmt.Errorf("change management is enabled for this canvas; pass --draft and publish via `superplane canvases change-requests`")
	}

	versionID, err := ensureCurrentUserDraftVersionID(ctx, canvasID)
	if err != nil {
		return err
	}
	version, err := describeCanvasVersionByID(ctx, canvasID, versionID)
	if err != nil {
		return err
	}
	canvas := canvasFromVersion(version)
	if canvas.Spec == nil {
		return fmt.Errorf("canvas has no spec to update")
	}

	target, err := findWidgetNode(canvas, ctx.Args[0])
	if err != nil {
		return err
	}

	if !confirmDelete(ctx, c.yes, target.GetId()) {
		_, err := fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Aborted.")
		return err
	}

	spec := canvas.GetSpec()
	updatedNodes := make([]openapi_client.SuperplaneComponentsNode, 0, len(spec.GetNodes()))
	for _, node := range spec.GetNodes() {
		if node.GetId() == target.GetId() {
			continue
		}
		updatedNodes = append(updatedNodes, node)
	}

	updatedEdges := make([]openapi_client.SuperplaneComponentsEdge, 0, len(spec.GetEdges()))
	for _, edge := range spec.GetEdges() {
		if edge.GetSourceId() == target.GetId() || edge.GetTargetId() == target.GetId() {
			continue
		}
		updatedEdges = append(updatedEdges, edge)
	}

	spec.SetNodes(updatedNodes)
	spec.SetEdges(updatedEdges)
	canvas.SetSpec(spec)

	updatedVersion, err := updateAndMaybePublish(ctx, canvasID, versionID, canvas, draftMode)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"id":      target.GetId(),
			"deleted": true,
		})
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Widget deleted: %s\n", target.GetId())
		metadata := updatedVersion.GetMetadata()
		_, err := fmt.Fprintf(stdout, "Canvas version: %s\n", metadata.GetId())
		return err
	})
}

func confirmDelete(ctx core.CommandContext, yes *bool, id string) bool {
	if yes != nil && *yes {
		return true
	}
	if !ctx.IsInteractive() {
		return true
	}
	_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Delete widget %q? [y/N]: ", id)
	var answer string
	_, _ = fmt.Fscanln(ctx.Cmd.InOrStdin(), &answer)
	return answer == "y" || answer == "Y" || answer == "yes" || answer == "YES"
}
