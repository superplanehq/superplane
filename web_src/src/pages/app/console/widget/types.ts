/**
 * Type definitions for dashboard widget renderers (table, chart, number).
 */

export type WidgetDataSourceKind = "memory" | "executions" | "runs";

export interface WidgetMemoryDataSource {
  kind: "memory";
  namespace: string;
  fieldPath?: string;
}

export interface WidgetExecutionsDataSource {
  kind: "executions";
  node?: string;
  limit?: number;
}

export interface WidgetRunsDataSource {
  kind: "runs";
  limit?: number;
}

export type WidgetDataSource = WidgetMemoryDataSource | WidgetExecutionsDataSource | WidgetRunsDataSource;

export type WidgetColumnFormat =
  | "text"
  | "number"
  | "percent"
  | "date"
  | "datetime"
  | "relative"
  | "duration"
  | "status"
  | "badge"
  | "code"
  | "link"
  | "avatar"
  | "progress"
  | "trend";

/** Text label rendered next to the bar for `format: progress`. */
export type WidgetProgressLabel = "none" | "number" | "percent";

export const WIDGET_PROGRESS_LABELS: WidgetProgressLabel[] = ["none", "number", "percent"];

/**
 * Direction that signals a "better" trend for `format: trend` columns.
 * `up` (default) → an increase is good (green), a decrease is bad (red).
 * `down` → an increase is bad (red), a decrease is good (green).
 */
export type WidgetTrendBetter = "up" | "down";
export const WIDGET_TREND_BETTER: WidgetTrendBetter[] = ["up", "down"];

/**
 * How a trend cell prints its magnitude alongside the arrow.
 * `percent` (default) → signed percent change vs. the row below.
 * `value` → signed absolute delta vs. the row below.
 * `none` → arrow only (still shows `- 0` / `...` / `-` for edge states).
 */
export type WidgetTrendDisplay = "percent" | "value" | "none";
export const WIDGET_TREND_DISPLAYS: WidgetTrendDisplay[] = ["percent", "value", "none"];

export interface WidgetTableColumn {
  field: string;
  label?: string;
  format?: WidgetColumnFormat;
  show?: string;
  href?: string;
  /** Secondary person map used for avatar initials when `format: avatar`. */
  avatarCommitterField?: string;
  /**
   * Target (100%) reference for `format: progress`. Accepts a numeric literal,
   * a dot path against the row, or a full `{{ CEL }}` expression — resolved
   * with the same helper as `field` at render time.
   */
  progressTarget?: string;
  /** Label style rendered next to the progress bar. Defaults to `"percent"`. */
  progressLabel?: WidgetProgressLabel;
  /** For `format: trend`: which direction is "better". Defaults to `up`. */
  trendBetter?: WidgetTrendBetter;
  /** For `format: trend`: what to show next to the arrow. Defaults to `percent`. */
  trendDisplay?: WidgetTrendDisplay;
}

export type WidgetFilterOp = "eq" | "neq" | "contains" | "not_contains" | "gt" | "lt" | "exists" | "not_exists";

export interface WidgetTableFilter {
  field: string;
  op: WidgetFilterOp;
  value?: string;
}

export type WidgetRowActionVariant = "default" | "primary" | "danger";
export type WidgetRowActionIcon = "play" | "stop" | "trash" | "refresh" | "external-link";

export type WidgetRowActionKind = "trigger";

export interface WidgetTriggerRowAction {
  kind: "trigger";
  label?: string;
  /** Canvas trigger node id or name. */
  node: string;
  /** Trigger hook name (default `run`). */
  hook?: string;
  /** Start template name when applicable. */
  template?: string;
  /** Dot-path → template string merged into the hook payload. */
  payload?: Record<string, string>;
  confirm?: string;
  show?: string;
  variant?: WidgetRowActionVariant;
  icon?: WidgetRowActionIcon;
  /** @deprecated Use `node` — legacy row-action field. */
  target?: string;
  /** @deprecated Use `template` — legacy alias. */
  triggerName?: string;
}

export type WidgetRowAction = WidgetTriggerRowAction;

export interface WidgetTableRender {
  kind: "table";
  columns: WidgetTableColumn[];
  rowActions?: WidgetRowAction[];
  /** Structured filters (ANDed). Preferred for tables. */
  where?: WidgetTableFilter[];
  /** Legacy string filters — still supported for backwards compatibility. */
  filters?: string[];
  emptyMessage?: string;
  /** Optional widget-level row sort applied after filters. */
  sort?: WidgetSort;
  /**
   * Optional per-row background tints. Each rule is one field/op/value
   * condition (same shape and semantics as `where[i]`) mapped to a tone
   * from the curated palette. Evaluated first-match-wins per row.
   */
  rowStyles?: WidgetRowStyle[];
}

/**
 * Curated palette of Tailwind background tones authors can pick from. The
 * tone enum is the persisted value; the actual class is resolved through
 * `ROW_STYLE_CLASS` in `widget/rowStyles.ts` so the YAML stays stable even
 * if we later swap the underlying utility classes.
 */
export type WidgetRowStyleTone =
  | "dimmed"
  | "yellow"
  | "yellow-soft"
  | "orange"
  | "orange-soft"
  | "red"
  | "red-soft"
  | "blue"
  | "blue-soft"
  | "green"
  | "green-soft";

export const WIDGET_ROW_STYLE_TONES: WidgetRowStyleTone[] = [
  "dimmed",
  "yellow-soft",
  "yellow",
  "orange-soft",
  "orange",
  "red-soft",
  "red",
  "blue-soft",
  "blue",
  "green-soft",
  "green",
];

/** One row-background rule. Same condition shape as `WidgetTableFilter`. */
export interface WidgetRowStyle {
  field: string;
  op: WidgetFilterOp;
  value?: string;
  tone: WidgetRowStyleTone;
}

export type WidgetChartKind = "bar" | "stacked-bar" | "line" | "area" | "donut";

export type WidgetChartLegendMode = "auto" | "show" | "hide";
export const WIDGET_CHART_LEGEND_MODES: WidgetChartLegendMode[] = ["auto", "show", "hide"];

export interface WidgetChartSeries {
  field?: string;
  label?: string;
  color?: string;
  /** Optional value format applied in tooltips (and in donut value rows). */
  format?: WidgetColumnFormat;
  /** Optional display string prepended to the formatted value (e.g. "$"). */
  prefix?: string;
  /** Optional display string appended to the formatted value (e.g. " MWh"). */
  suffix?: string;
}

export interface WidgetChartRender {
  kind: "chart";
  type: WidgetChartKind;
  xField: string;
  series: WidgetChartSeries[];
  title?: string;
  limit?: number;
  filters?: string[];
  /** Legend visibility. Defaults to "auto" — visible for donut charts or when 2+ series exist. */
  legend?: WidgetChartLegendMode;
  /** Optional widget-level row sort applied after filters, before chart binning. */
  sort?: WidgetSort;
  /**
   * Optional field whose distinct values pivot long-format rows into one
   * series per value (e.g. one stack segment per service). When set, the
   * chart uses the numeric `field` of the first configured series for
   * values and ignores additional series entries.
   */
  seriesField?: string;
  /**
   * Optional display format applied to X-axis tick labels. Reuses the
   * shared `WidgetColumnFormat` vocabulary so date / duration / number
   * X-axes don't require a CEL wrapper around `xField`.
   */
  xFormat?: WidgetColumnFormat;
  /**
   * Optional Y-axis title rendered alongside the axis ticks (e.g. "USD",
   * "Errors / day"). When omitted no axis label is drawn.
   */
  yLabel?: string;
  /**
   * Optional display format applied to Y-axis tick labels. When omitted
   * the renderer falls back to its locale-aware numeric default.
   */
  yFormat?: WidgetColumnFormat;
}

/** Sort direction. Defaults to `"asc"` when omitted on a `WidgetSort`. */
export type WidgetSortOrder = "asc" | "desc";
export const WIDGET_SORT_ORDERS: WidgetSortOrder[] = ["asc", "desc"];

/**
 * Widget-level row sort. `field` accepts the same surface as chart/table
 * fields: a literal dot path (e.g. `createdAt`) or a full `{{ cel }}`
 * expression (e.g. `{{ formatDate(createdAt, "yyyy-MM-dd") }}`).
 *
 * Null / undefined values always sort to the end so empty rows don't poison
 * the visible ordering regardless of direction.
 */
export interface WidgetSort {
  field: string;
  order?: WidgetSortOrder;
}

export type WidgetNumberAggregation = "count" | "sum" | "avg" | "min" | "max" | "first" | "last";

export interface WidgetNumberRender {
  kind: "number";
  /** Required for simple data sources. Composite memory sources carry their own per-source aggregation. */
  aggregation?: WidgetNumberAggregation;
  field?: string;
  filters?: string[];
  format?: WidgetColumnFormat;
  label?: string;
  sparklineField?: string;
  /** Optional display string rendered before the formatted value (e.g. "R$"). */
  prefix?: string;
  /** Optional display string rendered after the formatted value (e.g. " MWh"). */
  suffix?: string;
}

export type WidgetRender = WidgetTableRender | WidgetChartRender | WidgetNumberRender;

export interface WidgetConfig {
  title?: string;
  show?: string;
  dataSource: WidgetDataSource;
  render: WidgetRender;
}

export interface WidgetParseSuccess {
  ok: true;
  widget: WidgetConfig;
}

export interface WidgetParseFailure {
  ok: false;
  error: string;
}

export type WidgetParseResult = WidgetParseSuccess | WidgetParseFailure;

export const WIDGET_FILTER_OPS: WidgetFilterOp[] = [
  "eq",
  "neq",
  "contains",
  "not_contains",
  "gt",
  "lt",
  "exists",
  "not_exists",
];

export const WIDGET_ROW_ACTION_VARIANTS: WidgetRowActionVariant[] = ["default", "primary", "danger"];
export const WIDGET_ROW_ACTION_ICONS: WidgetRowActionIcon[] = ["play", "stop", "trash", "refresh", "external-link"];

/** Normalize persisted row actions (including legacy `target` / `triggerName`). */
export function normalizeRowAction(raw: unknown): WidgetRowAction | null {
  if (!raw || typeof raw !== "object") return null;
  const action = raw as Record<string, unknown>;
  if (action.kind !== "trigger") return null;

  return {
    kind: "trigger",
    label: stringOrUndefined(action.label),
    node: readRowActionNode(action),
    hook: stringOrUndefined(action.hook),
    template: readRowActionTemplate(action),
    payload: readRowActionPayload(action.payload),
    confirm: stringOrUndefined(action.confirm),
    show: stringOrUndefined(action.show),
    variant: knownValue(action.variant, WIDGET_ROW_ACTION_VARIANTS),
    icon: knownValue(action.icon, WIDGET_ROW_ACTION_ICONS),
  };
}

function readRowActionNode(action: Record<string, unknown>): string {
  return stringOrUndefined(action.node) ?? stringOrUndefined(action.target) ?? "";
}

function readRowActionTemplate(action: Record<string, unknown>): string | undefined {
  return stringOrUndefined(action.template) ?? stringOrUndefined(action.triggerName);
}

function readRowActionPayload(raw: unknown): Record<string, string> | undefined {
  if (!raw || typeof raw !== "object" || Array.isArray(raw)) return undefined;
  return raw as Record<string, string>;
}

function stringOrUndefined(value: unknown): string | undefined {
  return typeof value === "string" ? value : undefined;
}

function knownValue<T extends string>(value: unknown, allowed: readonly T[]): T | undefined {
  return typeof value === "string" && allowed.includes(value as T) ? (value as T) : undefined;
}
