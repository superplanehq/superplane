package console

import (
	"fmt"

	"github.com/superplanehq/superplane/pkg/openapi_client"
)

// panelTypeAPIToYAML maps the generated SDK enum string (SCREAMING_CASE) to
// the lowercase YAML form. The CLI intentionally avoids depending on
// `pkg/models`, so the mapping is duplicated here; keep it in sync with the
// `Console.Panel.Type` enum in `protos/canvases.proto`.
var panelTypeAPIToYAML = map[openapi_client.ConsolePanelType]string{
	openapi_client.CONSOLEPANELTYPE_MARKDOWN: "markdown",
	openapi_client.CONSOLEPANELTYPE_NODE:     "node",
	openapi_client.CONSOLEPANELTYPE_NODES:    "nodes",
	openapi_client.CONSOLEPANELTYPE_TABLE:    "table",
	openapi_client.CONSOLEPANELTYPE_CHART:    "chart",
	openapi_client.CONSOLEPANELTYPE_NUMBER:   "number",
}

var panelTypeYAMLToAPI = map[string]openapi_client.ConsolePanelType{
	"markdown": openapi_client.CONSOLEPANELTYPE_MARKDOWN,
	"node":     openapi_client.CONSOLEPANELTYPE_NODE,
	"nodes":    openapi_client.CONSOLEPANELTYPE_NODES,
	"table":    openapi_client.CONSOLEPANELTYPE_TABLE,
	"chart":    openapi_client.CONSOLEPANELTYPE_CHART,
	"number":   openapi_client.CONSOLEPANELTYPE_NUMBER,
}

// panelTypeFromAPI returns the lowercase YAML type for a panel returned by
// the API. Unknown / unset enum values fall back to the raw enum string so
// `superplane apps console export` still produces a readable YAML for panel
// kinds the CLI binary predates.
func panelTypeFromAPI(t openapi_client.ConsolePanelType) string {
	if mapped, ok := panelTypeAPIToYAML[t]; ok {
		return mapped
	}
	return string(t)
}

// panelTypeToAPI returns the SDK enum for a YAML panel type. Returns an
// error for unknown values so `superplane apps console import` fails fast
// with a clear message instead of hitting the server with `TYPE_UNSPECIFIED`.
func panelTypeToAPI(t string) (openapi_client.ConsolePanelType, error) {
	if mapped, ok := panelTypeYAMLToAPI[t]; ok {
		return mapped, nil
	}
	return openapi_client.CONSOLEPANELTYPE_TYPE_UNSPECIFIED, fmt.Errorf("unknown panel type %q", t)
}

// consoleYAMLFromAPI converts the API-returned console payload into the
// canonical YAML shape. Empty panels/layout default to empty slices so the
// exported YAML always has a stable form.
func consoleYAMLFromAPI(canvasName string, console openapi_client.CanvasesConsole) ConsoleYAML {
	panels := make([]ConsoleYAMLPanel, 0, len(console.GetPanels()))
	for _, panel := range console.GetPanels() {
		content := map[string]any{}
		for k, v := range panel.GetContent() {
			content[k] = v
		}
		panels = append(panels, ConsoleYAMLPanel{
			ID:      panel.GetId(),
			Type:    panelTypeFromAPI(panel.GetType()),
			Content: content,
		})
	}

	layout := make([]ConsoleYAMLLayoutItem, 0, len(console.GetLayout()))
	for _, item := range console.GetLayout() {
		converted := ConsoleYAMLLayoutItem{
			I: item.GetI(),
			X: int(item.GetX()),
			Y: int(item.GetY()),
			W: int(item.GetW()),
			H: int(item.GetH()),
		}
		if item.HasMinW() {
			minW := int(item.GetMinW())
			converted.MinW = &minW
		}
		if item.HasMinH() {
			minH := int(item.GetMinH())
			converted.MinH = &minH
		}
		layout = append(layout, converted)
	}

	return ConsoleYAML{
		APIVersion: ConsoleAPIVersion,
		Kind:       ConsoleKind,
		Metadata: ConsoleYAMLMetadata{
			CanvasID: console.GetCanvasId(),
			Name:     canvasName,
		},
		Spec: ConsoleYAMLSpec{
			Panels: panels,
			Layout: layout,
		},
	}
}

// apiPanelsFromYAML converts parsed YAML panels into the API request shape.
// Returns an error when a panel carries an unknown `type` so we surface the
// problem at conversion time instead of letting the server reject the
// request with `TYPE_UNSPECIFIED`.
func apiPanelsFromYAML(panels []ConsoleYAMLPanel) ([]openapi_client.ConsolePanel, error) {
	out := make([]openapi_client.ConsolePanel, 0, len(panels))
	for _, panel := range panels {
		id := panel.ID
		panelType, err := panelTypeToAPI(panel.Type)
		if err != nil {
			return nil, fmt.Errorf("panel %q: %w", panel.ID, err)
		}
		converted := openapi_client.ConsolePanel{
			Id:   &id,
			Type: &panelType,
		}
		if panel.Content != nil {
			converted.Content = panel.Content
		}
		out = append(out, converted)
	}
	return out, nil
}

// apiLayoutFromYAML converts parsed YAML layout entries into the API request
// shape.
func apiLayoutFromYAML(layout []ConsoleYAMLLayoutItem) []openapi_client.ConsoleLayoutItem {
	out := make([]openapi_client.ConsoleLayoutItem, 0, len(layout))
	for _, item := range layout {
		i := item.I
		x := int32(item.X)
		y := int32(item.Y)
		w := int32(item.W)
		h := int32(item.H)
		converted := openapi_client.ConsoleLayoutItem{
			I: &i,
			X: &x,
			Y: &y,
			W: &w,
			H: &h,
		}
		if item.MinW != nil {
			minW := int32(*item.MinW)
			converted.MinW = &minW
		}
		if item.MinH != nil {
			minH := int32(*item.MinH)
			converted.MinH = &minH
		}
		out = append(out, converted)
	}
	return out
}
