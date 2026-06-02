/**
 * Typed panel content schemas, templates, and validators.
 *
 * Each panel kind owns its own JSON-shape under `panel.content`. Validation is
 * shared between three callers:
 *  - `dashboardYaml.ts` — validates content during YAML import / round-trip
 *  - `useDashboardPanelState` — seeds new panels via `templateForPanelType`
 *  - Per-type form editors — validate the in-memory draft before commit
 *
 * Keep the backend Go validator (`pkg/models/canvas_dashboard_yml.go`) in
 * lockstep with the shapes declared here.
 */

import type {
  WidgetChartRender,
  WidgetNumberAggregation,
  WidgetNumberRender,
  WidgetRowAction,
  WidgetTableColumn,
  WidgetTableFilter,
  WidgetTableRender,
} from "./widget/types";
import { normalizeRowAction, WIDGET_CHART_LEGEND_MODES, WIDGET_FILTER_OPS, WIDGET_SORT_ORDERS } from "./widget/types";
import type { WidgetChartLegendMode, WidgetSort, WidgetSortOrder } from "./widget/types";
import { normalizeWidgetRowStyles, validateWidgetRowStyles } from "./widget/rowStyles";
import { templateForNodesPanel, validateNodesContent } from "./nodesPanelContent";

/** All panel kinds the dashboard currently understands. */
export const PANEL_TYPES = ["markdown", "node", "nodes", "table", "chart", "number"] as const;
export type PanelType = (typeof PANEL_TYPES)[number];

export interface PanelTypeMeta {
  type: PanelType;
  label: string;
  description: string;
}

/**
 * Display-time metadata for each panel kind. Powers the Add Panel picker and
 * the type label rendered in the per-panel editor dialog.
 */
export const PANEL_TYPE_META: Record<PanelType, PanelTypeMeta> = {
  markdown: {
    type: "markdown",
    label: "Markdown",
    description: "Free-form notes, docs, or runbooks rendered as GitHub-flavored markdown.",
  },
  node: {
    type: "node",
    label: "Node",
    description: "A single canvas node with its live status and an optional manual-run button.",
  },
  nodes: {
    type: "nodes",
    label: "Key Nodes",
    description: "Multiple canvas nodes in one card with live status and optional descriptions.",
  },
  table: {
    type: "table",
    label: "Table",
    description: "List rows from canvas executions or memory, with optional row actions.",
  },
  chart: {
    type: "chart",
    label: "Chart",
    description: "Bar, line, area, stacked-bar, or donut chart over execution / memory data.",
  },
  number: {
    type: "number",
    label: "Number",
    description: "A single aggregated KPI value with optional sparkline.",
  },
};

export function isPanelType(value: unknown): value is PanelType {
  return typeof value === "string" && (PANEL_TYPES as readonly string[]).includes(value);
}

// ────────────────────────────────────────────────────────────────────────────
// Per-type content shapes
// ────────────────────────────────────────────────────────────────────────────

export interface MarkdownPanelContent {
  title?: string;
  body?: string;
}

export interface NodePanelContent {
  title?: string;
  /** Canvas node id or name. Required. */
  node: string;
  /** When true and the viewer has run permission, render a manual-run button. */
  showRun?: boolean;
  /** Optional override for the trigger template name (for nodes with multiple triggers). */
  triggerName?: string;
}

export interface TablePanelContent {
  title?: string;
  dataSource: TablePanelDataSource;
  render: WidgetTableRender;
}

export interface ChartPanelContent {
  title?: string;
  dataSource: ChartPanelDataSource;
  render: WidgetChartRender;
}

export interface NumberPanelContent {
  title?: string;
  dataSource: NumberPanelDataSource;
  render: WidgetNumberRender;
}

export type TablePanelDataSource =
  | { kind: "memory"; namespace: string; fieldPath?: string }
  | { kind: "executions"; node?: string; limit?: number }
  | { kind: "runs"; limit?: number };
export type ChartPanelDataSource = TablePanelDataSource;

/** How partial aggregates from a composite memory data source are combined into a single value. */
export type WidgetNumberCombine = "sum" | "min" | "max" | "avg";
export const WIDGET_NUMBER_COMBINE_OPS: WidgetNumberCombine[] = ["sum", "min", "max", "avg"];

/** One namespace contribution inside a composite memory data source. */
export interface MemoryNumberSource {
  namespace: string;
  aggregation: WidgetNumberAggregation;
  field?: string;
  fieldPath?: string;
}

export type CompositeMemoryNumberDataSource = {
  kind: "memory";
  sources: MemoryNumberSource[];
  combine: WidgetNumberCombine;
};

/**
 * Number panels accept the shared table/chart data sources plus a composite
 * memory variant where each namespace carries its own aggregation and field.
 */
export type NumberPanelDataSource = TablePanelDataSource | CompositeMemoryNumberDataSource;

export function isCompositeMemoryDataSource(value: unknown): value is CompositeMemoryNumberDataSource {
  const obj = asObject(value);
  if (!obj) return false;
  return obj.kind === "memory" && Array.isArray(obj.sources);
}

/** True when YAML/config intends composite mode (sources key present), including invalid shapes. */
function hasCompositeMemorySourcesKey(obj: Record<string, unknown>): boolean {
  return obj.kind === "memory" && Object.prototype.hasOwnProperty.call(obj, "sources");
}

// ────────────────────────────────────────────────────────────────────────────
// Templates — used to seed new panels
// ────────────────────────────────────────────────────────────────────────────

const DEFAULT_TABLE_RENDER: WidgetTableRender = {
  kind: "table",
  columns: [],
};

const DEFAULT_CHART_RENDER: WidgetChartRender = {
  kind: "chart",
  type: "bar",
  xField: "status",
  series: [{ label: "Count" }],
};

const DEFAULT_NUMBER_RENDER: WidgetNumberRender = {
  kind: "number",
  aggregation: "count",
  label: "Runs",
};

/**
 * Default content for a newly-added panel of the given kind. The default node
 * reference is left blank; the form editor pre-selects the first canvas node
 * when one is available.
 */
export function templateForPanelType(type: PanelType, defaultTitle?: string): Record<string, unknown> {
  switch (type) {
    case "markdown":
      return { title: defaultTitle ?? "", body: "" } satisfies MarkdownPanelContent;
    case "node":
      return { title: defaultTitle ?? "", node: "", showRun: false } satisfies NodePanelContent;
    case "nodes":
      return { ...templateForNodesPanel(defaultTitle) };
    case "table":
      return {
        title: defaultTitle ?? "",
        dataSource: { kind: "memory", namespace: "" },
        render: DEFAULT_TABLE_RENDER,
      } satisfies TablePanelContent;
    case "chart":
      return {
        title: defaultTitle ?? "",
        dataSource: { kind: "executions", limit: 100 },
        render: DEFAULT_CHART_RENDER,
      } satisfies ChartPanelContent;
    case "number":
      return {
        title: defaultTitle ?? "",
        dataSource: { kind: "runs", limit: 100 },
        render: DEFAULT_NUMBER_RENDER,
      } satisfies NumberPanelContent;
  }
}

// ────────────────────────────────────────────────────────────────────────────
// Validators — returns null when valid, an error message otherwise.
// ────────────────────────────────────────────────────────────────────────────

/**
 * Validate a panel's content given its `type`. Mirrors the per-kind checks in
 * the backend's `ValidateDashboardContent`; the backend remains the source of
 * truth — these checks just give fast UX feedback in the form / YAML editor.
 */
export function validatePanelContent(type: PanelType, content: unknown): string | null {
  switch (type) {
    case "markdown":
      return validateMarkdownContent(content);
    case "node":
      return validateNodeContent(content);
    case "nodes":
      return validateNodesContent(content);
    case "table":
      return validateTableContent(content);
    case "chart":
      return validateChartContent(content);
    case "number":
      return validateNumberContent(content);
  }
}

function asObject(value: unknown): Record<string, unknown> | null {
  if (!value || typeof value !== "object" || Array.isArray(value)) return null;
  return value as Record<string, unknown>;
}

function validateMarkdownContent(content: unknown): string | null {
  if (content === undefined || content === null) return null;
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  if (obj.title !== undefined && obj.title !== null && typeof obj.title !== "string") {
    return "content.title must be a string.";
  }
  if (obj.body !== undefined && obj.body !== null && typeof obj.body !== "string") {
    return "content.body must be a string.";
  }
  return null;
}

function validateNodeContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  // `node` is required as a string but may be empty: newly added panels
  // start unconfigured and the card body renders a "configure me" hint
  // until the user picks one through the form.
  if (typeof obj.node !== "string") {
    return "content.node must be a string (canvas node id or name).";
  }
  if (obj.title !== undefined && obj.title !== null && typeof obj.title !== "string") {
    return "content.title must be a string.";
  }
  if (obj.showRun !== undefined && typeof obj.showRun !== "boolean") {
    return "content.showRun must be a boolean.";
  }
  if (obj.triggerName !== undefined && obj.triggerName !== null && typeof obj.triggerName !== "string") {
    return "content.triggerName must be a string.";
  }
  return null;
}

function validateDataSource(value: unknown): string | null {
  const obj = asObject(value);
  if (!obj) return "dataSource must be an object.";
  if (obj.kind === "memory") return validateMemoryDataSource(obj);
  if (obj.kind === "executions") return validateExecutionsDataSource(obj);
  if (obj.kind === "runs") return validateLimit(obj);
  return 'dataSource.kind must be "memory", "executions", or "runs".';
}

function validateMemoryDataSource(obj: Record<string, unknown>): string | null {
  if (typeof obj.namespace !== "string") {
    return "dataSource.namespace must be a string for memory sources.";
  }
  if (obj.fieldPath != null && typeof obj.fieldPath !== "string") {
    return "dataSource.fieldPath must be a string.";
  }
  return null;
}

/**
 * Number panels accept either the shared data-source shapes (memory with a
 * single namespace, executions, runs) or a composite memory variant where
 * each namespace declares its own aggregation/field and the partials are
 * merged with a configured combine operator.
 */
function validateNumberDataSource(value: unknown): string | null {
  const obj = asObject(value);
  if (!obj) return "dataSource must be an object.";
  if (hasCompositeMemorySourcesKey(obj)) {
    return validateCompositeMemoryDataSource(obj);
  }
  return validateDataSource(value);
}

const ALLOWED_NUMBER_AGGREGATIONS = ["count", "sum", "avg", "min", "max", "first", "last"];

function validateCompositeMemoryDataSource(obj: Record<string, unknown>): string | null {
  if (!Array.isArray(obj.sources)) return "dataSource.sources must be an array.";
  if (obj.sources.length === 0) return "dataSource.sources must be a non-empty array.";
  for (let i = 0; i < obj.sources.length; i += 1) {
    const sourceError = validateMemoryNumberSource(obj.sources[i], i);
    if (sourceError) return sourceError;
  }
  if (typeof obj.combine !== "string" || !WIDGET_NUMBER_COMBINE_OPS.includes(obj.combine as WidgetNumberCombine)) {
    return `dataSource.combine must be one of ${WIDGET_NUMBER_COMBINE_OPS.join(", ")}.`;
  }
  return null;
}

function validateMemoryNumberSource(raw: unknown, index: number): string | null {
  const source = asObject(raw);
  if (!source) return `dataSource.sources[${index}] must be an object.`;
  if (typeof source.namespace !== "string" || source.namespace.trim() === "") {
    return `dataSource.sources[${index}].namespace must be a non-empty string.`;
  }
  if (typeof source.aggregation !== "string" || !ALLOWED_NUMBER_AGGREGATIONS.includes(source.aggregation)) {
    return `dataSource.sources[${index}].aggregation must be one of ${ALLOWED_NUMBER_AGGREGATIONS.join(", ")}.`;
  }
  if (source.aggregation !== "count" && (typeof source.field !== "string" || source.field.trim() === "")) {
    return `dataSource.sources[${index}].field is required when aggregation is "${source.aggregation}".`;
  }
  if (source.fieldPath != null && typeof source.fieldPath !== "string") {
    return `dataSource.sources[${index}].fieldPath must be a string.`;
  }
  return null;
}

function validateLimit(obj: Record<string, unknown>): string | null {
  if (obj.limit != null && (typeof obj.limit !== "number" || !Number.isFinite(obj.limit))) {
    return "dataSource.limit must be a number.";
  }
  return null;
}

function validateExecutionsDataSource(obj: Record<string, unknown>): string | null {
  if (obj.node != null && typeof obj.node !== "string") {
    return "dataSource.node must be a string.";
  }
  return validateLimit(obj);
}

function validateTableContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  const dsError = validateDataSource(obj.dataSource);
  if (dsError) return dsError;
  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  if (render.kind !== "table") return 'render.kind must be "table".';
  return (
    validateTableColumns(render.columns) ??
    validateTableWhere(render.where) ??
    validateSort(render.sort) ??
    validateWidgetRowStyles(render.rowStyles) ??
    validateTableRowActions(render.rowActions)
  );
}

function validateTableWhere(where: unknown): string | null {
  if (where == null) return null;
  if (!Array.isArray(where)) return "render.where must be an array.";
  for (let i = 0; i < where.length; i += 1) {
    const item = asObject(where[i]);
    if (!item) return `render.where[${i}] must be an object.`;
    if (typeof item.field !== "string" || item.field.trim() === "") {
      return `render.where[${i}].field must be a non-empty string.`;
    }
    const op = item.op;
    if (typeof op !== "string" || !WIDGET_FILTER_OPS.includes(op as WidgetTableFilter["op"])) {
      return `render.where[${i}].op is not supported.`;
    }
  }
  return null;
}

export function normalizeTablePanelContent(raw: Record<string, unknown> | undefined): TablePanelContent {
  const r = raw ?? {};
  const renderRaw = (r.render as Record<string, unknown>) ?? {};

  return {
    title: typeof r.title === "string" ? r.title : "",
    dataSource: normalizeTableDataSource(r.dataSource),
    render: {
      kind: "table",
      columns: normalizeTableColumns(renderRaw.columns),
      rowActions: normalizeTableRowActions(renderRaw.rowActions),
      where: normalizeTableWhere(renderRaw.where),
      filters: Array.isArray(renderRaw.filters) ? (renderRaw.filters as string[]) : undefined,
      emptyMessage: typeof renderRaw.emptyMessage === "string" ? renderRaw.emptyMessage : undefined,
      sort: normalizeSort(renderRaw.sort),
      rowStyles: normalizeWidgetRowStyles(renderRaw.rowStyles),
    },
  };
}

/**
 * Coerce the persisted `render.sort` into our typed shape, dropping it
 * entirely when the field is missing/blank so editors that simply clear the
 * field don't leave dangling `{ field: "" }` objects in the YAML output.
 */
function normalizeSort(raw: unknown): WidgetSort | undefined {
  const obj = asObject(raw);
  if (!obj) return undefined;
  const field = typeof obj.field === "string" ? obj.field.trim() : "";
  if (!field) return undefined;
  const order =
    typeof obj.order === "string" && WIDGET_SORT_ORDERS.includes(obj.order as WidgetSortOrder)
      ? (obj.order as WidgetSortOrder)
      : undefined;
  return order ? { field, order } : { field };
}

function normalizeTableColumns(raw: unknown): WidgetTableColumn[] {
  if (!Array.isArray(raw)) return [];

  return raw.map((col) => {
    const c = asObject(col) ?? {};
    return {
      field: typeof c.field === "string" ? c.field : "",
      label: typeof c.label === "string" ? c.label : undefined,
      format: typeof c.format === "string" ? (c.format as WidgetTableColumn["format"]) : undefined,
      show: typeof c.show === "string" ? c.show : undefined,
      href: typeof c.href === "string" ? c.href : undefined,
    };
  });
}

function normalizeTableRowActions(raw: unknown): WidgetRowAction[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  return raw.map(normalizeRowAction).filter((action): action is WidgetRowAction => action != null);
}

function normalizeTableWhere(raw: unknown): WidgetTableFilter[] | undefined {
  if (!Array.isArray(raw)) return undefined;
  return raw.flatMap((filter) => {
    const item = asObject(filter) ?? {};
    const op = typeof item.op === "string" ? item.op : "eq";
    const field = typeof item.field === "string" ? item.field : "";
    if (!field.trim() || !WIDGET_FILTER_OPS.includes(op as WidgetTableFilter["op"])) return [];
    return [{ field, op: op as WidgetTableFilter["op"], value: stringOrUndefined(item.value) }];
  });
}

function normalizeTableDataSource(raw: unknown): TablePanelDataSource {
  const ds = asObject(raw);
  if (ds?.kind === "executions") return normalizeExecutionsDataSource(ds);
  if (ds?.kind === "runs") return { kind: "runs", limit: typeof ds.limit === "number" ? ds.limit : 100 };
  if (ds?.kind === "memory") return normalizeMemoryDataSource(ds);
  return { kind: "memory", namespace: "" };
}

function normalizeExecutionsDataSource(ds: Record<string, unknown>): TablePanelDataSource {
  return {
    kind: "executions",
    node: stringOrUndefined(ds.node),
    limit: typeof ds.limit === "number" ? ds.limit : 50,
  };
}

function normalizeMemoryDataSource(ds: Record<string, unknown>): TablePanelDataSource {
  return {
    kind: "memory",
    namespace: typeof ds.namespace === "string" ? ds.namespace : "",
    fieldPath: stringOrUndefined(ds.fieldPath),
  };
}

function stringOrUndefined(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function validateTableColumns(columns: unknown): string | null {
  if (!Array.isArray(columns)) return "render.columns must be an array.";
  for (let i = 0; i < columns.length; i += 1) {
    const col = asObject(columns[i]);
    if (!col) return `render.columns[${i}] must be an object.`;
    if (typeof col.field !== "string" || col.field.trim() === "") {
      return `render.columns[${i}].field must be a non-empty string.`;
    }
  }
  return null;
}

function validateTableRowActions(rowActions: unknown): string | null {
  if (rowActions == null) return null;
  if (!Array.isArray(rowActions)) return "render.rowActions must be an array.";
  for (let i = 0; i < rowActions.length; i += 1) {
    const action = normalizeRowAction(rowActions[i]);
    if (!action) return `render.rowActions[${i}] must be a trigger action.`;
    if (!action.node.trim()) return `render.rowActions[${i}].node must be set to a trigger node.`;
  }
  return null;
}

function validateChartContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  const dsError = validateDataSource(obj.dataSource);
  if (dsError) return dsError;
  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  return validateChartRender(render);
}

const ALLOWED_CHART_TYPES = ["bar", "stacked-bar", "line", "area", "donut"];

function validateChartRender(render: Record<string, unknown>): string | null {
  if (render.kind !== "chart") return 'render.kind must be "chart".';
  if (typeof render.type !== "string" || !ALLOWED_CHART_TYPES.includes(render.type)) {
    return `render.type must be one of ${ALLOWED_CHART_TYPES.join(", ")}.`;
  }
  if (typeof render.xField !== "string" || render.xField.trim() === "") {
    return "render.xField must be a non-empty string.";
  }
  if (render.seriesField !== undefined && render.seriesField !== null && typeof render.seriesField !== "string") {
    return "render.seriesField must be a string.";
  }
  if (!Array.isArray(render.series) || render.series.length === 0) {
    return "render.series must be a non-empty array.";
  }
  for (let i = 0; i < render.series.length; i += 1) {
    const seriesError = validateChartSeries(render.series[i], i);
    if (seriesError) return seriesError;
  }
  const legendError = validateChartLegend(render.legend);
  if (legendError) return legendError;
  return validateSort(render.sort);
}

function validateChartLegend(legend: unknown): string | null {
  if (legend === undefined) return null;
  if (typeof legend !== "string" || !WIDGET_CHART_LEGEND_MODES.includes(legend as WidgetChartLegendMode)) {
    return `render.legend must be one of ${WIDGET_CHART_LEGEND_MODES.join(", ")}.`;
  }
  return null;
}

function validateChartSeries(raw: unknown, index: number): string | null {
  const series = asObject(raw);
  if (!series) return `render.series[${index}] must be an object.`;
  for (const key of ["field", "label", "color", "format", "prefix", "suffix"] as const) {
    if (series[key] !== undefined && series[key] !== null && typeof series[key] !== "string") {
      return `render.series[${index}].${key} must be a string.`;
    }
  }
  return null;
}

function validateSort(sort: unknown): string | null {
  if (sort === undefined || sort === null) return null;
  const obj = asObject(sort);
  if (!obj) return "render.sort must be an object.";
  if (typeof obj.field !== "string" || obj.field.trim() === "") {
    return "render.sort.field must be a non-empty string.";
  }
  if (obj.order !== undefined && obj.order !== null) {
    if (typeof obj.order !== "string" || !WIDGET_SORT_ORDERS.includes(obj.order as WidgetSortOrder)) {
      return `render.sort.order must be one of ${WIDGET_SORT_ORDERS.join(", ")}.`;
    }
  }
  return null;
}

function validateNumberContent(content: unknown): string | null {
  const obj = asObject(content);
  if (!obj) return "content must be an object.";
  const dsError = validateNumberDataSource(obj.dataSource);
  if (dsError) return dsError;
  const render = asObject(obj.render);
  if (!render) return "render must be an object.";
  if (render.kind !== "number") return 'render.kind must be "number".';
  const symbolError = validateNumberRenderSymbols(render);
  if (symbolError) return symbolError;
  const dataSource = asObject(obj.dataSource);
  if (dataSource && hasCompositeMemorySourcesKey(dataSource)) {
    if (render.aggregation !== undefined) {
      return "render.aggregation must not be set when dataSource.sources is used (each source defines its own aggregation).";
    }
    if (render.field !== undefined) {
      return "render.field must not be set when dataSource.sources is used (each source defines its own field).";
    }
    return null;
  }
  const allowedAggregations = ["count", "sum", "avg", "min", "max", "first", "last"];
  if (typeof render.aggregation !== "string" || !allowedAggregations.includes(render.aggregation)) {
    return `render.aggregation must be one of ${allowedAggregations.join(", ")}.`;
  }
  if (render.aggregation !== "count" && (typeof render.field !== "string" || render.field.trim() === "")) {
    return `render.field is required when aggregation is "${render.aggregation}".`;
  }
  return null;
}

function validateNumberRenderSymbols(render: Record<string, unknown>): string | null {
  for (const key of ["prefix", "suffix"] as const) {
    const value = render[key];
    if (value !== undefined && value !== null && typeof value !== "string") {
      return `render.${key} must be a string.`;
    }
  }
  return null;
}
