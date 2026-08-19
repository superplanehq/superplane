/**
 * Type definitions for dashboard widget renderers (table, chart, number).
 */

import type { RunStatusFilter } from "@/ui/Runs/runStatusFilterVocab";

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
  /**
   * Optional status filter. Empty or omitted means "all statuses". Filtering
   * happens client-side after runs are fetched, so `limit` still bounds
   * what the fetch sees.
   */
  statuses?: RunStatusFilter[];
  /**
   * Optional trigger filter — each entry references a trigger node by id
   * or name. Resolved through the console context at match time so
   * renames don't silently drop rows. Empty or omitted means "all triggers".
   */
  triggers?: string[];
}

export type WidgetDataSource = WidgetMemoryDataSource | WidgetExecutionsDataSource | WidgetRunsDataSource;

/**
 * Uniform result from {@link useWidgetData} so renderers don't care which
 * underlying query produced the rows.
 */
export interface WidgetDataResult {
  rows: unknown[];
  isLoading: boolean;
  error?: string;
  /** Server-reported total for sources that expose one (currently `runs`). */
  totalCount?: number;
  /**
   * Whether more rows can be revealed by calling `loadMore()`. Only meaningful
   * for progressive callers (the table widget). `false` for chart/number
   * panels that always render against the full configured limit.
   */
  hasMore?: boolean;
  /**
   * `true` while a `loadMore()`-triggered (or scroll-triggered) fetch is in
   * flight, distinct from the initial fill which is reported via `isLoading`.
   */
  isFetchingMore?: boolean;
  /**
   * Grow the per-widget display window by `LOAD_MORE_STEP` rows (capped at
   * the configured limit, if any). No-op for non-progressive callers.
   */
  loadMore?: () => void;
  /**
   * Progressive display window size. When set, `rows` is the full loaded set
   * (so filter/sort see every already-fetched row) and the table renders only
   * the first `displayCount` rows after filter+sort. Trend neighbors can then
   * resolve against loaded-but-not-yet-shown rows.
   */
  displayCount?: number;
}

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
 * Direction that signals a "better" trend for trend columns / `showTrend`.
 * `up` (default) → an increase is good (green), a decrease is bad (red).
 * `down` → an increase is bad (red), a decrease is good (green).
 */
export type WidgetTrendBetter = "up" | "down";
export const WIDGET_TREND_BETTER: WidgetTrendBetter[] = ["up", "down"];

/**
 * How a trend chip prints its magnitude alongside the arrow.
 * `percent` (default) → signed percent change vs. the row below.
 * `value` → signed absolute delta vs. the row below.
 * `none` → arrow only (still shows `- 0` / `...` / `-` for edge states).
 */
export type WidgetTrendDisplay = "percent" | "value" | "none";
export const WIDGET_TREND_DISPLAYS: WidgetTrendDisplay[] = ["percent", "value", "none"];

/** Formats that can show a value + trend chip via `showTrend`. */
export const WIDGET_SHOW_TREND_FORMATS: WidgetColumnFormat[] = ["number", "percent", "duration"];

export function columnSupportsShowTrend(format: WidgetColumnFormat | undefined): boolean {
  return format != null && WIDGET_SHOW_TREND_FORMATS.includes(format);
}

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
  /**
   * When true on `number` | `percent` | `duration`, render the formatted value
   * plus a trend chip (same semantics as `format: trend`). Ignored otherwise.
   */
  showTrend?: boolean;
  /** Which direction is "better" for trend / `showTrend`. Defaults to `up`. */
  trendBetter?: WidgetTrendBetter;
  /** What to show next to the trend arrow. Defaults to `percent`. */
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

/**
 * How the scorecard change chip prints its magnitude. `both` mirrors the
 * screenshot pattern `-29 (-22.8%)`; `none` hides text and keeps just the
 * directional arrow.
 */
export type WidgetScorecardShowChange = "percent" | "number" | "both" | "none";
export const WIDGET_SCORECARD_SHOW_CHANGES: WidgetScorecardShowChange[] = ["percent", "number", "both", "none"];

/**
 * Scorecard render: single KPI value plus optional change vs the
 * immediately-previous value in the series, direction-aware
 * target/progress, and a status-colored sparkline. Composite memory and
 * multi-KPI shapes are intentionally not supported — the dedicated
 * `number` panel already covers those cases.
 */
export interface WidgetScorecardRender {
  kind: "scorecard";
  /** Same vocabulary as {@link WidgetNumberRender}. Required. */
  aggregation: WidgetNumberAggregation;
  /** Required when aggregation is not "count". */
  field?: string;
  /** Legacy show expressions applied to filter rows before aggregation / series extraction. */
  filters?: string[];
  format?: WidgetColumnFormat;
  label?: string;
  /** Optional display string rendered before the formatted value (e.g. "R$"). */
  prefix?: string;
  /** Optional display string rendered after the formatted value (e.g. " MWh"). */
  suffix?: string;
  /**
   * Direction that signals "better". Colors the change chip, the sparkline,
   * and the vs-target status. Defaults to `up`.
   */
  better?: WidgetTrendBetter;
  /**
   * Target value used for optional progress and (when the change is
   * incomputable) status coloring. Accepts a numeric literal (`"50"`,
   * `"100.5"`) or a full `{{ CEL }}` expression evaluated against the last
   * filtered row plus `now`.
   */
  target?: string;
  /** When true and the target resolves, render a direction-aware progress bar. */
  showProgress?: boolean;
  /**
   * Field extracted from each filtered row (in order) to draw the
   * sparkline. When omitted the sparkline is hidden — the change chip
   * still renders using the primary `field` as its fallback series.
   */
  sparklineField?: string;
  /** What the change chip prints alongside its arrow. Defaults to `both`. */
  showChange?: WidgetScorecardShowChange;
  /** Optional short caption rendered next to the change chip (e.g. "vs previous"). */
  changeCaption?: string;
}

/**
 * Palette accepted by `WidgetBoardLane.color`. Kept intentionally small
 * (neutral + status-family tones); YAML stays stable across future
 * Tailwind refactors thanks to `BOARD_LANE_STYLE` in `widget/boardLaneStyles.ts`.
 */
export type WidgetBoardLaneColor = "neutral" | "gray" | "blue" | "green" | "yellow" | "orange" | "red" | "purple";

export const WIDGET_BOARD_LANE_COLORS: WidgetBoardLaneColor[] = [
  "neutral",
  "gray",
  "blue",
  "green",
  "yellow",
  "orange",
  "red",
  "purple",
];

/** One lane in a `WidgetBoardRender.lanes` list. */
export interface WidgetBoardLane {
  /** Value to match against `groupBy` (case-insensitive, trimmed). Required. */
  value: string;
  /** Optional lane header label; defaults to `value`. */
  label?: string;
  /** Optional lane color from the {@link WidgetBoardLaneColor} palette. */
  color?: WidgetBoardLaneColor;
}

/** Card configuration for a board panel. */
export interface WidgetBoardCard {
  /** Row field to display as the card title. Required. */
  titleField: string;
  /**
   * Optional additional card fields. Each entry reuses {@link WidgetTableColumn}
   * semantics — `field`, optional `label`, `format`, `show`, and `href`.
   */
  fields?: WidgetTableColumn[];
}

export interface WidgetBoardRender {
  kind: "board";
  /** Required row field name used to group rows into lanes. */
  groupBy: string;
  /** Required list of lanes; at least one entry. */
  lanes: WidgetBoardLane[];
  /** Card display config. */
  card: WidgetBoardCard;
  /**
   * When true, rows whose `groupBy` value does not match any configured
   * lane render in a trailing "Other" lane instead of being hidden.
   */
  otherLane?: boolean;
  /** Same structured filters as the table widget. */
  where?: WidgetTableFilter[];
  /** Optional widget-level sort applied to rows inside each lane. */
  sort?: WidgetSort;
  /** Optional row actions (trigger-only, same rules as the table widget). */
  rowActions?: WidgetRowAction[];
  /** Optional label rendered when no rows match the current filters. */
  emptyMessage?: string;
}

export type WidgetRender =
  | WidgetTableRender
  | WidgetChartRender
  | WidgetNumberRender
  | WidgetScorecardRender
  | WidgetBoardRender;

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
