package console

import (
	"fmt"
	"io"
	"text/tabwriter"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

type panelsListCommand struct {
	canvasID *string
}

func (c *panelsListCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(dashboard.GetPanels())
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		panels := dashboard.GetPanels()
		if len(panels) == 0 {
			_, err := fmt.Fprintln(stdout, "No panels.")
			return err
		}
		writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
		_, _ = fmt.Fprintln(writer, "ID\tTYPE\tTITLE")
		for _, panel := range panels {
			_, _ = fmt.Fprintf(writer, "%s\t%s\t%s\n", panel.GetId(), panel.GetType(), panelTitle(panel))
		}
		return writer.Flush()
	})
}

type panelsGetCommand struct {
	canvasID *string
}

func (c *panelsGetCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	if len(ctx.Args) == 0 {
		return fmt.Errorf("panel id is required")
	}
	panelID := ctx.Args[0]

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	for _, panel := range dashboard.GetPanels() {
		if panel.GetId() != panelID {
			continue
		}
		if !ctx.Renderer.IsText() {
			return ctx.Renderer.Render(panel)
		}
		return ctx.Renderer.RenderText(func(stdout io.Writer) error {
			return renderPanelText(stdout, panel)
		})
	}

	return fmt.Errorf("panel %q not found", panelID)
}

func renderPanelText(stdout io.Writer, panel openapi_client.CanvasesDashboardPanel) error {
	if _, err := fmt.Fprintf(stdout, "ID:    %s\n", panel.GetId()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Type:  %s\n", panel.GetType()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Title: %s\n", panelTitle(panel)); err != nil {
		return err
	}
	return renderMapAsYAMLBlock(stdout, "Content", panel.GetContent())
}

type panelsDeleteCommand struct {
	canvasID *string
	yes      *bool
}

func (c *panelsDeleteCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	if len(ctx.Args) == 0 {
		return fmt.Errorf("panel id is required")
	}
	panelID := ctx.Args[0]

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	panels := dashboard.GetPanels()
	layout := dashboard.GetLayout()
	filteredPanels := make([]openapi_client.CanvasesDashboardPanel, 0, len(panels))
	found := false
	for _, panel := range panels {
		if panel.GetId() == panelID {
			found = true
			continue
		}
		filteredPanels = append(filteredPanels, panel)
	}
	if !found {
		return fmt.Errorf("panel %q not found", panelID)
	}

	filteredLayout := make([]openapi_client.CanvasesDashboardLayoutItem, 0, len(layout))
	for _, item := range layout {
		if item.GetI() == panelID {
			continue
		}
		filteredLayout = append(filteredLayout, item)
	}

	if !confirmDeletePanel(ctx, c.yes, panelID) {
		_, err := fmt.Fprintln(ctx.Cmd.OutOrStdout(), "Aborted.")
		return err
	}

	body := openapi_client.CanvasesUpdateCanvasDashboardBody{}
	body.SetPanels(filteredPanels)
	body.SetLayout(filteredLayout)

	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateCanvasDashboard(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.GetDashboard())
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Panel %q deleted from canvas %s\n", panelID, canvasID)
		return err
	})
}

func confirmDeletePanel(ctx core.CommandContext, yes *bool, panelID string) bool {
	if yes != nil && *yes {
		return true
	}
	if !ctx.IsInteractive() {
		return true
	}
	_, _ = fmt.Fprintf(ctx.Cmd.OutOrStdout(), "Delete panel %q? [y/N]: ", panelID)
	var answer string
	_, _ = fmt.Fscanln(ctx.Cmd.InOrStdin(), &answer)
	return answer == "y" || answer == "Y" || answer == "yes" || answer == "YES"
}

type panelsUpsertCommand struct {
	canvasID *string
	file     *string
	layout   *string
}

// panelDocument is the input shape for `console panels upsert`. It is a
// superset of the Console panel/layout JSON in the API to keep the file
// simple: a single document with the panel and an optional `layout` block.
type panelDocument struct {
	APIVersion string                 `json:"apiVersion,omitempty" yaml:"apiVersion,omitempty"`
	Kind       string                 `json:"kind,omitempty" yaml:"kind,omitempty"`
	ID         string                 `json:"id,omitempty" yaml:"id,omitempty"`
	Type       string                 `json:"type,omitempty" yaml:"type,omitempty"`
	Content    map[string]any         `json:"content,omitempty" yaml:"content,omitempty"`
	Layout     *consoleResourceLayout `json:"layout,omitempty" yaml:"layout,omitempty"`
	Panel      *panelDocumentEmbedded `json:"panel,omitempty" yaml:"panel,omitempty"`
}

type panelDocumentEmbedded struct {
	ID      string         `json:"id,omitempty" yaml:"id,omitempty"`
	Type    string         `json:"type,omitempty" yaml:"type,omitempty"`
	Content map[string]any `json:"content,omitempty" yaml:"content,omitempty"`
}

func (c *panelsUpsertCommand) Execute(ctx core.CommandContext) error {
	canvasID, err := core.ResolveCanvasID(ctx, valueOf(c.canvasID))
	if err != nil {
		return err
	}

	doc, err := parsePanelDocument(valueOf(c.file), ctx.Cmd.InOrStdin())
	if err != nil {
		return err
	}

	panel := openapi_client.CanvasesDashboardPanel{}
	panel.SetId(doc.ID)
	panel.SetType(doc.Type)
	if doc.Content != nil {
		panel.SetContent(doc.Content)
	}

	dashboard, err := fetchDashboard(ctx, canvasID)
	if err != nil {
		return err
	}

	updatedPanels := make([]openapi_client.CanvasesDashboardPanel, 0, len(dashboard.GetPanels())+1)
	replaced := false
	for _, existing := range dashboard.GetPanels() {
		if existing.GetId() == doc.ID {
			updatedPanels = append(updatedPanels, panel)
			replaced = true
			continue
		}
		updatedPanels = append(updatedPanels, existing)
	}
	if !replaced {
		updatedPanels = append(updatedPanels, panel)
	}

	updatedLayout := updatedLayoutForPanel(dashboard.GetLayout(), doc, valueOf(c.layout))

	body := openapi_client.CanvasesUpdateCanvasDashboardBody{}
	body.SetPanels(updatedPanels)
	body.SetLayout(updatedLayout)

	response, _, err := ctx.API.CanvasAPI.
		CanvasesUpdateCanvasDashboard(ctx.Context, canvasID).
		Body(body).
		Execute()
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(response.GetDashboard())
	}

	action := "added"
	if replaced {
		action = "updated"
	}
	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		_, err := fmt.Fprintf(stdout, "Panel %q %s on canvas %s\n", doc.ID, action, canvasID)
		return err
	})
}

func parsePanelDocument(path string, stdin io.Reader) (*panelDocument, error) {
	if path == "" {
		return nil, fmt.Errorf("--file is required (use - to read from stdin)")
	}

	doc := panelDocument{}
	data, err := readFileOrStdin(path, stdin)
	if err != nil {
		return nil, err
	}
	if err := core.NewDecoder(data).DecodeYAML(&doc); err != nil {
		return nil, fmt.Errorf("failed to parse panel: %w", err)
	}

	if doc.Panel != nil {
		if doc.ID == "" {
			doc.ID = doc.Panel.ID
		}
		if doc.Type == "" {
			doc.Type = doc.Panel.Type
		}
		if doc.Content == nil {
			doc.Content = doc.Panel.Content
		}
	}

	if doc.ID == "" {
		return nil, fmt.Errorf("panel id is required")
	}
	if doc.Type == "" {
		return nil, fmt.Errorf("panel type is required")
	}
	if !panelTypeIsSupported(doc.Type) {
		return nil, fmt.Errorf("panel %q has unsupported type %q (supported: %s)", doc.ID, doc.Type, joinStrings(supportedPanelTypes, ", "))
	}

	return &doc, nil
}

// updatedLayoutForPanel applies the provided layout (from --layout JSON or
// from the panel document) on top of the existing dashboard layout. If
// neither source is set and there is no existing layout entry for this
// panel, the layout list is left unchanged so the API can fill in defaults
// the same way the UI does on first display.
func updatedLayoutForPanel(existing []openapi_client.CanvasesDashboardLayoutItem, doc *panelDocument, layoutJSON string) []openapi_client.CanvasesDashboardLayoutItem {
	out := make([]openapi_client.CanvasesDashboardLayoutItem, 0, len(existing)+1)
	updated := false

	override := layoutFromDocOrJSON(doc, layoutJSON)
	for _, item := range existing {
		if item.GetI() == doc.ID && override != nil {
			out = append(out, layoutItemForAPI(*override))
			updated = true
			continue
		}
		out = append(out, item)
	}

	if override != nil && !updated {
		out = append(out, layoutItemForAPI(*override))
	}

	return out
}

func layoutFromDocOrJSON(doc *panelDocument, layoutJSON string) *consoleResourceLayout {
	if doc != nil && doc.Layout != nil {
		layout := *doc.Layout
		layout.I = doc.ID
		return &layout
	}
	if layoutJSON == "" {
		return nil
	}
	override := consoleResourceLayout{}
	if err := core.NewDecoder([]byte(layoutJSON)).DecodeYAML(&override); err != nil {
		return nil
	}
	override.I = doc.ID
	return &override
}

func layoutItemForAPI(layout consoleResourceLayout) openapi_client.CanvasesDashboardLayoutItem {
	apiItem := openapi_client.CanvasesDashboardLayoutItem{}
	apiItem.SetI(layout.I)
	apiItem.SetX(layout.X)
	apiItem.SetY(layout.Y)
	apiItem.SetW(layout.W)
	apiItem.SetH(layout.H)
	if layout.MinW != nil {
		apiItem.SetMinW(*layout.MinW)
	}
	if layout.MinH != nil {
		apiItem.SetMinH(*layout.MinH)
	}
	return apiItem
}

func readFileOrStdin(path string, stdin io.Reader) ([]byte, error) {
	if path == "-" {
		return io.ReadAll(stdin)
	}
	// #nosec G304 - file path is supplied by the CLI user.
	return readAndCloseFile(path)
}
