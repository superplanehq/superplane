package widgets

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// canvasNodeIsWidget returns true when the given node is a widget node
// (TYPE_WIDGET). Widget nodes are the canvas-side counterpart to the
// widgets registered through `pkg/widgets/<name>` — for example annotation
// notes are TYPE_WIDGET nodes referencing the `annotation` component.
func canvasNodeIsWidget(node openapi_client.SuperplaneComponentsNode) bool {
	if node.Type == nil {
		return false
	}
	return *node.Type == openapi_client.COMPONENTSNODETYPE_TYPE_WIDGET
}

// fetchCanvas loads the latest published canvas spec for read-only commands.
// Mutations should pull the user's draft via the change-management helpers
// instead so they don't observe a stale view.
func fetchCanvas(ctx core.CommandContext, canvasID string) (openapi_client.CanvasesCanvas, error) {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return openapi_client.CanvasesCanvas{}, err
	}
	if response.Canvas == nil {
		return openapi_client.CanvasesCanvas{}, fmt.Errorf("canvas %q not found", canvasID)
	}
	return *response.Canvas, nil
}

// findWidgetNode locates a TYPE_WIDGET node by id or name. Names are case
// sensitive but unambiguous: if more than one widget node shares a name the
// caller gets a clear error rather than silently picking one.
func findWidgetNode(canvas openapi_client.CanvasesCanvas, idOrName string) (openapi_client.SuperplaneComponentsNode, error) {
	idOrName = strings.TrimSpace(idOrName)
	if idOrName == "" {
		return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("widget id or name is required")
	}
	if canvas.Spec == nil {
		return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("canvas has no spec")
	}

	spec := *canvas.Spec
	matches := []openapi_client.SuperplaneComponentsNode{}
	for _, node := range spec.GetNodes() {
		if !canvasNodeIsWidget(node) {
			continue
		}
		if node.GetId() == idOrName || node.GetName() == idOrName {
			matches = append(matches, node)
		}
	}

	if len(matches) == 0 {
		return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("widget %q not found", idOrName)
	}
	if len(matches) > 1 {
		return openapi_client.SuperplaneComponentsNode{}, fmt.Errorf("multiple widgets match %q", idOrName)
	}
	return matches[0], nil
}

// configurationFromInput resolves a widget configuration map from one of
// three sources: --configuration inline JSON, --configuration @file, or the
// stdin sentinel `-`. The empty string is returned as nil so callers know
// no override was provided.
func configurationFromInput(input string, stdin io.Reader) (map[string]any, error) {
	trimmed := strings.TrimSpace(input)
	if trimmed == "" {
		return nil, nil
	}

	var raw []byte
	switch {
	case trimmed == "-":
		var err error
		raw, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration from stdin: %w", err)
		}
	case strings.HasPrefix(trimmed, "@"):
		path := strings.TrimPrefix(trimmed, "@")
		// #nosec G304 - path is supplied by the CLI user.
		data, err := os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read configuration file %s: %w", path, err)
		}
		raw = data
	default:
		raw = []byte(trimmed)
	}

	if len(raw) == 0 {
		return nil, nil
	}

	out := map[string]any{}
	if err := json.Unmarshal(raw, &out); err != nil {
		return nil, fmt.Errorf("invalid JSON configuration: %w", err)
	}
	return out, nil
}

// applyAnnotationShortcuts merges the annotation-specific convenience flags
// (--text, --color, --width, --height) into the configuration map. Values
// from --configuration win over individual flags so callers can use both
// without surprises (the JSON is treated as the explicit source of truth).
func applyAnnotationShortcuts(cfg map[string]any, text, color string, width, height int32) map[string]any {
	if cfg == nil {
		cfg = map[string]any{}
	}
	if text != "" {
		if _, ok := cfg["text"]; !ok {
			cfg["text"] = text
		}
	}
	if color != "" {
		if _, ok := cfg["color"]; !ok {
			cfg["color"] = color
		}
	}
	if width > 0 {
		if _, ok := cfg["width"]; !ok {
			cfg["width"] = width
		}
	}
	if height > 0 {
		if _, ok := cfg["height"]; !ok {
			cfg["height"] = height
		}
	}
	return cfg
}

// renderNodeText prints a single widget node in a friendly key/value form
// suitable for the default text renderer. The configuration block is
// pretty-printed as JSON because it varies per widget component.
func renderNodeText(stdout io.Writer, node openapi_client.SuperplaneComponentsNode) error {
	if _, err := fmt.Fprintf(stdout, "ID:        %s\n", node.GetId()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Name:      %s\n", node.GetName()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Component: %s\n", node.GetComponent()); err != nil {
		return err
	}
	if node.Position != nil {
		if _, err := fmt.Fprintf(stdout, "Position:  %d,%d\n", node.Position.GetX(), node.Position.GetY()); err != nil {
			return err
		}
	}
	if msg := node.GetErrorMessage(); msg != "" {
		if _, err := fmt.Fprintf(stdout, "Error:     %s\n", msg); err != nil {
			return err
		}
	}

	cfg := node.GetConfiguration()
	if len(cfg) == 0 {
		_, err := fmt.Fprintln(stdout, "Configuration: (none)")
		return err
	}

	encoded, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintln(stdout, "Configuration:"); err != nil {
		return err
	}
	for _, line := range strings.Split(string(encoded), "\n") {
		if _, err := fmt.Fprintf(stdout, "  %s\n", line); err != nil {
			return err
		}
	}
	return nil
}
