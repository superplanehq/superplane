package models

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"

	"gopkg.in/yaml.v3"
)

const (
	DashboardKind       = "Dashboard"
	DashboardAPIVersion = "v1"

	DashboardPanelTypeMarkdown = "markdown"
	DashboardPanelTypeNode     = "node"
	DashboardPanelTypeTable    = "table"
	DashboardPanelTypeChart    = "chart"
	DashboardPanelTypeNumber   = "number"

	MaxDashboardPanels       = 50
	MaxDashboardPayloadBytes = 1024 * 1024
)

// AllowedDashboardPanelTypes lists the panel `type` values accepted on import.
// Keep this list in lockstep with `web_src/src/pages/workflowv2/dashboard/panelTypes.ts`
// — the frontend validators and per-type form editors rely on the same set.
var AllowedDashboardPanelTypes = []string{
	DashboardPanelTypeMarkdown,
	DashboardPanelTypeNode,
	DashboardPanelTypeTable,
	DashboardPanelTypeChart,
	DashboardPanelTypeNumber,
}

// DashboardYAMLMetadata is informational only. `canvasId` is ignored on
// import; `name` is used solely for display/filename purposes.
type DashboardYAMLMetadata struct {
	CanvasID string `json:"canvasId,omitempty" yaml:"canvasId,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
}

// DashboardYAMLSpec carries the persisted dashboard shape (panels + layout)
// while keeping a stable, deterministic field ordering on export.
type DashboardYAMLSpec struct {
	Panels []DashboardPanel      `json:"panels" yaml:"panels"`
	Layout []DashboardLayoutItem `json:"layout" yaml:"layout"`
}

// DashboardYAML is the canonical YAML representation of a canvas dashboard.
//
// Import is replace-all: it overwrites every panel and layout entry for the
// canvas. Export is deterministic: identical dashboards produce identical
// YAML bytes regardless of how the underlying maps were ordered in memory.
type DashboardYAML struct {
	APIVersion string                `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                `json:"kind" yaml:"kind"`
	Metadata   DashboardYAMLMetadata `json:"metadata" yaml:"metadata"`
	Spec       DashboardYAMLSpec     `json:"spec" yaml:"spec"`
}

// DashboardFromYML parses raw YAML bytes into a validated DashboardYAML. The
// parser is strict: unknown top-level fields are rejected, panel content must
// be an object, and the configured limits apply.
func DashboardFromYML(raw []byte) (*DashboardYAML, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("dashboard yaml is empty")
	}

	var asAny any
	if err := yaml.Unmarshal(raw, &asAny); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	if _, ok := asAny.(map[string]any); !ok {
		return nil, errors.New("dashboard yaml must be an object")
	}

	jsonBytes, err := json.Marshal(asAny)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	var resource DashboardYAML
	if err := decoder.Decode(&resource); err != nil {
		return nil, fmt.Errorf("invalid dashboard yaml: %w", err)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	return &resource, nil
}

// DashboardToYML serializes a stored dashboard into the canonical YAML
// representation with stable field ordering. Empty dashboards produce a
// valid empty spec.
func DashboardToYML(dashboard *CanvasDashboard, canvasName string) ([]byte, error) {
	if dashboard == nil {
		return nil, errors.New("dashboard is required")
	}

	resource := DashboardYAML{
		APIVersion: DashboardAPIVersion,
		Kind:       DashboardKind,
		Metadata: DashboardYAMLMetadata{
			CanvasID: dashboard.CanvasID.String(),
			Name:     canvasName,
		},
		Spec: DashboardYAMLSpec{
			Panels: normalizeDashboardPanelsForExport(dashboard.Panels.Data()),
			Layout: normalizeDashboardLayoutForExport(dashboard.Layout.Data()),
		},
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize dashboard: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("failed to serialize dashboard: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("failed to encode dashboard yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode dashboard yaml: %w", err)
	}
	return buf.Bytes(), nil
}

// Validate enforces the structural and size invariants of a dashboard import.
func (d *DashboardYAML) Validate() error {
	if d.APIVersion == "" {
		return errors.New("apiVersion is required")
	}
	if d.APIVersion != DashboardAPIVersion {
		return fmt.Errorf("unsupported apiVersion %q (expected %q)", d.APIVersion, DashboardAPIVersion)
	}
	if d.Kind == "" {
		return errors.New("kind is required")
	}
	if d.Kind != DashboardKind {
		return fmt.Errorf("unsupported kind %q (expected %q)", d.Kind, DashboardKind)
	}

	return ValidateDashboardContent(d.Spec.Panels, d.Spec.Layout)
}

// ValidateDashboardContent enforces the shared validation rules used by both
// YAML import and the gRPC update endpoint. Keeping this in models means the
// rules live next to the persisted shape and stay consistent across surfaces.
func ValidateDashboardContent(panels []DashboardPanel, layout []DashboardLayoutItem) error {
	if len(panels) > MaxDashboardPanels {
		return fmt.Errorf("too many panels (max %d)", MaxDashboardPanels)
	}

	panelIDs := make(map[string]struct{}, len(panels))
	for _, panel := range panels {
		if panel.ID == "" {
			return errors.New("panel id is required")
		}
		if panel.Type == "" {
			return fmt.Errorf("panel %q type is required", panel.ID)
		}
		if !isAllowedDashboardPanelType(panel.Type) {
			return fmt.Errorf("panel %q has unsupported type %q", panel.ID, panel.Type)
		}
		if _, exists := panelIDs[panel.ID]; exists {
			return fmt.Errorf("duplicate panel id %q", panel.ID)
		}
		if err := validatePanelContent(panel); err != nil {
			return err
		}
		panelIDs[panel.ID] = struct{}{}
	}

	size, err := encodedDashboardPanelsSize(panels)
	if err != nil {
		return fmt.Errorf("failed to validate panel size: %w", err)
	}
	if size > MaxDashboardPayloadBytes {
		return fmt.Errorf("panels payload exceeds %d bytes", MaxDashboardPayloadBytes)
	}

	layoutIDs := make(map[string]struct{}, len(layout))
	for _, item := range layout {
		if item.I == "" {
			return errors.New("layout item i is required")
		}
		if _, exists := layoutIDs[item.I]; exists {
			return fmt.Errorf("duplicate layout id %q", item.I)
		}
		layoutIDs[item.I] = struct{}{}

		if _, ok := panelIDs[item.I]; !ok {
			return fmt.Errorf("layout item %q does not reference any panel", item.I)
		}
		if item.W <= 0 || item.H <= 0 {
			return fmt.Errorf("layout item %q must have positive width and height", item.I)
		}
		if item.X < 0 || item.Y < 0 {
			return fmt.Errorf("layout item %q must have non-negative x and y", item.I)
		}
	}

	return nil
}

func isAllowedDashboardPanelType(panelType string) bool {
	for _, allowed := range AllowedDashboardPanelTypes {
		if panelType == allowed {
			return true
		}
	}
	return false
}

func validatePanelContent(panel DashboardPanel) error {
	switch panel.Type {
	case DashboardPanelTypeMarkdown:
		return validateMarkdownContent(panel)
	case DashboardPanelTypeNode:
		return validateNodePanelContent(panel)
	case DashboardPanelTypeTable:
		return validateTablePanelContent(panel)
	case DashboardPanelTypeChart:
		return validateChartPanelContent(panel)
	case DashboardPanelTypeNumber:
		return validateNumberPanelContent(panel)
	}
	return nil
}

func validateMarkdownContent(panel DashboardPanel) error {
	if panel.Content == nil {
		return nil
	}
	if rawTitle, ok := panel.Content["title"]; ok && rawTitle != nil {
		if _, ok := rawTitle.(string); !ok {
			return fmt.Errorf("panel %q content.title must be a string", panel.ID)
		}
	}
	if rawBody, ok := panel.Content["body"]; ok && rawBody != nil {
		if _, ok := rawBody.(string); !ok {
			return fmt.Errorf("panel %q content.body must be a string", panel.ID)
		}
	}
	return nil
}

func validateNodePanelContent(panel DashboardPanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	// `node` must be present as a string but may be empty: newly added
	// panels start unconfigured and the UI renders a "configure me" hint
	// until the user picks one. The card body never executes a trigger /
	// status lookup against an empty reference.
	rawNode, ok := panel.Content["node"]
	if !ok {
		return fmt.Errorf("panel %q content.node is required", panel.ID)
	}
	if _, ok := rawNode.(string); !ok {
		return fmt.Errorf("panel %q content.node must be a string", panel.ID)
	}
	if rawShowRun, ok := panel.Content["showRun"]; ok && rawShowRun != nil {
		if _, ok := rawShowRun.(bool); !ok {
			return fmt.Errorf("panel %q content.showRun must be a boolean", panel.ID)
		}
	}
	return nil
}

func validateDataSource(panelID string, raw any) error {
	ds, ok := raw.(map[string]any)
	if !ok || ds == nil {
		return fmt.Errorf("panel %q dataSource must be an object", panelID)
	}
	kind, _ := ds["kind"].(string)
	switch kind {
	case "memory":
		ns, ok := ds["namespace"].(string)
		if !ok || ns == "" {
			return fmt.Errorf("panel %q dataSource.namespace must be a non-empty string for memory sources", panelID)
		}
	case "executions", "runs":
		// node (executions only) and limit are optional
	default:
		return fmt.Errorf("panel %q dataSource.kind must be \"memory\", \"executions\", or \"runs\"", panelID)
	}
	return nil
}

func validateRender(panelID string, raw any, expectedKind string) (map[string]any, error) {
	render, ok := raw.(map[string]any)
	if !ok || render == nil {
		return nil, fmt.Errorf("panel %q render must be an object", panelID)
	}
	kind, _ := render["kind"].(string)
	if kind != expectedKind {
		return nil, fmt.Errorf("panel %q render.kind must be %q", panelID, expectedKind)
	}
	return render, nil
}

func validateTablePanelContent(panel DashboardPanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "table")
	if err != nil {
		return err
	}
	cols, ok := render["columns"].([]any)
	if !ok || len(cols) == 0 {
		return fmt.Errorf("panel %q render.columns must be a non-empty array", panel.ID)
	}
	return nil
}

func validateChartPanelContent(panel DashboardPanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "chart")
	if err != nil {
		return err
	}
	if xField, ok := render["xField"].(string); !ok || xField == "" {
		return fmt.Errorf("panel %q render.xField must be a non-empty string", panel.ID)
	}
	if series, ok := render["series"].([]any); !ok || len(series) == 0 {
		return fmt.Errorf("panel %q render.series must be a non-empty array", panel.ID)
	}
	return nil
}

func validateNumberPanelContent(panel DashboardPanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "number")
	if err != nil {
		return err
	}
	aggregation, _ := render["aggregation"].(string)
	switch aggregation {
	case "count", "sum", "avg", "min", "max", "first", "last":
	default:
		return fmt.Errorf("panel %q render.aggregation must be one of count/sum/avg/min/max/first/last", panel.ID)
	}
	if aggregation != "count" {
		if field, ok := render["field"].(string); !ok || field == "" {
			return fmt.Errorf("panel %q render.field is required when aggregation is %q", panel.ID, aggregation)
		}
	}
	return nil
}

// normalizeDashboardPanelsForExport ensures stable field order in panel
// content maps so YAML output is deterministic across runs.
func normalizeDashboardPanelsForExport(panels []DashboardPanel) []DashboardPanel {
	if panels == nil {
		return []DashboardPanel{}
	}

	out := make([]DashboardPanel, len(panels))
	for i, panel := range panels {
		out[i] = DashboardPanel{
			ID:      panel.ID,
			Type:    panel.Type,
			Content: panel.Content,
		}
	}
	return out
}

func normalizeDashboardLayoutForExport(layout []DashboardLayoutItem) []DashboardLayoutItem {
	if layout == nil {
		return []DashboardLayoutItem{}
	}

	out := make([]DashboardLayoutItem, len(layout))
	copy(out, layout)
	return out
}

func encodedDashboardPanelsSize(panels []DashboardPanel) (int, error) {
	encoded, err := json.Marshal(panels)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}
