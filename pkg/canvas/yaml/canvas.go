package yaml

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/encoding/protojson"
)

const (
	CanvasKind       = "Canvas"
	CanvasAPIVersion = "v1"
)

// ParseCanvasResource parses canonical canvas.yaml bytes into a Canvas proto.
func ParseCanvasResource(data []byte) (*pb.Canvas, error) {
	jsonData, err := yaml.YAMLToJSON(data)
	if err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	jsonData, err = normalizeCanvasResourceJSON(jsonData)
	if err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	canvasJSON, err := canvasJSONFromResource(jsonData)
	if err != nil {
		return nil, err
	}

	var canvas pb.Canvas
	if err := protojson.Unmarshal(canvasJSON, &canvas); err != nil {
		return nil, fmt.Errorf("parse canvas definition: %w", err)
	}

	if canvas.Metadata == nil {
		return nil, fmt.Errorf("canvas metadata is required")
	}

	return &canvas, nil
}

func canvasJSONFromResource(jsonData []byte) ([]byte, error) {
	var resource map[string]json.RawMessage
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, fmt.Errorf("parse canvas yaml: %w", err)
	}

	if kindRaw, ok := resource["kind"]; ok {
		var kind string
		if err := json.Unmarshal(kindRaw, &kind); err != nil {
			return nil, fmt.Errorf("parse canvas definition: %w", err)
		}

		if kind != "" && kind != CanvasKind {
			return nil, fmt.Errorf("unsupported resource kind %q", kind)
		}
	}

	canvasPayload := make(map[string]json.RawMessage)
	if metadata, ok := resource["metadata"]; ok {
		canvasPayload["metadata"] = metadata
	}
	if spec, ok := resource["spec"]; ok {
		canvasPayload["spec"] = spec
	}

	if len(canvasPayload) == 0 {
		return jsonData, nil
	}

	canvasJSON, err := json.Marshal(canvasPayload)
	if err != nil {
		return nil, fmt.Errorf("parse canvas definition: %w", err)
	}

	return canvasJSON, nil
}

func normalizeCanvasResourceJSON(jsonData []byte) ([]byte, error) {
	var resource map[string]any
	if err := json.Unmarshal(jsonData, &resource); err != nil {
		return nil, err
	}

	spec, ok := resource["spec"].(map[string]any)
	if !ok {
		return jsonData, nil
	}

	nodes, ok := spec["nodes"].([]any)
	if !ok {
		return jsonData, nil
	}

	for i, raw := range nodes {
		node, ok := raw.(map[string]any)
		if !ok {
			continue
		}

		if position, ok := node["position"].(map[string]any); ok {
			node["position"] = normalizeYAMLPositionMap(position)
		}

		nodes[i] = node
	}

	spec["nodes"] = nodes
	resource["spec"] = spec

	return json.Marshal(resource)
}

func normalizeYAMLPositionMap(position map[string]any) map[string]any {
	normalized := make(map[string]any, len(position)+1)
	for key, value := range position {
		normalized[key] = value
	}

	if _, hasY := normalized["y"]; hasY {
		return normalized
	}

	if yValue, ok := normalized["true"]; ok {
		normalized["y"] = yValue
		delete(normalized, "true")
	}

	return normalized
}
