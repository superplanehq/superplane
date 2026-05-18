/**
 * Type definitions for Dashboard widget blocks (the ```widget fenced YAML).
 *
 * Widgets pull data from a `dataSource` (canvas memory, executions, or runs) and
 * render it via one of three modes: `table`, `chart`, or `number`. The shape
 * mirrors the guide; see `widgetParser.ts` for the parser/validator.
 */

export type WidgetDataSourceKind = "memory" | "executions" | "runs";

export interface WidgetMemoryDataSource {
  kind: "memory";
  /** Canvas memory namespace to read from (e.g. `deployments`). */
  namespace: string;
  /** Optional field path inside each namespace entry. */
  fieldPath?: string;
}

export interface WidgetExecutionsDataSource {
  kind: "executions";
  /** Optional node reference (id or name). When omitted, all node executions. */
  node?: string;
  /** Maximum number of executions to load. Defaults to 50. */
  limit?: number;
}

export interface WidgetRunsDataSource {
  kind: "runs";
  /** Maximum number of runs to load when a renderer needs row-level data. Defaults to 50. */
  limit?: number;
}

export type WidgetDataSource = WidgetMemoryDataSource | WidgetExecutionsDataSource | WidgetRunsDataSource;

export type WidgetColumnFormat =
  | "text"
  | "number"
  | "percent"
  | "date"
  | "datetime"
  | "duration"
  | "status"
  | "code"
  | "link";

export interface WidgetTableColumn {
  /** Field path of the column value in the row item. */
  field: string;
  /** Column header label. Defaults to the field path. */
  label?: string;
  /** Optional formatter for the cell. */
  format?: WidgetColumnFormat;
  /** Optional `show` expression evaluated per row (e.g. `row.status == 'failed'`). */
  show?: string;
  /** When set, link the cell to the given URL template (supports `{field}` placeholders). */
  href?: string;
}

export type WidgetRowActionKind = "trigger" | "approve" | "cancel" | "push-through";

export interface WidgetRowAction {
  /** Action kind. */
  kind: WidgetRowActionKind;
  /** Optional human label override. */
  label?: string;
  /** Optional `show` expression to gate visibility. */
  show?: string;
  /**
   * For `trigger` actions: the node to trigger; for `approve`/`cancel`/`push-through`:
   * the execution-id field path. Defaults differ per kind (see parser).
   */
  target?: string;
  /** Optional template name for trigger actions. */
  triggerName?: string;
}

export interface WidgetTableRender {
  kind: "table";
  columns: WidgetTableColumn[];
  /** Optional row actions appended at the end of every row. */
  rowActions?: WidgetRowAction[];
  /** Optional filter expressions applied before rendering. */
  filters?: string[];
  /** Empty state message. */
  emptyMessage?: string;
}

export type WidgetChartKind = "bar" | "stacked-bar" | "line" | "area" | "donut";

export interface WidgetChartSeries {
  /** Field path for the y-value (or category count when omitted). */
  field?: string;
  /** Series label. Defaults to `field`. */
  label?: string;
  /** Color hex. */
  color?: string;
}

export interface WidgetChartRender {
  kind: "chart";
  type: WidgetChartKind;
  /** Field path for the x-axis / categories. */
  xField: string;
  /** Series definitions. Multiple are stacked for `stacked-bar`. */
  series: WidgetChartSeries[];
  /** Optional title overlay. */
  title?: string;
  /** Limit categories. */
  limit?: number;
  /** Optional pre-render filters. */
  filters?: string[];
}

export type WidgetNumberAggregation = "count" | "sum" | "avg" | "min" | "max" | "first" | "last";

export interface WidgetNumberRender {
  kind: "number";
  /** Aggregation applied to the input data. */
  aggregation: WidgetNumberAggregation;
  /** Field path required for non-count aggregations. */
  field?: string;
  /** Optional pre-aggregation filters. */
  filters?: string[];
  /** Formatter applied to the result. */
  format?: WidgetColumnFormat;
  /** Optional label rendered above the value. */
  label?: string;
  /** Optional sparkline series field path. */
  sparklineField?: string;
}

export type WidgetRender = WidgetTableRender | WidgetChartRender | WidgetNumberRender;

export interface WidgetConfig {
  /** Optional widget title rendered above the body. */
  title?: string;
  /** Optional `show` expression (top-level visibility). */
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
