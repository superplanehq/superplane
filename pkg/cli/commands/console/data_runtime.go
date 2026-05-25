package console

import (
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
	"time"

	"github.com/superplanehq/superplane/pkg/cli/core"
	"github.com/superplanehq/superplane/pkg/openapi_client"
)

const (
	defaultDataSourceLimit       = 50
	maxExecutionsPagesPerCommand = 20
	executionsPageSize           = 25
)

type dataCommand struct {
	canvasID *string
	limit    *int64
}

// Execute fetches the runtime data backing a Console panel.
//
// The supported panel types are `table`, `chart`, `number`. Each one stores
// a `dataSource` block on the panel content. We resolve that block server-
// side using the same APIs the UI uses (memory, runs, events with embedded
// executions) and emit the rows as JSON/YAML, or a text-friendly summary.
//
// Markdown and node panels do not have a data source; we surface a clear
// error rather than silently returning an empty list.
func (c *dataCommand) Execute(ctx core.CommandContext) error {
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

	panel, ok := findPanel(dashboard, panelID)
	if !ok {
		return fmt.Errorf("panel %q not found", panelID)
	}

	dataSource, err := dataSourceFromPanel(panel)
	if err != nil {
		return err
	}

	overrideLimit := int64(0)
	if c.limit != nil {
		overrideLimit = *c.limit
	}

	rows, summary, err := fetchPanelData(ctx, canvasID, dataSource, overrideLimit)
	if err != nil {
		return err
	}

	if !ctx.Renderer.IsText() {
		return ctx.Renderer.Render(map[string]any{
			"panelId":    panel.GetId(),
			"type":       panel.GetType(),
			"dataSource": dataSource,
			"summary":    summary,
			"rows":       rows,
		})
	}

	return ctx.Renderer.RenderText(func(stdout io.Writer) error {
		return renderPanelDataText(stdout, panel, dataSource, summary, rows)
	})
}

// panelDataSummary captures totals reported by the API alongside the data.
//
// `totalCount` is populated only for sources that report it (currently
// `runs`); other sources leave it at zero so the renderer omits it.
type panelDataSummary struct {
	Source     string `json:"source"`
	RowCount   int    `json:"rowCount"`
	TotalCount int    `json:"totalCount,omitempty"`
}

func findPanel(dashboard openapi_client.CanvasesCanvasDashboard, panelID string) (openapi_client.CanvasesDashboardPanel, bool) {
	for _, panel := range dashboard.GetPanels() {
		if panel.GetId() == panelID {
			return panel, true
		}
	}
	return openapi_client.CanvasesDashboardPanel{}, false
}

// dataSourceFromPanel pulls the `dataSource` block off a panel's content.
// Markdown/node panels return a clear error instead of an empty source so
// the user knows why the command refuses.
func dataSourceFromPanel(panel openapi_client.CanvasesDashboardPanel) (map[string]any, error) {
	switch panel.GetType() {
	case "markdown":
		return nil, fmt.Errorf("panel %q is a markdown panel and has no data source", panel.GetId())
	case "node":
		return nil, fmt.Errorf("panel %q is a node panel; use 'superplane console trigger --node <id>' to invoke its run hook", panel.GetId())
	}

	content := panel.GetContent()
	if content == nil {
		return nil, fmt.Errorf("panel %q has no content", panel.GetId())
	}
	rawDataSource, ok := content["dataSource"]
	if !ok {
		return nil, fmt.Errorf("panel %q has no dataSource block", panel.GetId())
	}
	ds, ok := rawDataSource.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("panel %q dataSource must be an object", panel.GetId())
	}
	return ds, nil
}

func fetchPanelData(
	ctx core.CommandContext,
	canvasID string,
	dataSource map[string]any,
	overrideLimit int64,
) ([]map[string]any, panelDataSummary, error) {
	kind, _ := dataSource["kind"].(string)
	switch kind {
	case "memory":
		return fetchMemoryRows(ctx, canvasID, dataSource)
	case "runs":
		return fetchRunRows(ctx, canvasID, dataSource, overrideLimit)
	case "executions":
		return fetchExecutionRows(ctx, canvasID, dataSource, overrideLimit)
	default:
		return nil, panelDataSummary{}, fmt.Errorf("unsupported dataSource.kind %q", kind)
	}
}

// fetchMemoryRows returns the entries for a single memory namespace, then
// flattens `fieldPath` like the UI does (lists are spread, scalars become
// single-row entries) so the output matches what the user sees in the UI.
func fetchMemoryRows(
	ctx core.CommandContext,
	canvasID string,
	dataSource map[string]any,
) ([]map[string]any, panelDataSummary, error) {
	namespace, _ := dataSource["namespace"].(string)
	if namespace == "" {
		return nil, panelDataSummary{}, fmt.Errorf("dataSource.namespace is required for memory sources")
	}
	fieldPath, _ := dataSource["fieldPath"].(string)

	response, _, err := ctx.API.CanvasAPI.CanvasesListCanvasMemories(ctx.Context, canvasID).Execute()
	if err != nil {
		return nil, panelDataSummary{}, err
	}

	rows := []map[string]any{}
	for _, memory := range response.GetItems() {
		if memory.GetNamespace() != namespace {
			continue
		}
		expanded := flattenMemoryEntry(memory.GetValues(), fieldPath)
		rows = append(rows, expanded...)
	}

	return rows, panelDataSummary{Source: "memory", RowCount: len(rows)}, nil
}

// flattenMemoryEntry mirrors the frontend `flattenMemoryEntries` helper.
//
// When `fieldPath` is empty, the memory record is returned verbatim. When
// `fieldPath` resolves to a list, each element becomes its own row. When it
// resolves to a scalar/object, a single-row list with the value at `value`
// is returned for downstream renderers.
func flattenMemoryEntry(values map[string]any, fieldPath string) []map[string]any {
	if fieldPath == "" {
		return []map[string]any{values}
	}

	resolved := resolveFieldPath(values, fieldPath)
	if resolved == nil {
		return []map[string]any{}
	}

	if list, ok := resolved.([]any); ok {
		rows := make([]map[string]any, 0, len(list))
		for _, entry := range list {
			if asMap, ok := entry.(map[string]any); ok {
				rows = append(rows, asMap)
				continue
			}
			rows = append(rows, map[string]any{"value": entry})
		}
		return rows
	}

	if asMap, ok := resolved.(map[string]any); ok {
		return []map[string]any{asMap}
	}

	return []map[string]any{{"value": resolved}}
}

// resolveFieldPath walks a dotted path against a map. Numeric segments are
// treated as list indices so paths like "items.0.name" work without
// requiring users to learn a custom syntax.
func resolveFieldPath(value any, fieldPath string) any {
	current := value
	for _, segment := range strings.Split(fieldPath, ".") {
		segment = strings.TrimSpace(segment)
		if segment == "" {
			continue
		}
		switch typed := current.(type) {
		case map[string]any:
			current = typed[segment]
		case []any:
			idx := -1
			fmt.Sscanf(segment, "%d", &idx)
			if idx < 0 || idx >= len(typed) {
				return nil
			}
			current = typed[idx]
		default:
			return nil
		}
	}
	return current
}

func fetchRunRows(
	ctx core.CommandContext,
	canvasID string,
	dataSource map[string]any,
	overrideLimit int64,
) ([]map[string]any, panelDataSummary, error) {
	limit := resolveLimit(dataSource["limit"], overrideLimit)

	response, _, err := ctx.API.CanvasRunAPI.
		CanvasesListRuns(ctx.Context, canvasID).
		Limit(limit).
		Execute()
	if err != nil {
		return nil, panelDataSummary{}, err
	}

	rows := make([]map[string]any, 0, len(response.GetRuns()))
	for _, run := range response.GetRuns() {
		row, err := structToMap(run)
		if err != nil {
			return nil, panelDataSummary{}, err
		}
		rows = append(rows, row)
		if int64(len(rows)) >= limit {
			break
		}
	}

	totalCount := 0
	if response.HasTotalCount() {
		totalCount = int(response.GetTotalCount())
	}

	return rows, panelDataSummary{Source: "runs", RowCount: len(rows), TotalCount: totalCount}, nil
}

// fetchExecutionRows iterates through the events endpoint, collecting and
// filtering executions until the requested limit is satisfied or
// `maxExecutionsPagesPerCommand` is reached. The page-bound mirrors the
// UI's eager pagination cap (also 20 pages of 25 events each).
func fetchExecutionRows(
	ctx core.CommandContext,
	canvasID string,
	dataSource map[string]any,
	overrideLimit int64,
) ([]map[string]any, panelDataSummary, error) {
	limit := resolveLimit(dataSource["limit"], overrideLimit)
	targetNode, _ := dataSource["node"].(string)
	targetNodeID := targetNode

	if targetNode != "" {
		resolved, err := resolveNodeID(ctx, canvasID, targetNode)
		if err == nil && resolved != "" {
			targetNodeID = resolved
		}
	}

	rows := []map[string]any{}
	var before *time.Time
	for page := 0; page < maxExecutionsPagesPerCommand; page++ {
		req := ctx.API.CanvasEventAPI.
			CanvasesListCanvasEvents(ctx.Context, canvasID).
			Limit(executionsPageSize)
		if before != nil {
			req = req.Before(*before)
		}

		response, _, err := req.Execute()
		if err != nil {
			return nil, panelDataSummary{}, err
		}

		for _, event := range response.GetEvents() {
			for _, exec := range event.GetExecutions() {
				if targetNodeID != "" && exec.GetNodeId() != targetNodeID {
					continue
				}
				row, err := structToMap(exec)
				if err != nil {
					return nil, panelDataSummary{}, err
				}
				row["status"] = deriveExecutionStatus(string(exec.GetState()), string(exec.GetResult()))
				if exec.HasUpdatedAt() && exec.HasCreatedAt() {
					row["durationMs"] = exec.GetUpdatedAt().Sub(exec.GetCreatedAt()).Milliseconds()
				}
				rows = append(rows, row)
				if int64(len(rows)) >= limit {
					return rows, panelDataSummary{Source: "executions", RowCount: len(rows)}, nil
				}
			}
		}

		if !response.GetHasNextPage() {
			break
		}
		last, ok := response.GetLastTimestampOk()
		if !ok || last == nil {
			break
		}
		before = last
	}

	return rows, panelDataSummary{Source: "executions", RowCount: len(rows)}, nil
}

// deriveExecutionStatus mirrors the lowercase status vocabulary the UI uses
// (`passed`/`failed`/`cancelled`/`running`/`pending`/`unknown`) so CLI rows
// match what users see in panels.
func deriveExecutionStatus(state, result string) string {
	switch state {
	case "STATE_PENDING":
		return "pending"
	case "STATE_STARTED":
		return "running"
	case "STATE_FINISHED":
		switch result {
		case "RESULT_PASSED":
			return "passed"
		case "RESULT_FAILED":
			return "failed"
		case "RESULT_CANCELLED":
			return "cancelled"
		}
	}
	return "unknown"
}

// resolveLimit honors the override flag, falling back to the panel-level
// limit, and finally to a sensible default (50) matching the UI when
// neither is set.
func resolveLimit(rawPanelLimit any, override int64) int64 {
	if override > 0 {
		return override
	}
	switch v := rawPanelLimit.(type) {
	case float64:
		if v > 0 {
			return int64(v)
		}
	case int:
		if v > 0 {
			return int64(v)
		}
	case int64:
		if v > 0 {
			return v
		}
	}
	return defaultDataSourceLimit
}

// resolveNodeID accepts either a node id or a node name and returns the id.
// When the node cannot be resolved we return the original input so the
// caller can match by string equality (for canvases that use the user-
// facing name on event records).
func resolveNodeID(ctx core.CommandContext, canvasID, nameOrID string) (string, error) {
	if strings.TrimSpace(nameOrID) == "" {
		return "", nil
	}

	response, _, err := ctx.API.CanvasAPI.CanvasesDescribeCanvas(ctx.Context, canvasID).Execute()
	if err != nil {
		return nameOrID, err
	}
	if response.Canvas == nil || response.Canvas.Spec == nil {
		return nameOrID, nil
	}
	for _, node := range response.Canvas.Spec.GetNodes() {
		if node.GetId() == nameOrID || node.GetName() == nameOrID {
			return node.GetId(), nil
		}
	}
	return nameOrID, nil
}

// structToMap is a small helper that round-trips an OpenAPI model through
// JSON so the CLI can render it with predictable map-shaped output without
// pulling in reflection-heavy dependencies.
func structToMap(value any) (map[string]any, error) {
	bytes, err := json.Marshal(value)
	if err != nil {
		return nil, err
	}
	out := map[string]any{}
	if err := json.Unmarshal(bytes, &out); err != nil {
		return nil, err
	}
	return out, nil
}

func renderPanelDataText(
	stdout io.Writer,
	panel openapi_client.CanvasesDashboardPanel,
	dataSource map[string]any,
	summary panelDataSummary,
	rows []map[string]any,
) error {
	if _, err := fmt.Fprintf(stdout, "Panel:    %s (%s)\n", panel.GetId(), panel.GetType()); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(stdout, "Source:   %s\n", summary.Source); err != nil {
		return err
	}
	if summary.TotalCount > 0 {
		if _, err := fmt.Fprintf(stdout, "Total:    %d\n", summary.TotalCount); err != nil {
			return err
		}
	}
	if _, err := fmt.Fprintf(stdout, "Rows:     %d\n", summary.RowCount); err != nil {
		return err
	}

	if len(rows) == 0 {
		_, err := fmt.Fprintln(stdout, "(no data)")
		return err
	}

	columns := pickPreviewColumns(rows[0])
	if _, err := fmt.Fprintln(stdout); err != nil {
		return err
	}
	writer := tabwriter.NewWriter(stdout, 0, 8, 2, ' ', 0)
	_, _ = fmt.Fprintln(writer, strings.Join(columns, "\t"))
	for _, row := range rows {
		values := make([]string, 0, len(columns))
		for _, column := range columns {
			values = append(values, formatCellValue(row[column]))
		}
		_, _ = fmt.Fprintln(writer, strings.Join(values, "\t"))
	}
	return writer.Flush()
}

// pickPreviewColumns chooses a stable, friendly set of columns for text
// preview output. We bias toward the keys most relevant for status (`id`,
// `state`, `status`, etc.) but fall back to the first few keys so unknown
// payloads still render usefully.
func pickPreviewColumns(row map[string]any) []string {
	preferred := []string{"id", "nodeId", "nodeName", "name", "state", "result", "status", "createdAt", "value"}
	chosen := make([]string, 0, 6)
	seen := make(map[string]struct{})
	for _, key := range preferred {
		if _, ok := row[key]; ok {
			chosen = append(chosen, key)
			seen[key] = struct{}{}
		}
		if len(chosen) >= 6 {
			return chosen
		}
	}
	for key := range row {
		if _, alreadyChosen := seen[key]; alreadyChosen {
			continue
		}
		chosen = append(chosen, key)
		if len(chosen) >= 6 {
			return chosen
		}
	}
	return chosen
}

func formatCellValue(value any) string {
	if value == nil {
		return ""
	}
	switch v := value.(type) {
	case string:
		return v
	case bool:
		return fmt.Sprintf("%t", v)
	case float64:
		if v == float64(int64(v)) {
			return fmt.Sprintf("%d", int64(v))
		}
		return fmt.Sprintf("%g", v)
	default:
		bytes, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(bytes)
	}
}
