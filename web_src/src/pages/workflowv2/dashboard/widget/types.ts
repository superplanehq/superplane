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
