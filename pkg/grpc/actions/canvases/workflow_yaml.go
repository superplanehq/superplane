package canvases

import (
	"bytes"
	"encoding/json"

	canvasyaml "github.com/superplanehq/superplane/pkg/canvas/yaml"
	"github.com/superplanehq/superplane/pkg/models"
	pb "github.com/superplanehq/superplane/pkg/protos/canvases"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"strings"

	"gopkg.in/yaml.v3"
)

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

func canvasYAMLFromVersion(canvas *models.Canvas, version *models.CanvasVersion, organizationID string) (string, error) {
	protoVersion := SerializeCanvasVersion(version, organizationID)

	specJSON, err := json.Marshal(protoVersion.GetSpec())
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
		"apiVersion": canvasyaml.CanvasAPIVersion,
		"kind":       canvasyaml.CanvasKind,
		"metadata": map[string]any{
			"id":          canvas.ID.String(),
			"name":        protoVersion.GetMetadata().GetName(),
			"description": protoVersion.GetMetadata().GetDescription(),
			"isTemplate":  canvas.IsTemplate,
		},
		"spec": spec,
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(resource); err != nil {
		return "", err
	}
	if err := encoder.Close(); err != nil {
		return "", err
	}

	return quoteYAMLPositionYKeys(buf.String()), nil
}

func consoleYAMLFromVersion(version *models.CanvasVersion) (string, error) {
	raw, err := models.CanvasVersionToConsoleYML(version)
	if err != nil {
		return "", err
	}
	return string(raw), nil
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

func canvasFromYAMLText(text string) (*pb.Canvas, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, status.Error(codes.InvalidArgument, "canvas_yaml is empty")
	}

	canvas, err := canvasyaml.ParseCanvasResource([]byte(trimmed))
	if err != nil {
		return nil, status.Errorf(codes.InvalidArgument, "invalid canvas_yaml: %v", err)
	}

	return canvas, nil
}

func consolePanelsLayoutFromYAMLText(text string) ([]models.ConsolePanel, []models.ConsoleLayoutItem, error) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return nil, nil, status.Error(codes.InvalidArgument, "console_yaml is empty")
	}

	doc, err := models.ConsoleFromYML([]byte(trimmed))
	if err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid console_yaml: %v", err)
	}

	if err := doc.Validate(); err != nil {
		return nil, nil, status.Errorf(codes.InvalidArgument, "invalid console_yaml: %v", err)
	}

	return doc.Spec.Panels, doc.Spec.Layout, nil
}
