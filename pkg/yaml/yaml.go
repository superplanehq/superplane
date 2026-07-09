package yaml

const (
	KindCanvas  = "Canvas"
	KindConsole = "Console"
	APIVersion  = "v1"
)

// normalizeCanvasDocument fixes YAML 1.1 documents where unquoted "y" was parsed
// as boolean true, leaving node positions under a "true" key instead of "y".
func normalizeCanvasDocument(doc map[string]any) {
	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		return
	}

	nodes, ok := spec["nodes"].([]any)
	if !ok {
		return
	}

	for i, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		position, ok := node["position"].(map[string]any)
		if !ok {
			continue
		}

		normalizeYAML1YKey(position)
		node["position"] = position
		nodes[i] = node
	}

	spec["nodes"] = nodes
}

// normalizeConsoleDocument fixes YAML 1.1 layout items where unquoted "y" was
// parsed as boolean true.
func normalizeConsoleDocument(doc map[string]any) {
	spec, ok := doc["spec"].(map[string]any)
	if !ok {
		return
	}

	layout, ok := spec["layout"].([]any)
	if !ok {
		return
	}

	for i, raw := range layout {
		item, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		normalizeYAML1YKey(item)
		layout[i] = item
	}

	spec["layout"] = layout
}

func normalizeYAML1YKey(m map[string]any) {
	if _, hasY := m["y"]; hasY {
		return
	}

	if yValue, ok := m["true"]; ok {
		m["y"] = yValue
		delete(m, "true")
	}
}
