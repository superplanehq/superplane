package widgets

import (
	"fmt"
	"io"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type addCommand struct {
	canvasID      *string
	component     *string
	name          *string
	configuration *string
	positionX     *int32
	positionY     *int32
	width         *int32
	height        *int32
	color         *string
	text          *string
	draft         *bool
}

// Execute adds a new TYPE_WIDGET node to the canvas spec.
//
// The implementation reuses the canvases change-management flow: it
// resolves (or creates) the user's draft version, mutates the spec to add
// the widget, and publishes the draft unless --draft is specified or the
// canvas requires explicit change requests.
func (c *addCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	component := valueOf(c.component)
	if component == "" {
		return fmt.Errorf("--component is required (e.g. annotation)")
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

	cfg, err := configurationFromInput(valueOf(c.configuration), ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}
	cfg = applyAnnotationShortcuts(cfg, valueOf(c.text), valueOf(c.color), int32Value(c.width), int32Value(c.height))

	posX, hasX := flagInt32IfChanged(ctx, "position-x", c.positionX)
	posY, hasY := flagInt32IfChanged(ctx, "position-y", c.positionY)
	node := buildWidgetNode(component, valueOf(c.name), cfg, posX, posY, hasX || hasY)

	canvas := canvasFromVersion(version)
	if canvas.Spec == nil {
		canvas.SetSpec(openapi_client.CanvasesCanvasSpec{})
	}
	spec := canvas.GetSpec()
	nodes := spec.GetNodes()
	nodes = append(nodes, node)
	spec.SetNodes(nodes)
	canvas.SetSpec(spec)

	updatedVersion, err := updateAndMaybePublish(ctx, canvasID, versionID, canvas, draftMode)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(node)
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, _ = fmt.Fprintf(stdout, "Widget added: %s (component: %s)\n", node.GetId(), component)
		metadata := updatedVersion.GetMetadata()
		_, err := fmt.Fprintf(stdout, "Canvas version: %s\n", metadata.GetId())
		return err
	})
}

// buildWidgetNode assembles the SuperplaneComponentsNode payload for a new
// widget. When --name is omitted we fall back to the component name so the
// node renders with a friendly label in the UI immediately after creation.
//
// Position is only attached when the caller explicitly provided a flag,
// signaled by hasPosition. This avoids stamping (0, 0) on every new node
// just because the position-x/y flags default to zero.
func buildWidgetNode(
	component, name string,
	configuration map[string]any,
	x, y int32,
	hasPosition bool,
) openapi_client.SuperplaneComponentsNode {
	node := openapi_client.SuperplaneComponentsNode{}
	node.SetId(uuid.NewString())
	if name == "" {
		name = component
	}
	node.SetName(name)
	widgetType := openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET
	node.Type = &widgetType
	node.SetComponent(component)
	if configuration != nil {
		node.SetConfiguration(configuration)
	}
	if hasPosition {
		position := openapi_client.ComponentsPosition{}
		position.SetX(x)
		position.SetY(y)
		node.SetPosition(position)
	}
	return node
}

func int32Value(p *int32) int32 {
	if p == nil {
		return 0
	}
	return *p
}

// flagInt32IfChanged returns the current flag value and a bool indicating
// whether the user explicitly passed it (cobra `Flags().Changed`). This
// keeps the "did the user set this?" check at execute time so commands
// don't apply default-zero positions or sizes by accident.
func flagInt32IfChanged(ctx core.CommandContext, flagName string, value *int32) (int32, bool) {
	v := int32Value(value)
	if ctx.Cmd == nil || ctx.Cmd.Flags() == nil {
		return v, false
	}
	return v, ctx.Cmd.Flags().Changed(flagName)
}
