package console

import "github.com/superplanehq/superplane/pkg/openapi_client"

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
			Type:    panel.GetType(),
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
func apiPanelsFromYAML(panels []ConsoleYAMLPanel) []openapi_client.ConsolePanel {
	out := make([]openapi_client.ConsolePanel, 0, len(panels))
	for _, panel := range panels {
		id := panel.ID
		panelType := panel.Type
		converted := openapi_client.ConsolePanel{
			Id:   &id,
			Type: &panelType,
		}
		if panel.Content != nil {
			converted.Content = panel.Content
		}
		out = append(out, converted)
	}
	return out
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
