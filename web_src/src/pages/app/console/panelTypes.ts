/**
 * Typed panel content schemas, templates, and validators.
 * Keep the backend Go validator (`pkg/models/console_yml.go`) in lockstep.
 */

import type { RunStatusFilter } from "@/ui/Runs/runStatusFilterVocab";
import type {
  WidgetChartRender,
  WidgetNumberAggregation,
  WidgetNumberRender,
  WidgetRowAction,
  WidgetScorecardRender,
  WidgetTableColumn,
  WidgetTableFilter,
  WidgetTableRender,
} from "./widget/types";
import {
  normalizeRowAction,
  WIDGET_FILTER_OPS,
  WIDGET_PROGRESS_LABELS,
  WIDGET_SORT_ORDERS,
  WIDGET_TREND_BETTER,
  WIDGET_TREND_DISPLAYS,
} from "./widget/types";
import type { WidgetProgressLabel, WidgetSort, WidgetSortOrder } from "./widget/types";
import { validateChartRender } from "./chartRenderValidation";
import { normalizeWidgetRowStyles, validateWidgetRowStyles } from "./widget/rowStyles";
import { templateForNodesPanel, validateNodesContent } from "./nodesPanelContent";
import { validateNumberContent } from "./numberContentValidation";
import { validateMarkdownContent, type MarkdownVariable } from "./markdownVariables";
import { asObject, optionalBooleanError, optionalStringError } from "./panelContentValidation";
import {
  normalizeTableDataSource,
  validateRunStatusesArray,
  validateRunTriggersArray,
} from "./runDataSourceFilterSchema";
import { validateScorecardContent } from "./scorecardRenderValidation";

// Re-export markdown-variable types so existing import paths keep working.
export * from "./markdownVariables";

// Re-export runs filter schema helpers so existing import paths keep working.
export {
  normalizeRunStatuses,
  normalizeRunTriggers,
  normalizeRunsDataSource,
  validateRunStatusesArray,
  validateRunTriggersArray,
} from "./runDataSourceFilterSchema";

// Re-export the shared object narrow so downstream validators
// (e.g. `chartRenderValidation.ts`) keep their existing import path.
export { asObject };

/** All panel kinds the dashboard currently understands. */
export const PANEL_TYPES = ["markdown", "html", "node", "nodes", "table", "chart", "number", "scorecard"] as const;
export type PanelType = (typeof PANEL_TYPES)[number];

/**
 * Panel types offered in the Add Panel picker. `node` is intentionally
 * excluded — the merged {@link NodesPanelCard} renders both legacy `node`
 * and modern `nodes` panels, so authors always start from the plural
 * shape. The legacy `node` panel type remains in {@link PANEL_TYPES} for
 * validation and YAML import compatibility.
 */
export const CREATABLE_PANEL_TYPES = PANEL_TYPES.filter((t) => t !== "node") as readonly Exclude<PanelType, "node">[];

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
  html: {
    type: "html",
    label: "HTML",
    description:
      "Custom HTML with inline styles, scoped <style>, and Tailwind classes. Scripts and external resources are blocked.",
  },
  node: {
    type: "node",
    label: "Node",
    description: "A single canvas node with its live status and an optional manual-run button.",
  },
  nodes: {
    type: "nodes",
    label: "Nodes",
    description: "One or more canvas nodes with live status and optional manual-run buttons.",
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
  scorecard: {
    type: "scorecard",
    label: "Scorecard",
    description: "A KPI with target, change vs the previous value, and a status-colored sparkline.",
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
  /** Named variables referenced from the markdown body via `{{ name.field }}`. */
  variables?: MarkdownVariable[];
}

/**
 * Content shape for the `html` panel. Structurally identical to
 * {@link MarkdownPanelContent} - title, raw body, and the shared variable
 * system - but the body is HTML rendered through the strict sanitizer in
 * `htmlSanitize.ts` instead of the markdown pipeline.
 */
export type HtmlPanelContent = MarkdownPanelContent;

export interface NodePanelContent {
  title?: string;
  /** Canvas node id or name. Required. */
  node: string;
  /** Optional override for the displayed node name. Falls back to the resolved canvas node name. */
  label?: string;
  /** When true and the viewer has run permission, render a manual-run button. */
  showRun?: boolean;
  /** Optional override for the trigger template name (for nodes with multiple triggers). */
  triggerName?: string;
  /**
   * When true, clicking Run always opens the confirm dialog — even for
   * templates with no input fields. When false (default), a parameter-less
   * template fires immediately; templates with input fields always prompt.
   */
  promptConfirmation?: boolean;
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
  /** Used by single/composite modes. Absent in multi-number mode. */
  dataSource?: NumberPanelDataSource;
  /** Used by single/composite modes. Absent in multi-number mode. */
  render?: WidgetNumberRender;
  /** Present (and an array) when the panel is in multi-number mode. */
  metrics?: NumberMetric[];
}

/**
 * Content shape for the `scorecard` panel. Single-KPI only — use `number`
 * for composite memory / multi-KPI.
 */
export interface ScorecardPanelContent {
  title?: string;
  dataSource: TablePanelDataSource;
  render: WidgetScorecardRender;
}

/** One number inside a multi-number panel (own data source + aggregation). */
export interface NumberMetric {
  dataSource: TablePanelDataSource;
  render: WidgetNumberRender;
}

export type TablePanelDataSource =
  | { kind: "memory"; namespace: string; fieldPath?: string }
  | { kind: "executions"; node?: string; limit?: number }
  | {
      kind: "runs";
      limit?: number;
      /** See {@link WidgetRunsDataSource.statuses}. */
      statuses?: RunStatusFilter[];
      /** See {@link WidgetRunsDataSource.triggers}. */
      triggers?: string[];
    };
export type ChartPanelDataSource = TablePanelDataSource;

/** How partial aggregates from a composite memory data source are combined into a single value. */
export type WidgetNumberCombine = "sum" | "min" | "max" | "avg";
export const WIDGET_NUMBER_COMBINE_OPS: WidgetNumberCombine[] = ["sum", "min", "max", "avg"];

/**
 * Aggregations accepted by number / scorecard renders. Shared by validators
 * and form controls so the allowed set cannot drift.
 */
export const WIDGET_NUMBER_AGGREGATIONS: WidgetNumberAggregation[] = [
  "count",
  "sum",
  "avg",
  "min",
  "max",
  "first",
  "last",
];

/** Type guard for the shared number-aggregation set above. */
export function isAllowedNumberAggregation(value: unknown): value is WidgetNumberAggregation {
  return typeof value === "string" && (WIDGET_NUMBER_AGGREGATIONS as string[]).includes(value);
}

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
export function hasCompositeMemorySourcesKey(obj: Record<string, unknown>): boolean {
  return obj.kind === "memory" && Object.prototype.hasOwnProperty.call(obj, "sources");
}

/**
 * True when the panel content is in multi-number mode (a `metrics` array is
 * present, even if shaped invalidly). Distinct from the composite-combine
 * mode, which carries `dataSource.sources` instead.
 */
export function isMultiNumberContent(content: unknown): boolean {
  const obj = asObject(content);
  if (!obj) return false;
  return Array.isArray(obj.metrics);
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

// `count` needs no field, so a fresh scorecard validates before the author
// picks a data source or switches to a field-backed aggregation.
const DEFAULT_SCORECARD_RENDER: WidgetScorecardRender = {
  kind: "scorecard",
  aggregation: "count",
  better: "up",
  showChange: "both",
  changeCaption: "vs previous",
};

/** Default content for a newly-added panel of the given kind. */
export function templateForPanelType(type: PanelType, defaultTitle?: string): Record<string, unknown> {
  switch (type) {
    case "markdown":
    case "html":
      return { title: defaultTitle ?? "", body: "", variables: [] } satisfies MarkdownPanelContent;
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
    case "scorecard":
      return {
        title: defaultTitle ?? "",
        dataSource: { kind: "memory", namespace: "" },
        render: DEFAULT_SCORECARD_RENDER,
      } satisfies ScorecardPanelContent;
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
    case "html":
      // Both kinds carry the same `title?` + `body?` + variables shape; only
      // the renderer differs (markdown pipeline vs. HTML sanitizer).
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
    case "scorecard":
      return validateScorecardContent(content);
  }
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
  return (
    optionalStringError("content.title", obj.title) ??
    optionalStringError("content.label", obj.label) ??
    optionalBooleanError("content.showRun", obj.showRun) ??
    optionalStringError("content.triggerName", obj.triggerName) ??
    optionalBooleanError("content.promptConfirmation", obj.promptConfirmation)
  );
}

export function validateDataSource(value: unknown): string | null {
  const obj = asObject(value);
  if (!obj) return "dataSource must be an object.";
  if (obj.kind === "memory") return validateMemoryDataSource(obj);
  if (obj.kind === "executions") return validateExecutionsDataSource(obj);
  if (obj.kind === "runs") return validateRunsDataSource(obj);
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

function validateLimit(obj: Record<string, unknown>): string | null {
  if (obj.limit != null && (typeof obj.limit !== "number" || !Number.isFinite(obj.limit))) {
    return "dataSource.limit must be a number.";
  }
  return null;
}

function validateRunsDataSource(obj: Record<string, unknown>): string | null {
  const limitError = validateLimit(obj);
  if (limitError) return limitError;
  const statusesError = validateRunStatusesArray(obj.statuses, "dataSource.statuses");
  if (statusesError) return statusesError;
  return validateRunTriggersArray(obj.triggers, "dataSource.triggers");
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
      avatarCommitterField: typeof c.avatarCommitterField === "string" ? c.avatarCommitterField : undefined,
      progressTarget: typeof c.progressTarget === "string" ? c.progressTarget : undefined,
      progressLabel: optionalEnum(c.progressLabel, WIDGET_PROGRESS_LABELS),
      showTrend: c.showTrend === true ? true : undefined,
      trendBetter: optionalEnum(c.trendBetter, WIDGET_TREND_BETTER),
      trendDisplay: optionalEnum(c.trendDisplay, WIDGET_TREND_DISPLAYS),
    };
  });
}

function optionalEnum<T extends string>(value: unknown, allowed: readonly T[]): T | undefined {
  return typeof value === "string" && allowed.includes(value as T) ? (value as T) : undefined;
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
    const value = typeof item.value === "string" ? item.value : undefined;
    return [{ field, op: op as WidgetTableFilter["op"], value }];
  });
}

function validateTableColumns(columns: unknown): string | null {
  if (!Array.isArray(columns)) return "render.columns must be an array.";
  for (let i = 0; i < columns.length; i += 1) {
    const col = asObject(columns[i]);
    if (!col) return `render.columns[${i}] must be an object.`;
    if (typeof col.field !== "string" || col.field.trim() === "") {
      return `render.columns[${i}].field must be a non-empty string.`;
    }
    const progressError = validateProgressColumnFields(i, col);
    if (progressError) return progressError;
  }
  return null;
}

function validateProgressColumnFields(index: number, col: Record<string, unknown>): string | null {
  if (col.progressLabel !== undefined && col.progressLabel !== null) {
    if (
      typeof col.progressLabel !== "string" ||
      !WIDGET_PROGRESS_LABELS.includes(col.progressLabel as WidgetProgressLabel)
    ) {
      return `render.columns[${index}].progressLabel must be one of ${WIDGET_PROGRESS_LABELS.join(", ")}.`;
    }
  }
  if (col.format === "progress") {
    if (typeof col.progressTarget !== "string" || col.progressTarget.trim() === "") {
      return `render.columns[${index}].progressTarget must be a non-empty string for progress columns.`;
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

export function validateSort(sort: unknown): string | null {
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

export function validateNumberRenderSymbols(render: Record<string, unknown>): string | null {
  for (const key of ["prefix", "suffix"] as const) {
    const value = render[key];
    if (value !== undefined && value !== null && typeof value !== "string") {
      return `render.${key} must be a string.`;
    }
  }
  return null;
}
