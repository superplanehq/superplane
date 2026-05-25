package console

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
	"gopkg.in/yaml.v3"
)

// ConsoleKind is the canonical YAML kind for the SuperPlane Console resource.
//
// Console is the user-facing name; the backend still calls this resource
// "Dashboard" internally and on the gRPC/REST API. The CLI keeps the
// user-facing name aligned with the UI and product documentation.
const ConsoleKind = "Console"

// supportedPanelTypes mirrors `pkg/models/canvas_dashboard_yml.go` and the
// frontend `web_src/src/pages/workflowv2/dashboard/panelTypes.ts`. Updating
// this list requires updating those callers in lockstep.
var supportedPanelTypes = []string{
	"markdown",
	"node",
	"table",
	"chart",
	"number",
}

// ConsoleResourceMetadata is informational only on import. `canvasId` is
// resolved from the active CLI context or `--canvas-id`; the field on the
// resource is preserved on export so files round-trip cleanly.
type ConsoleResourceMetadata struct {
	CanvasID string `json:"canvasId,omitempty" yaml:"canvasId,omitempty"`
	Name     string `json:"name,omitempty" yaml:"name,omitempty"`
}

// ConsoleResourceSpec carries the panel and layout payload exactly as it is
// stored server-side. Map values are kept as `map[string]any` so panel
// content stays type-agnostic and forward-compatible.
type ConsoleResourceSpec struct {
	Panels []consoleResourcePanel  `json:"panels" yaml:"panels"`
	Layout []consoleResourceLayout `json:"layout" yaml:"layout"`
}

// ConsoleResource is the canonical YAML representation of a Console.
type ConsoleResource struct {
	APIVersion string                  `json:"apiVersion" yaml:"apiVersion"`
	Kind       string                  `json:"kind" yaml:"kind"`
	Metadata   ConsoleResourceMetadata `json:"metadata,omitempty" yaml:"metadata,omitempty"`
	Spec       ConsoleResourceSpec     `json:"spec" yaml:"spec"`
}

type consoleResourcePanel struct {
	ID      string         `json:"id" yaml:"id"`
	Type    string         `json:"type" yaml:"type"`
	Content map[string]any `json:"content,omitempty" yaml:"content,omitempty"`
}

type consoleResourceLayout struct {
	I    string `json:"i" yaml:"i"`
	X    int32  `json:"x" yaml:"x"`
	Y    int32  `json:"y" yaml:"y"`
	W    int32  `json:"w" yaml:"w"`
	H    int32  `json:"h" yaml:"h"`
	MinW *int32 `json:"minW,omitempty" yaml:"minW,omitempty"`
	MinH *int32 `json:"minH,omitempty" yaml:"minH,omitempty"`
}

// dashboardToResource converts the API Console (Dashboard) shape into the
// CLI YAML resource. Panels and layout are preserved as-is so that what is
// written matches what the API returns.
func dashboardToResource(dashboard openapi_client.CanvasesCanvasDashboard, canvasName string) ConsoleResource {
	resource := ConsoleResource{
		APIVersion: core.APIVersion,
		Kind:       ConsoleKind,
		Metadata: ConsoleResourceMetadata{
			CanvasID: dashboard.GetCanvasId(),
			Name:     canvasName,
		},
		Spec: ConsoleResourceSpec{
			Panels: make([]consoleResourcePanel, 0, len(dashboard.GetPanels())),
			Layout: make([]consoleResourceLayout, 0, len(dashboard.GetLayout())),
		},
	}

	for _, panel := range dashboard.GetPanels() {
		resource.Spec.Panels = append(resource.Spec.Panels, consoleResourcePanel{
			ID:      panel.GetId(),
			Type:    panel.GetType(),
			Content: panel.GetContent(),
		})
	}

	for _, item := range dashboard.GetLayout() {
		layout := consoleResourceLayout{
			I: item.GetI(),
			X: item.GetX(),
			Y: item.GetY(),
			W: item.GetW(),
			H: item.GetH(),
		}
		if item.HasMinW() {
			minW := item.GetMinW()
			layout.MinW = &minW
		}
		if item.HasMinH() {
			minH := item.GetMinH()
			layout.MinH = &minH
		}
		resource.Spec.Layout = append(resource.Spec.Layout, layout)
	}

	return resource
}

// resourceToUpdateBody converts the CLI resource into the API update body.
// Empty slices are sent so the API replace-all semantics produce a clean
// state when callers explicitly clear panels or layout.
func resourceToUpdateBody(resource ConsoleResource) openapi_client.CanvasesUpdateCanvasDashboardBody {
	body := openapi_client.CanvasesUpdateCanvasDashboardBody{}
	body.SetPanels(resourcePanelsToAPI(resource.Spec.Panels))
	body.SetLayout(resourceLayoutToAPI(resource.Spec.Layout))
	return body
}

func resourcePanelsToAPI(panels []consoleResourcePanel) []openapi_client.CanvasesDashboardPanel {
	out := make([]openapi_client.CanvasesDashboardPanel, 0, len(panels))
	for _, panel := range panels {
		apiPanel := openapi_client.CanvasesDashboardPanel{}
		apiPanel.SetId(panel.ID)
		apiPanel.SetType(panel.Type)
		if panel.Content != nil {
			apiPanel.SetContent(panel.Content)
		}
		out = append(out, apiPanel)
	}
	return out
}

func resourceLayoutToAPI(layout []consoleResourceLayout) []openapi_client.CanvasesDashboardLayoutItem {
	out := make([]openapi_client.CanvasesDashboardLayoutItem, 0, len(layout))
	for _, item := range layout {
		apiItem := openapi_client.CanvasesDashboardLayoutItem{}
		apiItem.SetI(item.I)
		apiItem.SetX(item.X)
		apiItem.SetY(item.Y)
		apiItem.SetW(item.W)
		apiItem.SetH(item.H)
		if item.MinW != nil {
			apiItem.SetMinW(*item.MinW)
		}
		if item.MinH != nil {
			apiItem.SetMinH(*item.MinH)
		}
		out = append(out, apiItem)
	}
	return out
}

// resourceFromInput reads YAML from either a file path or stdin (`-`),
// validates it as a Console resource, and returns a parsed ConsoleResource.
func resourceFromInput(path string, stdin io.Reader) (*ConsoleResource, error) {
	if path == "" {
		return nil, errors.New("--file is required (use - to read from stdin)")
	}

	var data []byte
	var err error
	if path == "-" {
		data, err = io.ReadAll(stdin)
		if err != nil {
			return nil, fmt.Errorf("failed to read from stdin: %w", err)
		}
	} else {
		// #nosec G304 - file path is supplied by the CLI user.
		data, err = os.ReadFile(path)
		if err != nil {
			return nil, fmt.Errorf("failed to read console file: %w", err)
		}
	}

	apiVersion, kind, err := core.ParseYamlResourceHeaders(data)
	if err != nil {
		return nil, err
	}
	if apiVersion != core.APIVersion {
		return nil, fmt.Errorf("unsupported apiVersion %q (expected %q)", apiVersion, core.APIVersion)
	}
	if kind != ConsoleKind {
		return nil, fmt.Errorf("unsupported resource kind %q (expected %q)", kind, ConsoleKind)
	}

	resource := ConsoleResource{}
	if err := core.NewDecoder(data).DecodeYAML(&resource); err != nil {
		return nil, fmt.Errorf("failed to parse console resource: %w", err)
	}

	if err := validateConsoleResource(&resource); err != nil {
		return nil, err
	}

	return &resource, nil
}

// validateConsoleResource enforces the same panel/layout invariants the
// backend uses on import (panel IDs unique, layout entries reference panels,
// types are in the supported set). Detailed per-type content validation is
// intentionally left to the API so the CLI does not drift from the server.
func validateConsoleResource(resource *ConsoleResource) error {
	if resource == nil {
		return errors.New("console resource is empty")
	}

	panelIDs := make(map[string]struct{}, len(resource.Spec.Panels))
	for _, panel := range resource.Spec.Panels {
		if panel.ID == "" {
			return errors.New("panel id is required")
		}
		if panel.Type == "" {
			return fmt.Errorf("panel %q type is required", panel.ID)
		}
		if !panelTypeIsSupported(panel.Type) {
			return fmt.Errorf("panel %q has unsupported type %q (supported: %s)", panel.ID, panel.Type, joinStrings(supportedPanelTypes, ", "))
		}
		if _, exists := panelIDs[panel.ID]; exists {
			return fmt.Errorf("duplicate panel id %q", panel.ID)
		}
		panelIDs[panel.ID] = struct{}{}
	}

	layoutIDs := make(map[string]struct{}, len(resource.Spec.Layout))
	for _, item := range resource.Spec.Layout {
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

// renderConsoleResourceYAML serializes a Console resource into stable YAML
// suitable for writing to disk. The output round-trips through json so map
// keys are sorted in insertion-friendly order, matching the canonical
// behavior used by the API.
func renderConsoleResourceYAML(resource ConsoleResource) ([]byte, error) {
	jsonBytes, err := json.Marshal(resource)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize console: %w", err)
	}

	var generic any
	if err := json.Unmarshal(jsonBytes, &generic); err != nil {
		return nil, fmt.Errorf("failed to serialize console: %w", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetIndent(2)
	if err := encoder.Encode(generic); err != nil {
		return nil, fmt.Errorf("failed to encode console yaml: %w", err)
	}
	if err := encoder.Close(); err != nil {
		return nil, fmt.Errorf("failed to encode console yaml: %w", err)
	}
	return buf.Bytes(), nil
}

func panelTypeIsSupported(panelType string) bool {
	for _, t := range supportedPanelTypes {
		if t == panelType {
			return true
		}
	}
	return false
}

func joinStrings(values []string, separator string) string {
	out := ""
	for i, v := range values {
		if i > 0 {
			out += separator
		}
		out += v
	}
	return out
}

// findCanvasName looks up the canvas name for export metadata. The error is
// non-fatal — when name resolution fails the export still succeeds with an
// empty `metadata.name`, matching the UI behavior.
func findCanvasName(ctx core.CommandContext, canvasID string) string {
	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil || response.Canvas == nil || response.Canvas.Metadata == nil {
		return ""
	}
	return response.Canvas.Metadata.GetName()
}
