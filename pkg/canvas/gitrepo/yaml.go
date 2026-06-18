// Package gitrepo builds and seeds the git representation of a canvas: the
// canonical YAML encoders for canvas.yaml / console.yaml, the initial repository
// seeding, and the backfill that migrates pre-git-first canvases into git.
package gitrepo

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/superplanehq/superplane/pkg/grpc/actions"
	"github.com/superplanehq/superplane/pkg/models"
	componentpb "github.com/superplanehq/superplane/pkg/protos/components"
	"gopkg.in/yaml.v3"
)

// CanvasYAML is the canonical YAML representation of a canvas spec used when
// seeding or backfilling git repositories.
type CanvasYAML struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   CanvasYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       CanvasYAMLSpec     `json:"spec" yaml:"spec"`
}

type CanvasYAMLMetadata struct {
	Name        string `json:"name" yaml:"name"`
	Description string `json:"description,omitempty" yaml:"description,omitempty"`
}

type CanvasYAMLSpec struct {
	Nodes []models.Node
	Edges []models.Edge
}

type canvasYAMLResource struct {
	APIVersion string             `json:"apiVersion" yaml:"apiVersion"`
	Kind       string             `json:"kind" yaml:"kind"`
	Metadata   CanvasYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       canvasYAMLSpec     `json:"spec" yaml:"spec"`
}

type canvasYAMLSpec struct {
	Nodes []*componentpb.Node `json:"nodes" yaml:"nodes"`
	Edges []*componentpb.Edge `json:"edges" yaml:"edges"`
}

func CanvasYAMLFromVersion(version *models.CanvasVersion) *CanvasYAML {
	if version == nil {
		return nil
	}

	return &CanvasYAML{
		APIVersion: "v1",
		Kind:       "Canvas",
		Metadata: CanvasYAMLMetadata{
			Name:        version.Name,
			Description: version.Description,
		},
		Spec: CanvasYAMLSpec{
			Nodes: version.Nodes,
			Edges: version.Edges,
		},
	}
}

// CanvasYAMLToBytes encodes a canvas spec into canvas.yaml bytes.
func CanvasYAMLToBytes(canvas *CanvasYAML) ([]byte, error) {
	if canvas == nil {
		return nil, fmt.Errorf("canvas yaml is required")
	}

	apiVersion := strings.TrimSpace(canvas.APIVersion)
	if apiVersion == "" {
		apiVersion = "v1"
	}

	kind := strings.TrimSpace(canvas.Kind)
	if kind == "" {
		kind = "Canvas"
	}

	resource := canvasYAMLResource{
		APIVersion: apiVersion,
		Kind:       kind,
		Metadata:   canvas.Metadata,
		Spec: canvasYAMLSpec{
			Nodes: actions.NodesToProto(canvas.Spec.Nodes),
			Edges: actions.EdgesToProto(canvas.Spec.Edges),
		},
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("marshal canvas yaml: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("encode canvas yaml: %w", err)
	}

	return buf.Bytes(), nil
}

// ConsoleYAMLToBytes encodes a console dashboard into console.yaml bytes.
func ConsoleYAMLToBytes(console *models.ConsoleYAML) ([]byte, error) {
	if console == nil {
		return EmptyConsoleYAMLToBytes("", "")
	}

	jsonBytes, err := json.Marshal(console)
	if err != nil {
		return nil, fmt.Errorf("marshal console yaml: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("encode console yaml: %w", err)
	}

	return buf.Bytes(), nil
}

// EmptyConsoleYAMLToBytes encodes a default console.yaml for a canvas.
func EmptyConsoleYAMLToBytes(canvasID, canvasName string) ([]byte, error) {
	if strings.TrimSpace(canvasID) == "" {
		return models.CanvasVersionToConsoleYML(&models.CanvasVersion{Name: canvasName})
	}

	canvasUUID, err := uuid.Parse(canvasID)
	if err != nil {
		return nil, fmt.Errorf("invalid canvas id: %w", err)
	}

	return models.CanvasVersionToConsoleYML(&models.CanvasVersion{
		WorkflowID: canvasUUID,
		Name:       canvasName,
	})
}

// ConsoleYAMLFromVersionToBytes encodes a console.yaml derived from a version row.
func ConsoleYAMLFromVersionToBytes(version *models.CanvasVersion) ([]byte, error) {
	if version == nil {
		return EmptyConsoleYAMLToBytes("", "")
	}

	return models.CanvasVersionToConsoleYML(version)
}
