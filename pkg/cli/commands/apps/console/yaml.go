// Package console contains the `superplane apps console` command group. It
// reads and writes canvas console configuration as canonical Console YAML
// (the same shape used by the UI Console YAML modal and `console.yaml`
// shipped with installable apps).
package console

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

// ConsoleAPIVersion is the only supported apiVersion for Console files.
const ConsoleAPIVersion = "v1"

// ConsoleKind is the canonical YAML kind for canvas consoles. The product
// name is "Console"; the backend type and folder use the legacy "dashboard"
// name internally, but every user-facing YAML carries `kind: Console`.
const ConsoleKind = "Console"

// ConsoleYAML mirrors the canonical Console YAML used everywhere else
// (frontend `dashboardYaml.ts`, backend `models.DashboardYAML`, install
// `console.yaml`). The CLI keeps its own minimal definition so the
// shipped binary does not need to depend on `pkg/models`.
type ConsoleYAML struct {
	APIVersion string              `json:"apiVersion" yaml:"apiVersion"`
	Kind       string              `json:"kind" yaml:"kind"`
	Metadata   ConsoleYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       ConsoleYAMLSpec     `json:"spec" yaml:"spec"`
}

// ConsoleYAMLMetadata is informational only on both export and import.
type ConsoleYAMLMetadata struct {
	CanvasID string `json:"canvasId,omitempty" yaml:"canvasId,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
}

// ConsoleYAMLSpec carries panels and layout.
type ConsoleYAMLSpec struct {
	Panels []ConsoleYAMLPanel      `json:"panels" yaml:"panels"`
	Layout []ConsoleYAMLLayoutItem `json:"layout" yaml:"layout"`
}

// ConsoleYAMLPanel matches `models.DashboardPanel`.
type ConsoleYAMLPanel struct {
	ID      string         `json:"id" yaml:"id"`
	Type    string         `json:"type" yaml:"type"`
	Content map[string]any `json:"content" yaml:"content"`
}

// ConsoleYAMLLayoutItem matches `models.DashboardLayoutItem`.
type ConsoleYAMLLayoutItem struct {
	I    string `json:"i" yaml:"i"`
	X    int    `json:"x" yaml:"x"`
	Y    int    `json:"y" yaml:"y"`
	W    int    `json:"w" yaml:"w"`
	H    int    `json:"h" yaml:"h"`
	MinW *int   `json:"minW,omitempty" yaml:"minW,omitempty"`
	MinH *int   `json:"minH,omitempty" yaml:"minH,omitempty"`
}

// ParseConsoleYAML decodes raw YAML bytes into a ConsoleYAML and verifies
// the apiVersion/kind headers. Deeper validation (panel types, size
// limits, content shapes) is left to the backend so that one set of rules
// applies regardless of the entry point.
func ParseConsoleYAML(raw []byte) (*ConsoleYAML, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("console yaml is empty")
	}

	var asAny any
	if err := yaml.Unmarshal(raw, &asAny); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if _, ok := asAny.(map[string]any); !ok {
		return nil, errors.New("console yaml must be an object")
	}

	jsonBytes, err := json.Marshal(asAny)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	var resource ConsoleYAML
	if err := decoder.Decode(&resource); err != nil {
		return nil, fmt.Errorf("invalid console yaml: %w", err)
	}

	if resource.APIVersion == "" {
		return nil, errors.New("apiVersion is required")
	}
	if resource.APIVersion != ConsoleAPIVersion {
		return nil, fmt.Errorf("unsupported apiVersion %q (expected %q)", resource.APIVersion, ConsoleAPIVersion)
	}
	if resource.Kind == "" {
		return nil, errors.New("kind is required")
	}
	if resource.Kind != ConsoleKind {
		return nil, fmt.Errorf("unsupported kind %q (expected %q)", resource.Kind, ConsoleKind)
	}

	return &resource, nil
}
