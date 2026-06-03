package yaml

import (
	"encoding/json"
	"fmt"

	"github.com/ghodss/yaml"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/protobuf/encoding/protojson"
)

func ParseCanvasYAML(data []byte) (*pb.Canvas, error) {
	jsonData, err := yaml.YAMLToJSON(data)
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

	if canvas.Metadata.GetIsTemplate() {
		return nil, fmt.Errorf("canvas templates cannot be installed")
	}

	canvas.Metadata.Id = ""
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

		if kind != "" && kind != "Canvas" {
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
