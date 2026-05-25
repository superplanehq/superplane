package widgets

import (
	"fmt"
	"io"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type updateCommand struct {
	canvasID      *string
	configuration *string
	name          *string
	positionX     *int32
	positionY     *int32
	width         *int32
	height        *int32
	color         *string
	text          *string
	draft         *bool
}

// Execute updates an existing widget node's configuration, name, or
// position. Configuration changes are merge-on-top: the new keys override
// existing keys but keys absent from the input are preserved so users can
// tweak a single field (e.g. annotation text) without rewriting the rest.
func (c *updateCommand) Execute(ctx core.CommandContext) error {
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

	overrides, err := configurationFromInput(valueOf(c.configuration), ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	width, _ := flagInt32IfChanged(ctx, "width", c.width)
	height, _ := flagInt32IfChanged(ctx, "height", c.height)
	overrides = applyAnnotationShortcuts(overrides, valueOf(c.text), valueOf(c.color), width, height)

	mutated := mergeConfiguration(target.GetConfiguration(), overrides)
	target.SetConfiguration(mutated)

	if name := valueOf(c.name); name != "" {
		target.SetName(name)
	}
	posX, hasX := flagInt32IfChanged(ctx, "position-x", c.positionX)
	posY, hasY := flagInt32IfChanged(ctx, "position-y", c.positionY)
	if hasX || hasY {
		current := openapi_client.ComponentsPosition{}
		if target.Position != nil {
			current = *target.Position
		}
		if hasX {
			current.SetX(posX)
		}
		if hasY {
			current.SetY(posY)
		}
		target.SetPosition(current)
	}

	spec := canvas.GetSpec()
	nodes := spec.GetNodes()
	for i, node := range nodes {
		if node.GetId() == target.GetId() {
			nodes[i] = target
			break
		}
	}
	spec.SetNodes(nodes)
	canvas.SetSpec(spec)

	updatedVersion, err := updateAndMaybePublish(ctx, canvasID, versionID, canvas, draftMode)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(target)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Widget updated: %s\n", target.GetId())
		metadata := updatedVersion.GetMetadata()
		_, err := fmt.Fprintf(stdout, "Canvas version: %s\n", metadata.GetId())
		return err
	})
}

// mergeConfiguration overlays new values on top of the existing
// configuration. We prefer merge-over-replace because users typically want
// to change one field without restating the rest of the widget config.
func mergeConfiguration(existing, overrides map[string]any) map[string]any {
	if len(existing) == 0 && len(overrides) == 0 {
		return nil
	}
	out := map[string]any{}
	for k, v := range existing {
		out[k] = v
	}
	for k, v := range overrides {
		out[k] = v
	}
	return out
}
