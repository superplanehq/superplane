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
	ConsolePanelTypeBoard     = "board"
	ConsolePanelTypeChart     = "chart"
	ConsolePanelTypeNumber    = "number"
	ConsolePanelTypeScorecard = "scorecard"

	// ConsoleNodesPanelFormMode* control whether a `nodes` panel entry
	// renders its manual-run parameter form as a modal dialog (default)
	// or inline in the panel body. Keep in lockstep with
	// `NODES_PANEL_FORM_MODES` in the frontend `nodesPanelContent.ts`.
	ConsoleNodesPanelFormModeModal  = "modal"
	ConsoleNodesPanelFormModeInline = "inline"

	// MaxConsolePages caps how many tabs a console may have. Existing
	// consoles predate the cap and are grandfathered at read time; any
	// import or commit that goes over is rejected.
	MaxConsolePages = 5
	// MaxConsolePanelsPerPage caps how many panels each page may hold.
	// Same grandfathering rule as MaxConsolePages: over-cap consoles
	// still render, only new saves are blocked.
	MaxConsolePanelsPerPage = 20
	// MaxConsolePayloadBytes bounds the JSON size of a single page's
	// panels[]. Kept the same as before pages existed so the per-page
	// storage envelope does not grow.
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
	ConsolePanelTypeBoard,
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

// Pages returns the canonical multi-page representation regardless of
// whether the source YAML used the legacy single-page shape or the
// `spec.pages[]` shape. Legacy documents are wrapped in a single implicit
// page with id `main` so downstream code has one path to reason about.
func (c *Console) Pages() []models.ConsolePage {
	if len(c.Spec.Pages) > 0 {
		return pagesToModels(c.Spec.Pages)
	}
	panels := c.Spec.Panels
	layout := c.Spec.Layout
	if len(panels) == 0 && len(layout) == 0 {
		return []models.ConsolePage{}
	}
	return []models.ConsolePage{
		{
			ID:     models.DefaultConsolePageID,
			Name:   models.DefaultConsolePageName,
			Panels: consolePanelsToModels(panels),
			Layout: consoleLayoutToModels(layout),
		},
	}
}

type ConsoleMetadata struct {
	CanvasID string `json:"canvasId" yaml:"canvasId"`
	Name     string `json:"name" yaml:"name"`
}

// ConsoleSpec accepts both the legacy single-page shape (top-level
// `panels`/`layout`) and the multi-page shape (`pages`). The two shapes
// are mutually exclusive; a document that mixes them is rejected in
// Validate. Consumers should read the normalized view via Console.Pages()
// rather than these fields directly.
type ConsoleSpec struct {
	Pages  []ConsolePage       `json:"pages,omitempty" yaml:"pages,omitempty"`
	Panels []ConsolePanel      `json:"panels,omitempty" yaml:"panels,omitempty"`
	Layout []ConsoleLayoutItem `json:"layout,omitempty" yaml:"layout,omitempty"`
}

type ConsolePage struct {
	ID     string              `json:"id" yaml:"id"`
	Name   string              `json:"name,omitempty" yaml:"name,omitempty"`
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

func pagesToModels(pages []ConsolePage) []models.ConsolePage {
	out := make([]models.ConsolePage, len(pages))
	for i, page := range pages {
		out[i] = models.ConsolePage{
			ID:     page.ID,
			Name:   page.Name,
			Panels: consolePanelsToModels(page.Panels),
			Layout: consoleLayoutToModels(page.Layout),
		}
	}
	return out
}

func consolePanelsToModels(panels []ConsolePanel) []models.ConsolePanel {
	out := make([]models.ConsolePanel, len(panels))
	for i, panel := range panels {
		out[i] = models.ConsolePanel{
			ID:      panel.ID,
			Type:    panel.Type,
			Content: panel.Content,
		}
	}
	return out
}

func consoleLayoutToModels(layout []ConsoleLayoutItem) []models.ConsoleLayoutItem {
	out := make([]models.ConsoleLayoutItem, len(layout))
	for i, item := range layout {
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

// ConsoleFromYML parses AND validates a console YAML document. Use this
// on save / import paths — commits, CLI writes, install-from-GitHub —
// where invalid input must not be persisted.
//
// For read-side use where an already-stored (potentially grandfathered)
// console must still render even if it now exceeds newer caps, use
// ConsoleFromYMLLenient.
func ConsoleFromYML(raw []byte) (*Console, error) {
	return consoleFromYML(raw, true)
}

// ConsoleFromYMLLenient parses a console YAML document without running
// the cap / uniqueness / schema validation. Structural YAML errors
// (unknown fields, wrong apiVersion, malformed JSON) still surface;
// this variant only skips Validate(). Intended for read paths where a
// pre-cap console has to render.
func ConsoleFromYMLLenient(raw []byte) (*Console, error) {
	return consoleFromYML(raw, false)
}

func consoleFromYML(raw []byte, validate bool) (*Console, error) {
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

	if validate {
		if err := resource.Validate(); err != nil {
			return nil, err
		}
	}

	return &resource, nil
}

func VersionToConsoleYML(canvasName string, canvasVersion *models.CanvasVersion) (string, error) {
	if canvasVersion == nil {
		return "", errors.New("canvas version is required")
	}

	spec := consoleSpecForExport(canvasVersion.ConsolePages.Data())

	// Serialize through map[string]any so `omitempty` on ConsoleSpec
	// does not drop legacy `panels: []` / `layout: []` for empty
	// single-page consoles.
	doc := map[string]any{
		"apiVersion": APIVersion,
		"kind":       KindConsole,
		"metadata":   map[string]any{},
		"spec":       spec,
	}

	jsonBytes, err := json.Marshal(doc)
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

// consoleSpecForExport implements the "legacy-until-multi" export rule:
// the legacy top-level `panels`/`layout` shape is only used when it is
// round-trippable — either the console is empty, or it has a single
// page still using the default id/name. If the sole page was renamed
// or given a custom id, escape to the multi-page `pages[]` shape so
// the rename survives the next save (the parser wraps legacy YAML back
// into the default `main`/`Main` page, which would otherwise silently
// drop a customized id or name).
func consoleSpecForExport(pages []models.ConsolePage) map[string]any {
	if len(pages) == 0 {
		return map[string]any{
			"panels": normalizeConsolePanelsForExport(nil),
			"layout": normalizeConsoleLayoutForExport(nil),
		}
	}

	if len(pages) == 1 && isDefaultConsolePage(pages[0]) {
		return map[string]any{
			"panels": normalizeConsolePanelsForExport(pages[0].Panels),
			"layout": normalizeConsoleLayoutForExport(pages[0].Layout),
		}
	}

	out := make([]map[string]any, len(pages))
	for i, page := range pages {
		entry := map[string]any{
			"id":     page.ID,
			"panels": normalizeConsolePanelsForExport(page.Panels),
			"layout": normalizeConsoleLayoutForExport(page.Layout),
		}
		if page.Name != "" {
			entry["name"] = page.Name
		}
		out[i] = entry
	}
	return map[string]any{"pages": out}
}

func isDefaultConsolePage(page models.ConsolePage) bool {
	if page.ID != models.DefaultConsolePageID {
		return false
	}
	if page.Name == "" {
		return true
	}
	return page.Name == models.DefaultConsolePageName
}

func (c *Console) Validate() error {
	if err := c.ValidateShape(); err != nil {
		return err
	}

	if len(c.Spec.Pages) > 0 {
		return validateConsolePages(c.Spec.Pages)
	}

	return ValidateConsoleContent(c.Spec.Panels, c.Spec.Layout)
}

// ValidateShape enforces the document-level invariants that must hold
// even for grandfathered consoles: apiVersion, kind, mutual exclusion
// of the two spec shapes, and per-page structural checks (unique
// page/panel ids, allowed panel types, well-formed panel content,
// layout references). Cap-based rules (page count, per-page panel
// count, payload size) are intentionally *not* enforced here — they
// are the caller's responsibility, either via full Validate() (strict
// authoring) or via ValidateConsolePagesDelta (write paths that want
// to allow reducing an over-cap page without wedging on a hard cap).
//
// Write paths (CLI `apps console set`, commit_canvas_staging, install)
// use ConsoleFromYMLLenient + ValidateShape + ValidateConsolePagesDelta
// so a grandfathered console can be re-committed while a malformed
// document (wrong kind/apiVersion, mixed spec shapes, unknown panel
// type) still fails fast.
func (c *Console) ValidateShape() error {
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

	// The two shapes are exposed as separate fields on ConsoleSpec so
	// legacy documents keep parsing cleanly, but a single document must
	// pick one. Mixing them would be ambiguous (which set is the source
	// of truth for the first page?) so we reject it before validating
	// any content.
	hasLegacy := c.Spec.Panels != nil || c.Spec.Layout != nil
	if len(c.Spec.Pages) > 0 && hasLegacy {
		return errors.New("spec.pages cannot be combined with top-level spec.panels or spec.layout")
	}

	if len(c.Spec.Pages) > 0 {
		return validateConsolePagesStructural(c.Spec.Pages)
	}
	return validateConsoleContentStructure(c.Spec.Panels, c.Spec.Layout)
}

// validateConsolePagesStructural enforces the per-page structural
// invariants (unique/non-empty page ids, structural panel/layout
// checks per page) without applying the page-count cap. Shared by
// ValidateShape (write-path shape gate) and by ValidateConsolePagesDelta
// (which then layers its own delta-aware cap enforcement on top).
func validateConsolePagesStructural(pages []ConsolePage) error {
	pageIDs := make(map[string]struct{}, len(pages))
	for i, page := range pages {
		if strings.TrimSpace(page.ID) == "" {
			return fmt.Errorf("pages[%d].id is required", i)
		}
		if _, exists := pageIDs[page.ID]; exists {
			return fmt.Errorf("duplicate page id %q", page.ID)
		}
		pageIDs[page.ID] = struct{}{}

		if err := validateConsoleContentStructure(page.Panels, page.Layout); err != nil {
			return fmt.Errorf("page %q: %w", page.ID, err)
		}
	}
	return nil
}

// validateConsolePages enforces the invariants that apply across pages
// (page count cap, unique ids) and then delegates per-page panel/layout
// validation to the shared ValidateConsoleContent so both shapes share
// the same panel type / content rules.
func validateConsolePages(pages []ConsolePage) error {
	if len(pages) > MaxConsolePages {
		return fmt.Errorf("too many pages (max %d)", MaxConsolePages)
	}

	pageIDs := make(map[string]struct{}, len(pages))
	for i, page := range pages {
		if strings.TrimSpace(page.ID) == "" {
			return fmt.Errorf("pages[%d].id is required", i)
		}
		if _, exists := pageIDs[page.ID]; exists {
			return fmt.Errorf("duplicate page id %q", page.ID)
		}
		pageIDs[page.ID] = struct{}{}

		if err := ValidateConsoleContent(page.Panels, page.Layout); err != nil {
			return fmt.Errorf("page %q: %w", page.ID, err)
		}
	}

	return nil
}

// ValidateConsolePagesDelta enforces the page/panel caps but grandfathers
// content that was already over-cap in `previous`. Rules:
//
//   - Page count: reject if new count > cap AND new count > previous count.
//     A migrated console with 6 pages stays valid so long as it does not
//     grow further; the user can add pages again only after reducing to
//     the cap.
//   - Per-page panel count: reject if the new count > cap AND the new
//     count > the previous count for either the same id OR the same
//     positional slot. The positional fallback keeps a grandfathered
//     page valid across renames (e.g. `main` → `overview`) that do not
//     change the panel count. Newly-introduced pages that share neither
//     id nor position with any previous page must be at-or-under the
//     cap on their first commit.
//   - Structural checks (missing/duplicate ids, unknown panel types,
//     malformed content) always run.
//
// The function accepts `[]models.ConsolePage` because that is the shape
// on both sides at commit time — `yaml.Console.Pages()` returns models
// pages, and `CanvasVersion.ConsolePages.Data()` stores them. This lets
// grandfathered consoles progressively reduce their size without
// wedging the commit path while still preventing brand-new content from
// silently exceeding the cap.
func ValidateConsolePagesDelta(pages []models.ConsolePage, previous []models.ConsolePage) error {
	if err := validateModelPagesStructure(pages); err != nil {
		return err
	}

	if len(pages) > MaxConsolePages && len(pages) > len(previous) {
		return fmt.Errorf("too many pages (max %d)", MaxConsolePages)
	}

	// Pair each new page with its exact-id match in `previous` (if
	// any) and record which previous indices are claimed by that
	// pairing. A previous slot must be inherited by at most one new
	// page — otherwise a user with a grandfathered over-cap page
	// could keep that page *and* add a positional twin that inherits
	// the same allowance, effectively duplicating over-cap content
	// past the delta cap. See `isGrandfatheredOverCapPage` for how
	// this map is consumed on the positional (rename) fallback.
	previousIndexByID := make(map[string]int, len(previous))
	for i, page := range previous {
		previousIndexByID[page.ID] = i
	}
	claimedPreviousIdx := make(map[int]struct{}, len(pages))
	exactIDMatch := make(map[int]int, len(pages))
	for i, page := range pages {
		if prevIdx, ok := previousIndexByID[page.ID]; ok {
			claimedPreviousIdx[prevIdx] = struct{}{}
			exactIDMatch[i] = prevIdx
		}
	}

	for i, page := range pages {
		if err := validateConsoleContentStructure(fromModelPanels(page.Panels), fromModelLayout(page.Layout)); err != nil {
			return fmt.Errorf("page %q: %w", page.ID, err)
		}
		newCount := len(page.Panels)
		if newCount <= MaxConsolePanelsPerPage {
			continue
		}

		if isGrandfatheredOverCapPage(page, i, previous, exactIDMatch, claimedPreviousIdx) {
			continue
		}
		return fmt.Errorf("page %q: too many panels (max %d per page)", page.ID, MaxConsolePanelsPerPage)
	}

	return nil
}

// isGrandfatheredOverCapPage reports whether an over-cap page in the
// new document is inherited from `previous` (and therefore allowed to
// stay over the cap) rather than freshly authored. Two match modes:
//
//   - Exact id match: allowance = previous count for the same id.
//   - Positional match with panel-id-subset check: same position, and
//     every panel id in the new page also existed in the previous page
//     at that position. This recognizes renames like `main` → `overview`
//     while refusing "fresh" over-cap pages that happen to reuse a
//     grandfathered slot.
//
// A page is grandfathered when its panel count is at-or-under the
// allowance for whichever mode matches. Each previous slot can back
// at most one new page: `claimedPreviousIdx` records slots already
// pinned by an exact-id match elsewhere in this document, and the
// positional fallback refuses those. Without that claim check, a
// grandfathered over-cap page could be positionally duplicated into
// a new tab that also keeps the original id — bypassing the delta
// cap despite both pages carrying inflated panel counts.
func isGrandfatheredOverCapPage(page models.ConsolePage, index int, previous []models.ConsolePage, exactIDMatch map[int]int, claimedPreviousIdx map[int]struct{}) bool {
	if prevIdx, ok := exactIDMatch[index]; ok {
		return len(page.Panels) <= len(previous[prevIdx].Panels)
	}

	if index >= len(previous) {
		return false
	}
	if _, claimed := claimedPreviousIdx[index]; claimed {
		return false
	}
	prevPage := previous[index]
	if len(page.Panels) > len(prevPage.Panels) {
		return false
	}

	prevPanelIDs := make(map[string]struct{}, len(prevPage.Panels))
	for _, p := range prevPage.Panels {
		prevPanelIDs[p.ID] = struct{}{}
	}
	for _, p := range page.Panels {
		if _, ok := prevPanelIDs[p.ID]; !ok {
			return false
		}
	}
	return true
}

// validateModelPagesStructure checks the page-level invariants that hold
// regardless of cap grandfathering (unique / non-empty ids). The cap
// check itself is handled by the caller.
func validateModelPagesStructure(pages []models.ConsolePage) error {
	pageIDs := make(map[string]struct{}, len(pages))
	for i, page := range pages {
		if strings.TrimSpace(page.ID) == "" {
			return fmt.Errorf("pages[%d].id is required", i)
		}
		if _, exists := pageIDs[page.ID]; exists {
			return fmt.Errorf("duplicate page id %q", page.ID)
		}
		pageIDs[page.ID] = struct{}{}
	}
	return nil
}

func fromModelPanels(panels []models.ConsolePanel) []ConsolePanel {
	out := make([]ConsolePanel, len(panels))
	for i, p := range panels {
		out[i] = ConsolePanel{ID: p.ID, Type: p.Type, Content: p.Content}
	}
	return out
}

func fromModelLayout(layout []models.ConsoleLayoutItem) []ConsoleLayoutItem {
	out := make([]ConsoleLayoutItem, len(layout))
	for i, l := range layout {
		out[i] = ConsoleLayoutItem{I: l.I, X: l.X, Y: l.Y, W: l.W, H: l.H, MinW: l.MinW, MinH: l.MinH}
	}
	return out
}

func ValidateConsoleContent(panels []ConsolePanel, layout []ConsoleLayoutItem) error {
	if len(panels) > MaxConsolePanelsPerPage {
		return fmt.Errorf("too many panels (max %d per page)", MaxConsolePanelsPerPage)
	}
	return validateConsoleContentStructure(panels, layout)
}

// validateConsoleContentStructure runs all per-page invariants except
// the panel-count cap: panel id uniqueness/type/content, layout id
// coverage, payload size. Callers that need cap grandfathering
// (ValidateConsolePagesDelta) run this instead of ValidateConsoleContent
// and enforce their own cap rule.
func validateConsoleContentStructure(panels []ConsolePanel, layout []ConsoleLayoutItem) error {
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
	case ConsolePanelTypeBoard:
		return validateBoardPanelContent(panel)
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
	if rawConcurrent, ok := panel.Content["allowConcurrentRuns"]; ok && rawConcurrent != nil {
		if _, ok := rawConcurrent.(bool); !ok {
			return fmt.Errorf("panel %q content.allowConcurrentRuns must be a boolean", panel.ID)
		}
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
	for _, key := range []string{"label", "description", "triggerName", "submitLabel"} {
		if err := validateOptionalString(panelID, fmt.Sprintf("content.nodes[%d].%s", index, key), entry[key]); err != nil {
			return err
		}
	}
	for _, key := range []string{"showRun", "promptConfirmation", "showNodeLabel", "showFieldLabels"} {
		rawValue, present := entry[key]
		if present && rawValue != nil {
			if _, ok := rawValue.(bool); !ok {
				return fmt.Errorf("panel %q content.nodes[%d].%s must be a boolean", panelID, index, key)
			}
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

// AllowedBoardLaneColors mirrors `WIDGET_BOARD_LANE_COLORS` on the FE.
// Keep in lockstep with `web_src/src/pages/app/console/widget/types.ts`.
var AllowedBoardLaneColors = []string{
	"neutral",
	"gray",
	"blue",
	"green",
	"yellow",
	"orange",
	"red",
	"purple",
}

var allowedBoardCardFormats = []string{
	"text",
	"number",
	"percent",
	"date",
	"datetime",
	"relative",
	"duration",
	"status",
	"badge",
	"code",
	"link",
}

// validateBoardPanelContent enforces the shape of a `board` (kanban) panel.
// Structure mirrors `validateBoardContent` in
// `web_src/src/pages/app/console/boardPanelContent.ts` so the frontend and
// backend validators can never drift.
func validateBoardPanelContent(panel ConsolePanel) error {
	if panel.Content == nil {
		return fmt.Errorf("panel %q content is required", panel.ID)
	}
	if err := validateOptionalString(panel.ID, "content.title", panel.Content["title"]); err != nil {
		return err
	}
	if err := validateDataSource(panel.ID, panel.Content["dataSource"]); err != nil {
		return err
	}
	render, err := validateRender(panel.ID, panel.Content["render"], "board")
	if err != nil {
		return err
	}
	groupBy, ok := render["groupBy"].(string)
	if !ok || strings.TrimSpace(groupBy) == "" {
		return fmt.Errorf("panel %q render.groupBy must be a non-empty string", panel.ID)
	}
	if err := validateBoardLanes(panel.ID, render["lanes"]); err != nil {
		return err
	}
	if raw, ok := render["otherLane"]; ok && raw != nil {
		if _, isBool := raw.(bool); !isBool {
			return fmt.Errorf("panel %q render.otherLane must be a boolean", panel.ID)
		}
	}
	if err := validateBoardCard(panel.ID, render["card"]); err != nil {
		return err
	}
	if err := validateTableWhere(panel.ID, render["where"]); err != nil {
		return err
	}
	if err := validateSort(panel.ID, render["sort"]); err != nil {
		return err
	}
	if err := validateTableRowActions(panel.ID, render["rowActions"]); err != nil {
		return err
	}
	return validateOptionalString(panel.ID, "render.emptyMessage", render["emptyMessage"])
}

func validateBoardLanes(panelID string, raw any) error {
	lanes, ok := raw.([]any)
	if !ok || len(lanes) == 0 {
		return fmt.Errorf("panel %q render.lanes must be a non-empty array", panelID)
	}
	seenValues := map[string]struct{}{}
	for i, rawLane := range lanes {
		lane, ok := rawLane.(map[string]any)
		if !ok || lane == nil {
			return fmt.Errorf("panel %q render.lanes[%d] must be an object", panelID, i)
		}
		value, ok := lane["value"].(string)
		if !ok || strings.TrimSpace(value) == "" {
			return fmt.Errorf("panel %q render.lanes[%d].value must be a non-empty string", panelID, i)
		}
		normalizedValue := strings.ToLower(strings.TrimSpace(value))
		if _, exists := seenValues[normalizedValue]; exists {
			return fmt.Errorf("panel %q render.lanes[%d].value must be unique after trimming and case folding", panelID, i)
		}
		seenValues[normalizedValue] = struct{}{}
		if err := validateOptionalString(panelID, fmt.Sprintf("render.lanes[%d].label", i), lane["label"]); err != nil {
			return err
		}
		if rawColor, ok := lane["color"]; ok && rawColor != nil {
			color, isString := rawColor.(string)
			if !isString || !slices.Contains(AllowedBoardLaneColors, color) {
				return fmt.Errorf("panel %q render.lanes[%d].color must be one of %s", panelID, i, strings.Join(AllowedBoardLaneColors, "/"))
			}
		}
	}
	return nil
}

func validateBoardCard(panelID string, raw any) error {
	card, ok := raw.(map[string]any)
	if !ok || card == nil {
		return fmt.Errorf("panel %q render.card must be an object", panelID)
	}
	titleField, ok := card["titleField"].(string)
	if !ok || strings.TrimSpace(titleField) == "" {
		return fmt.Errorf("panel %q render.card.titleField must be a non-empty string", panelID)
	}
	if raw, ok := card["fields"]; ok && raw != nil {
		return validateBoardCardFields(panelID, raw)
	}
	return nil
}

func validateBoardCardFields(panelID string, raw any) error {
	fields, ok := raw.([]any)
	if !ok {
		return fmt.Errorf("panel %q render.card.fields must be an array", panelID)
	}
	for i, rawField := range fields {
		field, ok := rawField.(map[string]any)
		if !ok || field == nil {
			return fmt.Errorf("panel %q render.card.fields[%d] must be an object", panelID, i)
		}
		name, ok := field["field"].(string)
		if !ok || strings.TrimSpace(name) == "" {
			return fmt.Errorf("panel %q render.card.fields[%d].field must be a non-empty string", panelID, i)
		}
		for _, key := range []string{"label", "href", "show", "format"} {
			if err := validateOptionalString(panelID, fmt.Sprintf("render.card.fields[%d].%s", i, key), field[key]); err != nil {
				return err
			}
		}
		if rawFormat, ok := field["format"]; ok && rawFormat != nil {
			format := rawFormat.(string)
			if !slices.Contains(allowedBoardCardFormats, format) {
				return fmt.Errorf("panel %q render.card.fields[%d].format must be one of %s", panelID, i, strings.Join(allowedBoardCardFormats, "/"))
			}
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
