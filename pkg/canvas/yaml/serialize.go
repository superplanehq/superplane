package yaml

import (
	"bytes"
	"encoding/json"
	"strings"

	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/encoding/protojson"
	goyaml "gopkg.in/yaml.v3"
)

// CanvasResourceYAML serializes a canvas version into the canonical canvas.yaml
// representation. The caller passes the already-serialized proto version so this
// package stays free of model/database dependencies.
func CanvasResourceYAML(version *pb.CanvasVersion, canvasID string) (string, error) {
	// Use protojson (not encoding/json) so proto enums such as node.type are
	// emitted as their canonical names (e.g. "TYPE_TRIGGER") instead of their
	// numeric values. ParseCanvasResource reads canvas.yaml back with protojson,
	// and the UI relies on the string enum names, so serialization must match.
	specJSON, err := protojson.Marshal(version.GetSpec())
	if err != nil {
		return "", err
	}

	var spec map[string]any
	if err := json.Unmarshal(specJSON, &spec); err != nil {
		return "", err
	}

	if _, ok := spec["nodes"]; !ok {
		spec["nodes"] = []any{}
	}
	if _, ok := spec["edges"]; !ok {
		spec["edges"] = []any{}
	}
	ensureCanvasYAMLNodeDefaults(spec)
	ensureCanvasYAMLEdgeDefaults(spec)

	resource := map[string]any{
		"apiVersion": CanvasAPIVersion,
		"kind":       CanvasKind,
		"metadata": map[string]any{
			"id":          canvasID,
			"name":        version.GetMetadata().GetName(),
			"description": version.GetMetadata().GetDescription(),
		},
		"spec": spec,
	}

	var buf bytes.Buffer
	encoder := goyaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(resource); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}

	return quoteYAMLPositionYKeys(buf.String()), nil
}

// quoteYAMLPositionYKeys quotes bare `y:` keys so YAML 1.1 parsers do not read
// them back as the boolean `true`.
func quoteYAMLPositionYKeys(text string) string {
	lines := strings.Split(text, "\n")
	for i, line := range lines {
		trimmed := strings.TrimLeft(line, " \t")
		if strings.HasPrefix(trimmed, "y:") {
			indent := line[:len(line)-len(trimmed)]
			lines[i] = indent + `"y"` + trimmed[1:]
		}
	}
	return strings.Join(lines, "\n")
}

func ensureCanvasYAMLNodeDefaults(spec map[string]any) {
	nodes, ok := spec["nodes"].([]any)
	if !ok {
		return
	}

	for i, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		if _, hasType := node["type"]; !hasType {
			// Proto JSON omits TYPE_ACTION (enum value 0); the UI needs an explicit type.
			node["type"] = "TYPE_ACTION"
		}

		nodes[i] = node
	}

	spec["nodes"] = nodes
}

func ensureCanvasYAMLEdgeDefaults(spec map[string]any) {
	edges, ok := spec["edges"].([]any)
	if !ok {
		return
	}

	for i, raw := range edges {
		edge, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		channel, _ := edge["channel"].(string)
		if channel == "" {
			edge["channel"] = "default"
		}

		edges[i] = edge
	}

	spec["edges"] = edges
}
