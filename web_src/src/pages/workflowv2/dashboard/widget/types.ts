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
  | "code"
  | "link";

export interface WidgetTableColumn {
  field: string;
  label?: string;
  format?: WidgetColumnFormat;
  show?: string;
  href?: string;
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
}

export type WidgetChartKind = "bar" | "stacked-bar" | "line" | "area" | "donut";

export interface WidgetChartSeries {
  field?: string;
  label?: string;
  color?: string;
}

export interface WidgetChartRender {
  kind: "chart";
  type: WidgetChartKind;
  xField: string;
  series: WidgetChartSeries[];
  title?: string;
  limit?: number;
  filters?: string[];
}

export type WidgetNumberAggregation = "count" | "sum" | "avg" | "min" | "max" | "first" | "last";

export interface WidgetNumberRender {
  kind: "number";
  aggregation: WidgetNumberAggregation;
  field?: string;
  filters?: string[];
  format?: WidgetColumnFormat;
  label?: string;
  sparklineField?: string;
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
  const node = typeof action.node === "string" ? action.node : typeof action.target === "string" ? action.target : "";
  return {
    kind: "trigger",
    label: typeof action.label === "string" ? action.label : undefined,
    node,
    hook: typeof action.hook === "string" ? action.hook : undefined,
    template:
      typeof action.template === "string"
        ? action.template
        : typeof action.triggerName === "string"
          ? action.triggerName
          : undefined,
    payload:
      action.payload && typeof action.payload === "object" && !Array.isArray(action.payload)
        ? (action.payload as Record<string, string>)
        : undefined,
    confirm: typeof action.confirm === "string" ? action.confirm : undefined,
    show: typeof action.show === "string" ? action.show : undefined,
    variant:
      typeof action.variant === "string" &&
      WIDGET_ROW_ACTION_VARIANTS.includes(action.variant as WidgetRowActionVariant)
        ? (action.variant as WidgetRowActionVariant)
        : undefined,
    icon:
      typeof action.icon === "string" && WIDGET_ROW_ACTION_ICONS.includes(action.icon as WidgetRowActionIcon)
        ? (action.icon as WidgetRowActionIcon)
        : undefined,
  };
}
