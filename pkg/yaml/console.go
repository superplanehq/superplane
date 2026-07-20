package yaml

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"slices"
	"strings"

	"github.com/superplanehq/superplane/pkg/models"
	"gopkg.in/yaml.v3"
)

const (
	ConsolePanelTypeMarkdown  = "markdown"
	ConsolePanelTypeHTML      = "html"
	ConsolePanelTypeNode      = "node"
	ConsolePanelTypeNodes     = "nodes"
	ConsolePanelTypeTable     = "table"
	ConsolePanelTypeChart     = "chart"
	ConsolePanelTypeNumber    = "number"
	ConsolePanelTypeScorecard = "scorecard"

	// ConsoleNodesPanelFormMode* control whether a `nodes` panel entry
	// renders its manual-run parameter form as a modal dialog (default)
	// or inline in the panel body. Keep in lockstep with
	// `NODES_PANEL_FORM_MODES` in the frontend `nodesPanelContent.ts`.
	ConsoleNodesPanelFormModeModal  = "modal"
	ConsoleNodesPanelFormModeInline = "inline"

	MaxConsolePanels       = 50
	MaxConsolePayloadBytes = 1024 * 1024
)

// AllowedConsolePanelTypes lists the panel `type` values accepted on import.
// Keep this list in lockstep with `web_src/src/pages/app/console/panelTypes.ts`
// — the frontend validators and per-type form editors rely on the same set.
var AllowedConsolePanelTypes = []string{
	ConsolePanelTypeMarkdown,
	ConsolePanelTypeHTML,
	ConsolePanelTypeNode,
	ConsolePanelTypeNodes,
	ConsolePanelTypeTable,
	ConsolePanelTypeChart,
	ConsolePanelTypeNumber,
	ConsolePanelTypeScorecard,
}

type Console struct {
	APIVersion string          `json:"apiVersion" yaml:"apiVersion"`
	Kind       string          `json:"kind" yaml:"kind"`
	Metadata   ConsoleMetadata `json:"metadata" yaml:"metadata"`
	Spec       ConsoleSpec     `json:"spec" yaml:"spec"`
}

func (c *Console) Panels() []models.ConsolePanel {
	out := make([]models.ConsolePanel, len(c.Spec.Panels))
	for i, panel := range c.Spec.Panels {
		out[i] = models.ConsolePanel{
			ID:      panel.ID,
			Type:    panel.Type,
			Content: panel.Content,
		}
	}
	return out
}

func (c *Console) Layout() []models.ConsoleLayoutItem {
	out := make([]models.ConsoleLayoutItem, len(c.Spec.Layout))
	for i, item := range c.Spec.Layout {
		out[i] = models.ConsoleLayoutItem{
			I:    item.I,
			X:    item.X,
			Y:    item.Y,
			W:    item.W,
			H:    item.H,
			MinW: item.MinW,
			MinH: item.MinH,
		}
	}
	return out
}

type ConsoleMetadata struct {
	CanvasID string `json:"canvasId" yaml:"canvasId"`
	Name     string `json:"name" yaml:"name"`
}

type ConsoleSpec struct {
	Panels []ConsolePanel      `json:"panels" yaml:"panels"`
	Layout []ConsoleLayoutItem `json:"layout" yaml:"layout"`
}

type ConsolePanel struct {
	ID      string         `json:"id" yaml:"id"`
	Type    string         `json:"type" yaml:"type"`
	Content map[string]any `json:"content" yaml:"content"`
}

type ConsoleLayoutItem struct {
	I    string `json:"i" yaml:"i"`
	X    int    `json:"x" yaml:"x"`
	Y    int    `json:"y" yaml:"y"`
	W    int    `json:"w" yaml:"w"`
	H    int    `json:"h" yaml:"h"`
	MinW *int   `json:"minW,omitempty" yaml:"minW,omitempty"`
	MinH *int   `json:"minH,omitempty" yaml:"minH,omitempty"`
}

func ConsoleFromYML(raw []byte) (*Console, error) {
	if len(bytes.TrimSpace(raw)) == 0 {
		return nil, errors.New("console yaml is empty")
	}

	var asAny any
	if err := yaml.Unmarshal(raw, &asAny); err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}
	doc, ok := asAny.(map[string]any)
	if !ok {
		return nil, errors.New("console yaml must be an object")
	}

	normalizeConsoleDocument(doc)

	jsonBytes, err := json.Marshal(doc)
	if err != nil {
		return nil, fmt.Errorf("invalid yaml: %w", err)
	}

	decoder := json.NewDecoder(bytes.NewReader(jsonBytes))
	decoder.DisallowUnknownFields()

	var resource Console
	if err := decoder.Decode(&resource); err != nil {
		return nil, fmt.Errorf("invalid console yaml: %w", err)
	}

	if err := resource.Validate(); err != nil {
		return nil, err
	}

	return &resource, nil
}

func VersionToConsoleYML(canvasName string, canvasVersion *models.CanvasVersion) (string, error) {
	if canvasVersion == nil {
		return "", errors.New("canvas version is required")
	}

	resource := Console{
		APIVersion: APIVersion,
		Kind:       KindConsole,
		Spec: ConsoleSpec{
			Panels: normalizeConsolePanelsForExport(canvasVersion.ConsolePanels.Data()),
			Layout: normalizeConsoleLayoutForExport(canvasVersion.ConsoleLayout.Data()),
		},
	}

	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return "", fmt.Errorf("failed to serialize console: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return "", fmt.Errorf("failed to serialize console: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return "", fmt.Errorf("failed to encode console yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return "", fmt.Errorf("failed to encode console yaml: %w", err)
	}
	return buf.String(), nil
}

func (c *Console) Validate() error {
	if c.APIVersion == "" {
		return errors.New("apiVersion is required")
	}
	if c.APIVersion != APIVersion {
		return fmt.Errorf("unsupported apiVersion %q (expected %q)", c.APIVersion, APIVersion)
	}
	if c.Kind == "" {
		return errors.New("kind is required")
	}
	if c.Kind != KindConsole {
		return fmt.Errorf("unsupported kind %q (expected %q)", c.Kind, KindConsole)
	}

	return ValidateConsoleContent(c.Spec.Panels, c.Spec.Layout)
}

func ValidateConsoleContent(panels []ConsolePanel, layout []ConsoleLayoutItem) error {
	if len(panels) > MaxConsolePanels {
		return fmt.Errorf("too many panels (max %d)", MaxConsolePanels)
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

	size, err := encodedConsolePanelsSize(panels)
	if err != nil {
		return fmt.Errorf("failed to validate panel size: %w", err)
	}
	if size > MaxConsolePayloadBytes {
		return fmt.Errorf("panels payload exceeds %d bytes", MaxConsolePayloadBytes)
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
	for _, allowed := range AllowedConsolePanelTypes {
		if panelType == allowed {
			return true
		}
	}
	return false
}

func validatePanelContent(panel ConsolePanel) error {
	switch panel.Type {
	case ConsolePanelTypeMarkdown:
		return validateMarkdownContent(panel)
	case ConsolePanelTypeHTML:
		return validateHTMLContent(panel)
	case ConsolePanelTypeNode:
		return validateNodePanelContent(panel)
	case ConsolePanelTypeNodes:
		return validateNodesPanelContent(panel)
	case ConsolePanelTypeTable:
		return validateTablePanelContent(panel)
	case ConsolePanelTypeChart:
		return validateChartPanelContent(panel)
	case ConsolePanelTypeNumber:
		return validateNumberPanelContent(panel)
	case ConsolePanelTypeScorecard:
		return validateScorecardPanelContent(panel)
	}
	return nil
}

// markdownVariableNameRe mirrors the FE regex in `panelTypes.ts`. Variable
// names must be valid CEL identifiers because the markdown body references
// them inside `{{ }}` expressions.
var markdownVariableNameRe = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

// AllowedMarkdownRunSelects mirrors `MARKDOWN_RUN_SELECTS` on the FE.
var AllowedMarkdownRunSelects = []string{"latest", "latest_passed", "latest_failed"}

// AllowedRunStatusFilters mirrors `RUN_STATUS_FILTER_OPTIONS` on the FE.
// Shared by the widget runs datasource and markdown/html run variables so
// the accepted vocabulary cannot drift between kinds.
var AllowedRunStatusFilters = []string{"running", "passed", "failed", "cancelled"}

// AllowedMarkdownVariableDirections mirrors `MARKDOWN_VARIABLE_DIRECTIONS` on the FE.
var AllowedMarkdownVariableDirections = []string{"asc", "desc"}

// AllowedMarkdownVariableModes mirrors `MARKDOWN_VARIABLE_MODES` on the FE.
// `single` keeps the existing first-row behavior; `list` resolves the
// variable to every matching row so authors can use CEL list macros.
var AllowedMarkdownVariableModes = []string{"single", "list"}

func validateMarkdownContent(panel ConsolePanel) error {
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
	if rawVars, ok := panel.Content["variables"]; ok && rawVars != nil {
		if err := validateMarkdownVariables(panel.ID, rawVars); err != nil {
			return err
		}
	}
	return nil
}

// validateHTMLContent enforces the shape of an html panel's content. The body
// is stored raw and sanitized client-side at render time (same trust model as
// markdown), so the backend only checks structure: title and body are optional
// strings and the shared variable system rules apply.
func validateHTMLContent(panel ConsolePanel) error {
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
	if rawVars, ok := panel.Content["variables"]; ok && rawVars != nil {
		if err := validateMarkdownVariables(panel.ID, rawVars); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkdownVariables(panelID string, raw any) error {
	list, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q content.variables must be an array", panelID)
	}
	names := make(map[string]struct{}, len(list))
	for i, item := range list {
		obj, ok := item.(map[string]any)
		if !ok {
			return fmt.Errorf("panel %q content.variables[%d] must be an object", panelID, i)
		}
		name, ok := obj["name"].(string)
		if !ok || !markdownVariableNameRe.MatchString(name) {
			return fmt.Errorf("panel %q content.variables[%d].name must be a valid identifier (letters, digits, underscore; not starting with a digit)", panelID, i)
		}
		if _, exists := names[name]; exists {
			return fmt.Errorf("panel %q content.variables[%d].name %q is duplicated", panelID, i, name)
		}
		names[name] = struct{}{}
		if err := validateMarkdownVariableSource(panelID, i, obj["source"]); err != nil {
			return err
		}
	}
	return nil
}

func validateMarkdownVariableSource(panelID string, index int, raw any) error {
	obj, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("panel %q content.variables[%d].source must be an object", panelID, index)
	}
	switch obj["kind"] {
	case "memory":
		return validateMarkdownMemorySource(panelID, index, obj)
	case "run":
		return validateMarkdownRunSource(panelID, index, obj)
	default:
		return fmt.Errorf("panel %q content.variables[%d].source.kind must be \"memory\" or \"run\"", panelID, index)
	}
}

func validateMarkdownMemorySource(panelID string, index int, source map[string]any) error {
	namespace, ok := source["namespace"].(string)
	if !ok || strings.TrimSpace(namespace) == "" {
		return fmt.Errorf("panel %q content.variables[%d].source.namespace must be a non-empty string", panelID, index)
	}
	if raw, ok := source["orderBy"]; ok && raw != nil {
		if _, ok := raw.(string); !ok {
			return fmt.Errorf("panel %q content.variables[%d].source.orderBy must be a string", panelID, index)
		}
	}
	if raw, ok := source["direction"]; ok && raw != nil {
		direction, ok := raw.(string)
		if !ok || !slices.Contains(AllowedMarkdownVariableDirections, direction) {
			return fmt.Errorf("panel %q content.variables[%d].source.direction must be \"asc\" or \"desc\"", panelID, index)
		}
	}
	if err := validateMarkdownMemoryMatches(panelID, index, source["matches"]); err != nil {
		return err
	}
	if err := validateMarkdownMemoryMode(panelID, index, source["mode"]); err != nil {
		return err
	}
	return validateMarkdownMemoryLimit(panelID, index, source["limit"])
}

func validateMarkdownMemoryMatches(panelID string, index int, raw any) error {
	if raw == nil {
		return nil
	}
	matches, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q content.variables[%d].source.matches must be an array", panelID, index)
	}
	for j, m := range matches {
		match, ok := m.(map[string]any)
		if !ok {
			return fmt.Errorf("panel %q content.variables[%d].source.matches[%d] must be an object", panelID, index, j)
		}
		field, ok := match["field"].(string)
		if !ok || strings.TrimSpace(field) == "" {
			return fmt.Errorf("panel %q content.variables[%d].source.matches[%d].field must be a non-empty string", panelID, index, j)
		}
		if rawValue, ok := match["value"]; ok && rawValue != nil {
			if _, ok := rawValue.(string); !ok {
				return fmt.Errorf("panel %q content.variables[%d].source.matches[%d].value must be a string", panelID, index, j)
			}
		}
	}
	return nil
}

func validateMarkdownMemoryMode(panelID string, index int, raw any) error {
	if raw == nil {
		return nil
	}
	mode, ok := raw.(string)
	if !ok || !slices.Contains(AllowedMarkdownVariableModes, mode) {
		return fmt.Errorf("panel %q content.variables[%d].source.mode must be \"single\" or \"list\"", panelID, index)
	}
	return nil
}

// validateMarkdownMemoryLimit accepts the integer-shaped JSON / YAML decoder
// outputs we see in practice: `int`, `int64`, and `float64` carrying a whole
// number. Anything else - non-numeric, fractional, zero, negative - is
// rejected with the same message so the UI and YAML editors see consistent
// feedback.
func validateMarkdownMemoryLimit(panelID string, index int, raw any) error {
	if raw == nil {
		return nil
	}
	msg := fmt.Errorf("panel %q content.variables[%d].source.limit must be a positive integer", panelID, index)
	switch v := raw.(type) {
	case int:
		if v <= 0 {
			return msg
		}
		return nil
	case int64:
		if v <= 0 {
			return msg
		}
		return nil
	case float64:
		if v <= 0 || v != float64(int64(v)) {
			return msg
		}
		return nil
	default:
		return msg
	}
}

func validateMarkdownRunSource(panelID string, index int, source map[string]any) error {
	selectValue, ok := source["select"].(string)
	if !ok || !slices.Contains(AllowedMarkdownRunSelects, selectValue) {
		return fmt.Errorf("panel %q content.variables[%d].source.select must be one of %s", panelID, index, strings.Join(AllowedMarkdownRunSelects, ", "))
	}
	statusesField := fmt.Sprintf("content.variables[%d].source.statuses", index)
	if err := validateRunStatusesField(panelID, statusesField, source["statuses"]); err != nil {
		return err
	}
	triggersField := fmt.Sprintf("content.variables[%d].source.triggers", index)
	return validateRunTriggersField(panelID, triggersField, source["triggers"])
}

// validateRunStatusesField accepts undefined / nil / empty (meaning "all
// statuses") and any subset of AllowedRunStatusFilters. Shared by the
// widget runs datasource and markdown/html run variables so the accepted
// vocabulary stays identical between kinds.
func validateRunStatusesField(panelID, fieldPath string, raw any) error {
	if raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q %s must be an array", panelID, fieldPath)
	}
	for i, item := range list {
		status, ok := item.(string)
		if !ok || !slices.Contains(AllowedRunStatusFilters, status) {
			return fmt.Errorf("panel %q %s[%d] must be one of %s", panelID, fieldPath, i, strings.Join(AllowedRunStatusFilters, ", "))
		}
	}
	return nil
}

// validateRunTriggersField accepts undefined / nil / empty (meaning "all
// triggers") and any list of non-empty strings. Individual entries are
// matched at runtime against the canvas nodes so unknown ids simply fail
// to match rather than fail validation.
func validateRunTriggersField(panelID, fieldPath string, raw any) error {
	if raw == nil {
		return nil
	}
	list, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q %s must be an array", panelID, fieldPath)
	}
	for i, item := range list {
		trigger, ok := item.(string)
		if !ok || strings.TrimSpace(trigger) == "" {
			return fmt.Errorf("panel %q %s[%d] must be a non-empty string", panelID, fieldPath, i)
		}
	}
	return nil
}

func validateNodePanelContent(panel ConsolePanel) error {
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
	if err := validateOptionalString(panel.ID, "content.label", panel.Content["label"]); err != nil {
		return err
	}
	if rawShowRun, ok := panel.Content["showRun"]; ok && rawShowRun != nil {
		if _, ok := rawShowRun.(bool); !ok {
			return fmt.Errorf("panel %q content.showRun must be a boolean", panel.ID)
		}
	}
	if rawPrompt, ok := panel.Content["promptConfirmation"]; ok && rawPrompt != nil {
		if _, ok := rawPrompt.(bool); !ok {
			return fmt.Errorf("panel %q content.promptConfirmation must be a boolean", panel.ID)
		}
	}
	return nil
}

// validateNodesPanelContent enforces the shape of a plural "nodes" panel.
// `nodes` is an array (possibly empty for newly created panels). Each entry
// must reference a canvas node by id or name; optional fields tighten the
// rendered row (label, purpose description, manual-run button).
func validateNodesPanelContent(panel ConsolePanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateOptionalString(panel.ID, "content.title", panel.Content["title"]); err != nil {
		return err
	}
	rawNodes, ok := panel.Content["nodes"]
	if !ok || rawNodes == nil {
		return fmt.Errorf("panel %q content.nodes must be an array", panel.ID)
	}
	entries, ok := rawNodes.([]any)
	if !ok {
		return fmt.Errorf("panel %q content.nodes must be an array", panel.ID)
	}
	for i, raw := range entries {
		if err := validateNodesPanelEntry(panel.ID, i, raw); err != nil {
			return err
		}
	}
	return nil
}

func validateNodesPanelEntry(panelID string, index int, raw any) error {
	entry, ok := raw.(map[string]any)
	if !ok || entry == nil {
		return fmt.Errorf("panel %q content.nodes[%d] must be an object", panelID, index)
	}
	node, _ := entry["node"].(string)
	if strings.TrimSpace(node) == "" {
		return fmt.Errorf("panel %q content.nodes[%d].node must be a non-empty string", panelID, index)
	}
	for _, key := range []string{"label", "description", "triggerName"} {
		if err := validateOptionalString(panelID, fmt.Sprintf("content.nodes[%d].%s", index, key), entry[key]); err != nil {
			return err
		}
	}
	if rawShowRun, present := entry["showRun"]; present && rawShowRun != nil {
		if _, ok := rawShowRun.(bool); !ok {
			return fmt.Errorf("panel %q content.nodes[%d].showRun must be a boolean", panelID, index)
		}
	}
	if rawPrompt, present := entry["promptConfirmation"]; present && rawPrompt != nil {
		if _, ok := rawPrompt.(bool); !ok {
			return fmt.Errorf("panel %q content.nodes[%d].promptConfirmation must be a boolean", panelID, index)
		}
	}
	if rawFormMode, present := entry["formMode"]; present && rawFormMode != nil {
		mode, ok := rawFormMode.(string)
		if !ok {
			return fmt.Errorf("panel %q content.nodes[%d].formMode must be a string", panelID, index)
		}
		switch mode {
		case ConsoleNodesPanelFormModeModal, ConsoleNodesPanelFormModeInline:
		default:
			return fmt.Errorf(
				"panel %q content.nodes[%d].formMode must be %q or %q",
				panelID, index,
				ConsoleNodesPanelFormModeModal, ConsoleNodesPanelFormModeInline,
			)
		}
	}
	return nil
}

func validateDataSource(panelID string, raw any) error {
	return validateDataSourceField(panelID, "dataSource", raw)
}

// validateDataSourceField is like validateDataSource but lets callers
// override the field-path prefix used in error messages. The multi-number
// metric validator uses this to produce errors like
// `panel "n" metrics[0].dataSource ...` instead of the default
// `panel "n" dataSource ...`.
func validateDataSourceField(panelID, fieldPrefix string, raw any) error {
	ds, ok := raw.(map[string]any)
	if !ok || ds == nil {
		return fmt.Errorf("panel %q %s must be an object", panelID, fieldPrefix)
	}
	kind, _ := ds["kind"].(string)
	switch kind {
	case "memory":
		if _, ok := ds["namespace"].(string); !ok {
			return fmt.Errorf("panel %q %s.namespace must be a string for memory sources", panelID, fieldPrefix)
		}
		if err := validateOptionalString(panelID, fieldPrefix+".fieldPath", ds["fieldPath"]); err != nil {
			return err
		}
	case "executions":
		if err := validateOptionalString(panelID, fieldPrefix+".node", ds["node"]); err != nil {
			return err
		}
		if err := validateOptionalNumber(panelID, fieldPrefix+".limit", ds["limit"]); err != nil {
			return err
		}
	case "runs":
		if err := validateOptionalNumber(panelID, fieldPrefix+".limit", ds["limit"]); err != nil {
			return err
		}
		if err := validateRunStatusesField(panelID, fieldPrefix+".statuses", ds["statuses"]); err != nil {
			return err
		}
		if err := validateRunTriggersField(panelID, fieldPrefix+".triggers", ds["triggers"]); err != nil {
			return err
		}
	default:
		return fmt.Errorf("panel %q %s.kind must be \"memory\", \"executions\", or \"runs\"", panelID, fieldPrefix)
	}
	return nil
}

func validateRender(panelID string, raw any, expectedKind string) (map[string]any, error) {
	return validateRenderField(panelID, "render", raw, expectedKind)
}

// validateRenderField is like validateRender but lets callers override the
// field-path prefix used in error messages.
func validateRenderField(panelID, fieldPrefix string, raw any, expectedKind string) (map[string]any, error) {
	render, ok := raw.(map[string]any)
	if !ok || render == nil {
		return nil, fmt.Errorf("panel %q %s must be an object", panelID, fieldPrefix)
	}
	kind, _ := render["kind"].(string)
	if kind != expectedKind {
		return nil, fmt.Errorf("panel %q %s.kind must be %q", panelID, fieldPrefix, expectedKind)
	}
	return render, nil
}

func validateTablePanelContent(panel ConsolePanel) error {
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
	if !ok {
		return fmt.Errorf("panel %q render.columns must be an array", panel.ID)
	}
	for i, rawColumn := range cols {
		column, ok := rawColumn.(map[string]any)
		if !ok || column == nil {
			return fmt.Errorf("panel %q render.columns[%d] must be an object", panel.ID, i)
		}
		field, ok := column["field"].(string)
		if !ok || field == "" {
			return fmt.Errorf("panel %q render.columns[%d].field must be a non-empty string", panel.ID, i)
		}
		if err := validateTableProgressColumn(panel.ID, i, column); err != nil {
			return err
		}
		if err := validateTableColumnTrend(panel.ID, i, column); err != nil {
			return err
		}
	}
	if err := validateTableWhere(panel.ID, render["where"]); err != nil {
		return err
	}
	if err := validateSort(panel.ID, render["sort"]); err != nil {
		return err
	}
	if err := validateTableRowStyles(panel.ID, render["rowStyles"]); err != nil {
		return err
	}
	return validateTableRowActions(panel.ID, render["rowActions"])
}

// allowedProgressLabels mirrors `WIDGET_PROGRESS_LABELS` on the FE. Keep the
// two in lockstep so a valid YAML round-trips through either side.
var allowedProgressLabels = []string{"none", "number", "percent"}

// validateTableProgressColumn enforces the extra constraints on
// `format: progress` columns: the target expression must be present and the
// label enum, when set, must be one of the allowed values. Other formats
// simply ignore these fields.
func validateTableProgressColumn(panelID string, index int, column map[string]any) error {
	if rawLabel, ok := column["progressLabel"]; ok && rawLabel != nil {
		label, ok := rawLabel.(string)
		if !ok || !slices.Contains(allowedProgressLabels, label) {
			return fmt.Errorf("panel %q render.columns[%d].progressLabel must be one of %s", panelID, index, strings.Join(allowedProgressLabels, "/"))
		}
	}
	format, _ := column["format"].(string)
	if format != "progress" {
		return nil
	}
	target, ok := column["progressTarget"].(string)
	if !ok || strings.TrimSpace(target) == "" {
		return fmt.Errorf("panel %q render.columns[%d].progressTarget must be a non-empty string for progress columns", panelID, index)
	}
	return nil
}

// allowedTrendBetter / allowedTrendDisplay must stay in lockstep with the
// frontend `WIDGET_TREND_BETTER` / `WIDGET_TREND_DISPLAYS` enums in
// `web_src/.../widget/types.ts`.
var (
	allowedTrendBetter  = []string{"up", "down"}
	allowedTrendDisplay = []string{"percent", "value", "none"}
)

func validateTableColumnTrend(panelID string, index int, column map[string]any) error {
	if raw, ok := column["showTrend"]; ok && raw != nil {
		if _, isBool := raw.(bool); !isBool {
			return fmt.Errorf("panel %q render.columns[%d].showTrend must be a boolean", panelID, index)
		}
	}
	if raw, ok := column["trendBetter"]; ok && raw != nil {
		s, isString := raw.(string)
		if !isString || !slices.Contains(allowedTrendBetter, s) {
			return fmt.Errorf("panel %q render.columns[%d].trendBetter must be one of %s", panelID, index, strings.Join(allowedTrendBetter, "/"))
		}
	}
	if raw, ok := column["trendDisplay"]; ok && raw != nil {
		s, isString := raw.(string)
		if !isString || !slices.Contains(allowedTrendDisplay, s) {
			return fmt.Errorf("panel %q render.columns[%d].trendDisplay must be one of %s", panelID, index, strings.Join(allowedTrendDisplay, "/"))
		}
	}
	return nil
}

// allowedRowStyleTones must stay in lockstep with the frontend tone enum
// (`WIDGET_ROW_STYLE_TONES` in `web_src/.../widget/types.ts`). Adding a new
// tone requires updating both lists and the class map.
var allowedRowStyleTones = []string{
	"dimmed",
	"yellow",
	"yellow-soft",
	"orange",
	"orange-soft",
	"red",
	"red-soft",
	"blue",
	"blue-soft",
	"green",
	"green-soft",
}

func validateTableRowStyles(panelID string, raw any) error {
	if raw == nil {
		return nil
	}

	rowStyles, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q render.rowStyles must be an array", panelID)
	}

	allowedOps := []string{"eq", "neq", "contains", "not_contains", "gt", "lt", "exists", "not_exists"}
	for i, rawRule := range rowStyles {
		rule, ok := rawRule.(map[string]any)
		if !ok || rule == nil {
			return fmt.Errorf("panel %q render.rowStyles[%d] must be an object", panelID, i)
		}
		field, ok := rule["field"].(string)
		if !ok || strings.TrimSpace(field) == "" {
			return fmt.Errorf("panel %q render.rowStyles[%d].field must be a non-empty string", panelID, i)
		}
		op, ok := rule["op"].(string)
		if !ok || !slices.Contains(allowedOps, op) {
			return fmt.Errorf("panel %q render.rowStyles[%d].op is not supported", panelID, i)
		}
		tone, ok := rule["tone"].(string)
		if !ok || !slices.Contains(allowedRowStyleTones, tone) {
			return fmt.Errorf("panel %q render.rowStyles[%d].tone must be one of %s", panelID, i, strings.Join(allowedRowStyleTones, "/"))
		}
		if err := validateOptionalString(panelID, fmt.Sprintf("render.rowStyles[%d].value", i), rule["value"]); err != nil {
			return err
		}
	}

	return nil
}

func validateChartPanelContent(panel ConsolePanel) error {
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
	chartType, _ := render["type"].(string)
	if !slices.Contains([]string{"bar", "stacked-bar", "line", "area", "donut"}, chartType) {
		return fmt.Errorf("panel %q render.type must be one of bar/stacked-bar/line/area/donut", panel.ID)
	}
	if xField, ok := render["xField"].(string); !ok || xField == "" {
		return fmt.Errorf("panel %q render.xField must be a non-empty string", panel.ID)
	}
	if err := validateOptionalString(panel.ID, "render.seriesField", render["seriesField"]); err != nil {
		return err
	}
	for _, key := range []string{"xFormat", "yLabel", "yFormat"} {
		if err := validateOptionalString(panel.ID, "render."+key, render[key]); err != nil {
			return err
		}
	}
	series, ok := render["series"].([]any)
	if !ok || len(series) == 0 {
		return fmt.Errorf("panel %q render.series must be a non-empty array", panel.ID)
	}
	for i, rawSeries := range series {
		if err := validateChartSeries(panel.ID, i, rawSeries); err != nil {
			return err
		}
	}
	if legend, ok := render["legend"]; ok && legend != nil {
		legendStr, isString := legend.(string)
		if !isString || !slices.Contains([]string{"auto", "show", "hide"}, legendStr) {
			return fmt.Errorf("panel %q render.legend must be one of auto/show/hide", panel.ID)
		}
	}
	return validateSort(panel.ID, render["sort"])
}

var allowedSortOrders = []string{"asc", "desc"}

// validateSort enforces the shape of the optional `render.sort` widget-level
// sort spec. `field` is a non-empty string (literal path or `{{ expr }}`),
// `order` is an optional asc/desc enum. Mirrors the frontend `validateSort`
// in `web_src/src/pages/app/console/panelTypes.ts`.
func validateSort(panelID string, raw any) error {
	if raw == nil {
		return nil
	}
	sort, ok := raw.(map[string]any)
	if !ok {
		return fmt.Errorf("panel %q render.sort must be an object", panelID)
	}
	field, ok := sort["field"].(string)
	if !ok || strings.TrimSpace(field) == "" {
		return fmt.Errorf("panel %q render.sort.field must be a non-empty string", panelID)
	}
	if order, present := sort["order"]; present && order != nil {
		orderStr, isString := order.(string)
		if !isString || !slices.Contains(allowedSortOrders, orderStr) {
			return fmt.Errorf("panel %q render.sort.order must be one of %s", panelID, strings.Join(allowedSortOrders, "/"))
		}
	}
	return nil
}

func validateChartSeries(panelID string, index int, raw any) error {
	series, ok := raw.(map[string]any)
	if !ok || series == nil {
		return fmt.Errorf("panel %q render.series[%d] must be an object", panelID, index)
	}
	for _, key := range []string{"field", "label", "color", "format", "prefix", "suffix"} {
		if err := validateOptionalString(panelID, fmt.Sprintf("render.series[%d].%s", index, key), series[key]); err != nil {
			return err
		}
	}
	return nil
}

func validateOptionalString(panelID, field string, raw any) error {
	if raw == nil {
		return nil
	}
	if _, ok := raw.(string); !ok {
		return fmt.Errorf("panel %q %s must be a string", panelID, field)
	}
	return nil
}

func validateOptionalNumber(panelID, field string, raw any) error {
	if raw == nil {
		return nil
	}

	var value float64
	switch v := raw.(type) {
	case float64:
		value = v
	case float32:
		value = float64(v)
	case int:
		value = float64(v)
	case int32:
		value = float64(v)
	case int64:
		value = float64(v)
	default:
		return fmt.Errorf("panel %q %s must be a number", panelID, field)
	}

	if math.IsNaN(value) || math.IsInf(value, 0) {
		return fmt.Errorf("panel %q %s must be a number", panelID, field)
	}

	return nil
}

func validateTableWhere(panelID string, raw any) error {
	if raw == nil {
		return nil
	}

	where, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q render.where must be an array", panelID)
	}

	allowedOps := []string{"eq", "neq", "contains", "not_contains", "gt", "lt", "exists", "not_exists"}
	for i, rawFilter := range where {
		filter, ok := rawFilter.(map[string]any)
		if !ok || filter == nil {
			return fmt.Errorf("panel %q render.where[%d] must be an object", panelID, i)
		}
		field, ok := filter["field"].(string)
		if !ok || field == "" {
			return fmt.Errorf("panel %q render.where[%d].field must be a non-empty string", panelID, i)
		}
		op, ok := filter["op"].(string)
		if !ok || !slices.Contains(allowedOps, op) {
			return fmt.Errorf("panel %q render.where[%d].op is not supported", panelID, i)
		}
	}

	return nil
}

func validateTableRowActions(panelID string, raw any) error {
	if raw == nil {
		return nil
	}

	actions, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q render.rowActions must be an array", panelID)
	}

	for i, rawAction := range actions {
		action, ok := rawAction.(map[string]any)
		if !ok || action == nil || action["kind"] != "trigger" {
			return fmt.Errorf("panel %q render.rowActions[%d] must be a trigger action", panelID, i)
		}
		node, _ := action["node"].(string)
		target, _ := action["target"].(string)
		if node == "" && target == "" {
			return fmt.Errorf("panel %q render.rowActions[%d].node must be set to a trigger node", panelID, i)
		}
	}

	return nil
}

func validateNumberPanelContent(panel ConsolePanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}

	// Multi-number mode: each metric carries its own dataSource + render.
	// Top-level dataSource/render are not used and are not required.
	if rawMetrics, ok := panel.Content["metrics"]; ok {
		return validateNumberMetrics(panel.ID, rawMetrics)
	}

	if err := validateNumberDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "number")
	if err != nil {
		return err
	}
	if err := validateOptionalString(panel.ID, "render.prefix", render["prefix"]); err != nil {
		return err
	}
	if err := validateOptionalString(panel.ID, "render.suffix", render["suffix"]); err != nil {
		return err
	}

	// Composite memory sources carry per-source aggregation; render-level
	// aggregation/field must be absent so configuration is unambiguous.
	if isCompositeMemoryDataSource(panel.Content["dataSource"]) {
		if _, hasAgg := render["aggregation"]; hasAgg {
			return fmt.Errorf("panel %q render.aggregation must not be set when dataSource.sources is used (each source defines its own aggregation)", panel.ID)
		}
		if _, hasField := render["field"]; hasField {
			return fmt.Errorf("panel %q render.field must not be set when dataSource.sources is used (each source defines its own field)", panel.ID)
		}
		return nil
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

// validateNumberMetrics validates a multi-number panel's `metrics` array.
// Each metric uses a simple (non-composite) data source plus its own number
// render so the panel can display multiple independently-configured numbers
// in a wrapping row.
func validateNumberMetrics(panelID string, raw any) error {
	metrics, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q metrics must be an array", panelID)
	}
	if len(metrics) == 0 {
		return fmt.Errorf("panel %q metrics must be a non-empty array", panelID)
	}
	for i, item := range metrics {
		if err := validateNumberMetric(panelID, i, item); err != nil {
			return err
		}
	}
	return nil
}

func validateNumberMetric(panelID string, index int, raw any) error {
	metric, ok := raw.(map[string]any)
	if !ok || metric == nil {
		return fmt.Errorf("panel %q metrics[%d] must be an object", panelID, index)
	}
	dsPrefix := fmt.Sprintf("metrics[%d].dataSource", index)
	renderPrefix := fmt.Sprintf("metrics[%d].render", index)
	// Composite memory sources are not allowed inside a multi-number metric;
	// the panel itself already lets users repeat the data source per metric.
	if isCompositeMemoryDataSource(metric["dataSource"]) {
		return fmt.Errorf("panel %q %s must be a single-source memory/executions/runs source", panelID, dsPrefix)
	}
	if err := validateDataSourceField(panelID, dsPrefix, metric["dataSource"]); err != nil {
		return err
	}
	render, err := validateRenderField(panelID, renderPrefix, metric["render"], "number")
	if err != nil {
		return err
	}
	if err := validateOptionalString(panelID, renderPrefix+".prefix", render["prefix"]); err != nil {
		return err
	}
	if err := validateOptionalString(panelID, renderPrefix+".suffix", render["suffix"]); err != nil {
		return err
	}
	aggregation, _ := render["aggregation"].(string)
	if !slices.Contains(allowedNumberAggregations, aggregation) {
		return fmt.Errorf("panel %q %s.aggregation must be one of %s", panelID, renderPrefix, strings.Join(allowedNumberAggregations, "/"))
	}
	if aggregation != "count" {
		if field, ok := render["field"].(string); !ok || field == "" {
			return fmt.Errorf("panel %q %s.field is required when aggregation is %q", panelID, renderPrefix, aggregation)
		}
	}
	return nil
}

func isCompositeMemoryDataSource(raw any) bool {
	ds, ok := raw.(map[string]any)
	if !ok || ds == nil {
		return false
	}
	if ds["kind"] != "memory" {
		return false
	}
	_, ok = ds["sources"].([]any)
	return ok
}

var allowedNumberCombineOps = []string{"sum", "min", "max", "avg"}
var allowedNumberAggregations = []string{"count", "sum", "avg", "min", "max", "first", "last"}

// validateNumberDataSource accepts the shared data-source shapes plus the
// composite memory variant where each namespace has its own aggregation and
// the partials are merged via `combine`.
func validateNumberDataSource(panelID string, raw any) error {
	ds, ok := raw.(map[string]any)
	if !ok || ds == nil {
		return fmt.Errorf("panel %q dataSource must be an object", panelID)
	}
	if ds["kind"] == "memory" {
		if _, hasSources := ds["sources"]; hasSources {
			return validateCompositeMemoryDataSource(panelID, ds)
		}
	}
	return validateDataSource(panelID, raw)
}

func validateCompositeMemoryDataSource(panelID string, ds map[string]any) error {
	sources, ok := ds["sources"].([]any)
	if !ok {
		return fmt.Errorf("panel %q dataSource.sources must be an array", panelID)
	}
	if len(sources) == 0 {
		return fmt.Errorf("panel %q dataSource.sources must be a non-empty array", panelID)
	}
	for i, raw := range sources {
		if err := validateMemoryNumberSource(panelID, i, raw); err != nil {
			return err
		}
	}
	combine, _ := ds["combine"].(string)
	if !slices.Contains(allowedNumberCombineOps, combine) {
		return fmt.Errorf("panel %q dataSource.combine must be one of %s", panelID, strings.Join(allowedNumberCombineOps, "/"))
	}
	return nil
}

func validateMemoryNumberSource(panelID string, index int, raw any) error {
	source, ok := raw.(map[string]any)
	if !ok || source == nil {
		return fmt.Errorf("panel %q dataSource.sources[%d] must be an object", panelID, index)
	}
	namespace, _ := source["namespace"].(string)
	if namespace == "" {
		return fmt.Errorf("panel %q dataSource.sources[%d].namespace must be a non-empty string", panelID, index)
	}
	aggregation, _ := source["aggregation"].(string)
	if !slices.Contains(allowedNumberAggregations, aggregation) {
		return fmt.Errorf("panel %q dataSource.sources[%d].aggregation must be one of %s", panelID, index, strings.Join(allowedNumberAggregations, "/"))
	}
	if aggregation != "count" {
		if field, ok := source["field"].(string); !ok || field == "" {
			return fmt.Errorf("panel %q dataSource.sources[%d].field is required when aggregation is %q", panelID, index, aggregation)
		}
	}
	return validateOptionalString(panelID, fmt.Sprintf("dataSource.sources[%d].fieldPath", index), source["fieldPath"])
}

// allowedScorecardBetter / allowedScorecardShowChange must stay in lockstep
// with the frontend `WIDGET_TREND_BETTER` / `WIDGET_SCORECARD_SHOW_CHANGES`
// enums in `web_src/.../widget/types.ts`.
var (
	allowedScorecardBetter     = []string{"up", "down"}
	allowedScorecardShowChange = []string{"percent", "number", "both", "none"}
)

// validateScorecardPanelContent enforces the shape of a `scorecard` panel.
// It reuses the shared data-source validator (single-source memory /
// executions / runs), and then validates the scorecard-specific render:
// standard number aggregation/field/format plus optional target (literal or
// CEL string), better direction, showChange enum, progress toggle, and
// change caption.
func validateScorecardPanelContent(panel ConsolePanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "scorecard")
	if err != nil {
		return err
	}
	aggregation, _ := render["aggregation"].(string)
	if !slices.Contains(allowedNumberAggregations, aggregation) {
		return fmt.Errorf("panel %q render.aggregation must be one of %s", panel.ID, strings.Join(allowedNumberAggregations, "/"))
	}
	if aggregation != "count" {
		if field, ok := render["field"].(string); !ok || field == "" {
			return fmt.Errorf("panel %q render.field is required when aggregation is %q", panel.ID, aggregation)
		}
	}
	for _, key := range []string{"prefix", "suffix", "label", "format", "sparklineField", "target", "changeCaption"} {
		if err := validateOptionalString(panel.ID, "render."+key, render[key]); err != nil {
			return err
		}
	}
	if raw, ok := render["showProgress"]; ok && raw != nil {
		if _, isBool := raw.(bool); !isBool {
			return fmt.Errorf("panel %q render.showProgress must be a boolean", panel.ID)
		}
	}
	if raw, ok := render["better"]; ok && raw != nil {
		s, isString := raw.(string)
		if !isString || !slices.Contains(allowedScorecardBetter, s) {
			return fmt.Errorf("panel %q render.better must be one of %s", panel.ID, strings.Join(allowedScorecardBetter, "/"))
		}
	}
	if raw, ok := render["showChange"]; ok && raw != nil {
		s, isString := raw.(string)
		if !isString || !slices.Contains(allowedScorecardShowChange, s) {
			return fmt.Errorf("panel %q render.showChange must be one of %s", panel.ID, strings.Join(allowedScorecardShowChange, "/"))
		}
	}
	return nil
}

// normalizeDashboardPanelsForExport ensures stable field order in panel
// content maps so YAML output is deterministic across runs.
func normalizeConsolePanelsForExport(panels []models.ConsolePanel) []ConsolePanel {
	if panels == nil {
		return []ConsolePanel{}
	}

	out := make([]ConsolePanel, len(panels))
	for i, panel := range panels {
		out[i] = ConsolePanel{
			ID:      panel.ID,
			Type:    panel.Type,
			Content: panel.Content,
		}
	}
	return out
}

func normalizeConsoleLayoutForExport(layout []models.ConsoleLayoutItem) []ConsoleLayoutItem {
	if layout == nil {
		return []ConsoleLayoutItem{}
	}

	out := make([]ConsoleLayoutItem, len(layout))
	for i, item := range layout {
		out[i] = ConsoleLayoutItem{
			I:    item.I,
			X:    item.X,
			Y:    item.Y,
			W:    item.W,
			H:    item.H,
			MinW: item.MinW,
			MinH: item.MinH,
		}
	}

	return out
}

func encodedConsolePanelsSize(panels []ConsolePanel) (int, error) {
	encoded, err := json.Marshal(panels)
	if err != nil {
		return 0, err
	}
	return len(encoded), nil
}
