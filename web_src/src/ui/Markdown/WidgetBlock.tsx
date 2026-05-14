import { useContext, useId, useMemo, useState, type ComponentProps } from "react";
import * as yaml from "js-yaml";
import {
  AlertTriangle,
  ArrowRight,
  Check,
  ExternalLink,
  Loader2,
  Play,
  RefreshCw,
  RotateCcw,
  Square,
  Trash2,
  X,
} from "lucide-react";

import { useCanvasMemoryEntries, useInfiniteCanvasEvents, type CanvasMemoryEntry } from "@/hooks/useCanvasData";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeExecutionRef } from "@/api-client";
import { getAggregateStatus } from "@/pages/workflowv2/lib/canvas-runs";
import { formatRelativeTime } from "@/lib/timezone";
import { showSuccessToast } from "@/lib/toast";
import { cn } from "@/lib/utils";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
import { Button } from "@/components/ui/button";
import type { NodeChipContext } from "./CanvasMarkdown";
import { AppsPanelMarkdownLinksContext } from "./appsPanelMarkdownLinksContext";
import {
  buildEnv,
  compileMaybeExpr,
  compileTemplate,
  evalRowField as evalRowFieldExpr,
  evalTemplate as evalTemplateExpr,
  type CompiledTemplate,
  type ExprEnv,
  type MaybeExpr,
} from "./widgetExpr";
import {
  Area,
  AreaChart,
  Bar,
  BarChart,
  CartesianGrid,
  Cell,
  Line,
  LineChart,
  Pie,
  PieChart,
  XAxis,
  YAxis,
} from "recharts";
import { ChartContainer, ChartTooltip, ChartTooltipContent, type ChartConfig } from "@/components/ui/chart";

//
// Authored as a fenced ```widget code block (or the deprecated ```query alias)
// inside Apps markdown panels. Two data sources are supported:
//
//   ```widget
//   source: memory
//   namespace: environments
//   columns:                  # optional — auto-detected from values keys when omitted
//     - label: PR
//       field: pr_number
//   where:                    # optional — all conditions ANDed
//     - field: url
//       op: exists
//   actions:                  # optional — kind: trigger | approve | cancel | push-through
//                             # optional `icon:` when label keywords don't apply; defaults use kind/variant too
//                             # (label tokens approve / reject|deny / rollback / reapply pick icons first)
//     - label: Open
//       kind: trigger
//       trigger: my-trigger
//   render:                   # optional — table | chart | number; defaults to table
//     kind: chart
//     chart:
//       type: bar
//       x: name
//       y: duration
//       label: Latencies           # optional; label containing "duration" + large Y infers ms to seconds
//       y_unit: ms                  # optional explicit ms column → axis/tooltips use compact seconds
//   ```
//
//   ```widget
//   source: executions
//   trigger: deploy-cmd       # optional, filters to runs started by this trigger
//   status: running           # optional: running | passed | failed | cancelled
//   limit: 10                 # default 10, capped at 100
//   columns:
//     - label: PR
//       field: root.data.data.issue.number
//     - label: Status
//       field: status
//       format: badge
//   ```
//
// Render modes:
//   - table  (default): row-per-entry table with optional Actions column.
//   - chart : Mermaid xychart-beta source generated from rows.
//   - number: single big aggregate stat (avg / sum / min / max / count) over a field.
//
// Live updates flow through the existing canvas WebSocket: every relevant
// socket event invalidates `canvasKeys.infiniteEvents(canvasId)`, which makes
// `useInfiniteCanvasEvents` re-fetch and re-render the widget automatically.
//

export interface WidgetBlockProps {
  body: string;
  canvasId: string;
  /** Forwarded by CanvasMarkdown so row actions can resolve trigger slugs and emit events. */
  nodeRefs?: NodeChipContext;
  /**
   * When true, the chart / number renderer fills its parent's height instead
   * of using fixed pixel dimensions. Used by single-widget panels in Apps so
   * the chart resizes with the panel. The parent MUST provide a defined
   * height (`h-full` from a flex column ancestor is fine).
   */
  fill?: boolean;
}

//
// Column / format types (unchanged from increments 2/3).
//

type Format = "plain" | "link" | { kind: "linkLabel"; label: string } | "relative" | "date" | "badge" | "code";

interface ColumnSpec {
  label: string;
  field: MaybeExpr;
  format: Format;
}

//
// Filter types (unchanged from increment 3).
//

const FILTER_OPS = ["eq", "neq", "contains", "not_contains", "gt", "lt", "exists", "not_exists"] as const;
type FilterOp = (typeof FILTER_OPS)[number];

interface FilterCondition {
  field: MaybeExpr;
  op: FilterOp;
  /** `value` is parsed as a `MaybeExpr` so authors can either compare against a literal scalar or a CEL expression like `{{ int(now) - 7200 }}`. Unused for `exists` / `not_exists`. */
  value: MaybeExpr;
}

//
// Action types (extended in v5 with kind discriminator + show condition).
//

const ACTION_VARIANTS = ["default", "danger", "primary"] as const;
type ActionVariant = (typeof ACTION_VARIANTS)[number];

const ACTION_ICONS = ["trash", "play", "refresh", "stop", "external-link"] as const;
type ActionIcon = (typeof ACTION_ICONS)[number];

const ACTION_KINDS = ["trigger", "approve", "cancel", "push-through"] as const;
type ActionKind = (typeof ACTION_KINDS)[number];

interface ActionBase {
  label: string;
  variant: ActionVariant;
  icon?: ActionIcon;
  /** Compiled at parse time so confirm dialogs can interpolate per-row values without re-parsing on every render. */
  confirm?: CompiledTemplate;
  /**
   * `MaybeExpr` so a CEL `show: "{{ status == 'running' }}"` is compiled
   * once, while a legacy `show: 'status == "running"'` falls back to the
   * old `evalShow` simple comparator.
   */
  show?: MaybeExpr;
}

interface TriggerActionSpec extends ActionBase {
  kind: "trigger";
  trigger: string;
  template?: string;
  /** Each value is compiled as a template; the runtime-evaluated payload values are coerced to strings to keep the trigger payload contract unchanged. */
  fill?: Record<string, CompiledTemplate>;
}

interface ApproveActionSpec extends ActionBase {
  kind: "approve";
  node: string;
}

interface CancelActionSpec extends ActionBase {
  kind: "cancel";
}

interface PushThroughActionSpec extends ActionBase {
  kind: "push-through";
  node: string;
}

type ActionSpec = TriggerActionSpec | ApproveActionSpec | CancelActionSpec | PushThroughActionSpec;

//
// Render mode types (v5).
//

type AggregateOp = "avg" | "sum" | "min" | "max" | "count";
const AGGREGATE_OPS: readonly AggregateOp[] = ["avg", "sum", "min", "max", "count"];

type ChartType = "bar" | "line" | "area" | "stacked-bar" | "donut";
const CHART_TYPES: readonly ChartType[] = ["bar", "line", "area", "stacked-bar", "donut"];

type ChartColorKeyword = "blue" | "sky" | "green" | "red" | "yellow" | "gray";
const CHART_COLORS: readonly ChartColorKeyword[] = ["blue", "sky", "green", "red", "yellow", "gray"];

type ChartYUnit = "ms" | "s";

interface ChartSpec {
  type: ChartType;
  x: MaybeExpr;
  /** Required for bar | line | area | stacked-bar. Optional for donut. */
  y?: MaybeExpr;
  /** Required for stacked-bar; ignored elsewhere. Dot-path or CEL expression. */
  group?: MaybeExpr;
  label?: string;
  aggregate?: AggregateOp;
  /** Single-series only (bar | line | area). Ignored for stacked-bar / donut. */
  color?: ChartColorKeyword;
  /**
   * Y-axis units: `ms` divides values by 1000 so axes show seconds (`30.7s`).
   * Also accepts YAML `y_unit`. When omitted, widgets whose `label` mentions
   * “duration” may auto-detect ms from large magnitudes (see inferChartYUnitFromRows).
   */
  yUnit?: ChartYUnit;
}

interface NumberSpec {
  field: MaybeExpr;
  aggregate: AggregateOp;
  label: string;
  format: "number" | "duration" | "percent";
  /** When true, renders a small area chart of per-row values behind the stat. */
  sparkline: boolean;
}

type RenderSpec = { kind: "table" } | { kind: "chart"; chart: ChartSpec } | { kind: "number"; number: NumberSpec };

//
// Parsed widget — discriminated by source.
//

const WIDGET_RUN_STATUSES = ["running", "passed", "failed", "cancelled"] as const;
type WidgetRunStatus = (typeof WIDGET_RUN_STATUSES)[number];

type ParsedWidget =
  | {
      source: "memory";
      namespace: string;
      columns?: ColumnSpec[];
      where?: FilterCondition[];
      actions?: ActionSpec[];
      render: RenderSpec;
    }
  | {
      source: "executions";
      trigger?: string;
      status?: WidgetRunStatus;
      limit: number;
      columns?: ColumnSpec[];
      where?: FilterCondition[];
      actions?: ActionSpec[];
      render: RenderSpec;
    };

interface WidgetParseError {
  message: string;
}

type ParseResult = { kind: "ok"; widget: ParsedWidget } | { kind: "error"; error: WidgetParseError };
type ParsePartResult<T> = { kind: "ok"; value: T | undefined } | { kind: "error"; error: WidgetParseError };

const DEFAULT_LIMIT = 10;
const MAX_LIMIT = 100;

function parseWidgetBody(body: string): ParseResult {
  let parsed: unknown;
  try {
    parsed = yaml.load(body);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { kind: "error", error: { message: `Invalid widget block: ${message}` } };
  }

  if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return {
      kind: "error",
      error: { message: "Invalid widget block: expected an object with `source`" },
    };
  }

  const obj = parsed as Record<string, unknown>;
  const source = obj.source;

  if (typeof source !== "string" || source.trim() === "") {
    return { kind: "error", error: { message: "Invalid widget block: missing `source`" } };
  }
  if (source !== "memory" && source !== "executions") {
    return {
      kind: "error",
      error: {
        message: `Invalid widget block: unsupported source "${source}" (only "memory" and "executions" are supported)`,
      },
    };
  }

  const columnsResult = parseColumns(obj.columns);
  if (columnsResult.kind === "error") return columnsResult;

  const whereResult = parseWhere(obj.where);
  if (whereResult.kind === "error") return whereResult;

  const actionsResult = parseActions(obj.actions);
  if (actionsResult.kind === "error") return actionsResult;

  const renderResult = parseRender(obj.render);
  if (renderResult.kind === "error") return renderResult;

  if (source === "memory") {
    const namespace = obj.namespace;
    if (typeof namespace !== "string" || namespace.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: missing `namespace`" } };
    }
    return {
      kind: "ok",
      widget: {
        source: "memory",
        namespace,
        columns: columnsResult.value,
        where: whereResult.value,
        actions: actionsResult.value,
        render: renderResult.value!,
      },
    };
  }

  // executions
  const trigger = obj.trigger;
  if (trigger !== undefined && (typeof trigger !== "string" || trigger.trim() === "")) {
    return { kind: "error", error: { message: "Invalid widget block: `trigger` must be a string" } };
  }
  let status: WidgetRunStatus | undefined;
  if (obj.status !== undefined) {
    if (typeof obj.status !== "string") {
      return { kind: "error", error: { message: "Invalid widget block: `status` must be a string" } };
    }
    if (!(WIDGET_RUN_STATUSES as readonly string[]).includes(obj.status)) {
      return {
        kind: "error",
        error: { message: `Invalid widget block: unsupported status "${obj.status}"` },
      };
    }
    status = obj.status as WidgetRunStatus;
  }
  let limit = DEFAULT_LIMIT;
  if (obj.limit !== undefined) {
    if (typeof obj.limit !== "number" || !Number.isFinite(obj.limit) || obj.limit <= 0) {
      return { kind: "error", error: { message: "Invalid widget block: `limit` must be a positive number" } };
    }
    limit = Math.min(Math.floor(obj.limit), MAX_LIMIT);
  }

  return {
    kind: "ok",
    widget: {
      source: "executions",
      trigger: typeof trigger === "string" ? trigger : undefined,
      status,
      limit,
      columns: columnsResult.value,
      where: whereResult.value,
      actions: actionsResult.value,
      render: renderResult.value!,
    },
  };
}

function parseColumns(raw: unknown): ParsePartResult<ColumnSpec[]> {
  if (raw === undefined || raw === null) return { kind: "ok", value: undefined };
  if (!Array.isArray(raw)) {
    return { kind: "error", error: { message: "Invalid widget block: `columns` must be a list" } };
  }
  if (raw.length === 0) {
    return { kind: "error", error: { message: "Invalid widget block: `columns` must not be empty" } };
  }

  const result: ColumnSpec[] = [];
  for (let i = 0; i < raw.length; i++) {
    const item = raw[i];
    if (!isPlainObject(item)) {
      return { kind: "error", error: { message: `Invalid widget block: columns[${i}] must be an object` } };
    }
    const label = item.label;
    const field = item.field;
    const formatRaw = item.format;
    if (typeof label !== "string" || label.trim() === "") {
      return { kind: "error", error: { message: `Invalid widget block: columns[${i}] missing \`label\`` } };
    }
    if (typeof field !== "string" || field.trim() === "") {
      return { kind: "error", error: { message: `Invalid widget block: columns[${i}] missing \`field\`` } };
    }
    if (formatRaw !== undefined && typeof formatRaw !== "string") {
      return {
        kind: "error",
        error: { message: `Invalid widget block: columns[${i}].format must be a string` },
      };
    }
    result.push({ label, field: compileMaybeExpr(field), format: parseFormat(formatRaw) });
  }
  return { kind: "ok", value: result };
}

function parseFormat(raw: string | undefined): Format {
  if (raw === undefined) return "plain";
  if (raw === "link") return "link";
  if (raw.startsWith("link:")) {
    return { kind: "linkLabel", label: raw.slice("link:".length) };
  }
  if (raw === "plain" || raw === "relative" || raw === "date" || raw === "badge" || raw === "code") {
    return raw;
  }
  // eslint-disable-next-line no-console
  console.warn(`[WidgetBlock] Unknown format "${raw}", falling back to plain text`);
  return "plain";
}

function parseWhere(raw: unknown): ParsePartResult<FilterCondition[]> {
  if (raw === undefined || raw === null) return { kind: "ok", value: undefined };
  if (!Array.isArray(raw)) {
    return { kind: "error", error: { message: "Invalid widget block: `where` must be a list" } };
  }
  if (raw.length === 0) {
    return { kind: "error", error: { message: "Invalid widget block: `where` must not be empty" } };
  }

  const result: FilterCondition[] = [];
  for (let i = 0; i < raw.length; i++) {
    const item = raw[i];
    if (!isPlainObject(item)) {
      return { kind: "error", error: { message: `Invalid widget block: where[${i}] must be an object` } };
    }
    const field = item.field;
    const op = item.op;
    const value = item.value;
    if (typeof field !== "string" || field.trim() === "") {
      return { kind: "error", error: { message: `Invalid widget block: where[${i}] missing \`field\`` } };
    }
    if (typeof op !== "string" || op.trim() === "") {
      return { kind: "error", error: { message: `Invalid widget block: where[${i}] missing \`op\`` } };
    }
    if (!isFilterOp(op)) {
      return { kind: "error", error: { message: `Unknown filter operator: "${op}"` } };
    }
    const opNeedsValue = op !== "exists" && op !== "not_exists";
    if (opNeedsValue) {
      if (value === undefined || value === null) {
        return { kind: "error", error: { message: `Invalid widget block: where[${i}] missing \`value\`` } };
      }
      if (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: where[${i}].value must be a scalar` },
        };
      }
    }
    result.push({
      field: compileMaybeExpr(field),
      op,
      value: compileMaybeExpr(opNeedsValue ? String(value) : ""),
    });
  }
  return { kind: "ok", value: result };
}

function isFilterOp(op: string): op is FilterOp {
  return (FILTER_OPS as readonly string[]).includes(op);
}

function parseActions(raw: unknown): ParsePartResult<ActionSpec[]> {
  if (raw === undefined || raw === null) return { kind: "ok", value: undefined };
  if (!Array.isArray(raw)) {
    return { kind: "error", error: { message: "Invalid widget block: `actions` must be a list" } };
  }
  if (raw.length === 0) {
    return { kind: "error", error: { message: "Invalid widget block: `actions` must not be empty" } };
  }

  const result: ActionSpec[] = [];
  for (let i = 0; i < raw.length; i++) {
    const item = raw[i];
    if (!isPlainObject(item)) {
      return { kind: "error", error: { message: `Invalid widget block: actions[${i}] must be an object` } };
    }
    const label = item.label;
    const confirm = item.confirm;
    const variantRaw = item.variant;
    const iconRaw = item.icon;
    const showRaw = item.show;

    if (typeof label !== "string" || label.trim() === "") {
      return { kind: "error", error: { message: `Invalid widget block: actions[${i}] missing \`label\`` } };
    }
    if (confirm !== undefined && typeof confirm !== "string") {
      return {
        kind: "error",
        error: { message: `Invalid widget block: actions[${i}].confirm must be a string` },
      };
    }
    if (showRaw !== undefined && typeof showRaw !== "string") {
      return {
        kind: "error",
        error: { message: `Invalid widget block: actions[${i}].show must be a string` },
      };
    }

    let variant: ActionVariant = "default";
    if (variantRaw !== undefined) {
      if (typeof variantRaw !== "string") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: actions[${i}].variant must be a string` },
        };
      }
      const normalized = variantRaw.trim().toLowerCase();
      const variantAliases: Record<string, ActionVariant> = {
        destructive: "danger",
      };
      const resolved = variantAliases[normalized] ?? normalized;
      if ((ACTION_VARIANTS as readonly string[]).includes(resolved)) {
        variant = resolved as ActionVariant;
      } else {
        // eslint-disable-next-line no-console
        console.warn(`[WidgetBlock] Unknown action variant "${variantRaw}", falling back to default`);
      }
    }

    let icon: ActionIcon | undefined;
    if (iconRaw !== undefined) {
      if (typeof iconRaw !== "string") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: actions[${i}].icon must be a string` },
        };
      }
      if ((ACTION_ICONS as readonly string[]).includes(iconRaw)) {
        icon = iconRaw as ActionIcon;
      } else {
        // eslint-disable-next-line no-console
        console.warn(`[WidgetBlock] Unknown action icon "${iconRaw}", omitting icon`);
      }
    }

    const base: ActionBase = {
      label,
      variant,
      icon,
      confirm: typeof confirm === "string" ? compileTemplate(confirm) : undefined,
      show: typeof showRaw === "string" ? compileMaybeExpr(showRaw) : undefined,
    };

    // Resolve discriminator: explicit `kind`, or back-compat default to "trigger" when `trigger` is set.
    let kindRaw = item.kind;
    if (kindRaw === undefined && typeof item.trigger === "string") kindRaw = "trigger";
    if (typeof kindRaw !== "string") {
      return { kind: "error", error: { message: `Invalid widget block: actions[${i}] missing \`kind\`` } };
    }
    if (!(ACTION_KINDS as readonly string[]).includes(kindRaw)) {
      return {
        kind: "error",
        error: { message: `Invalid widget block: actions[${i}].kind "${kindRaw}" is not supported` },
      };
    }
    const kind = kindRaw as ActionKind;

    if (kind === "trigger") {
      const trigger = item.trigger;
      const template = item.template;
      const fill = item.fill;
      if (typeof trigger !== "string" || trigger.trim() === "") {
        return { kind: "error", error: { message: `Invalid widget block: actions[${i}] missing \`trigger\`` } };
      }
      if (template !== undefined && typeof template !== "string") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: actions[${i}].template must be a string` },
        };
      }
      let parsedFill: Record<string, CompiledTemplate> | undefined;
      if (fill !== undefined && fill !== null) {
        if (!isPlainObject(fill)) {
          return {
            kind: "error",
            error: { message: `Invalid widget block: actions[${i}].fill must be an object` },
          };
        }
        parsedFill = {};
        for (const [path, value] of Object.entries(fill)) {
          if (typeof value !== "string") {
            return {
              kind: "error",
              error: {
                message: `Invalid widget block: actions[${i}].fill.${path} must be a string`,
              },
            };
          }
          parsedFill[path] = compileTemplate(value);
        }
      }
      result.push({
        ...base,
        kind: "trigger",
        trigger,
        template: typeof template === "string" ? template : undefined,
        fill: parsedFill,
      });
      continue;
    }

    if (kind === "approve" || kind === "push-through") {
      const node = item.node;
      if (typeof node !== "string" || node.trim() === "") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: actions[${i}] missing \`node\` (required for ${kind})` },
        };
      }
      result.push({ ...base, kind, node });
      continue;
    }

    // kind === "cancel"
    result.push({ ...base, kind: "cancel" });
  }
  return { kind: "ok", value: result };
}

function parseChartYUnitField(raw: unknown): ChartYUnit | undefined {
  if (typeof raw !== "string") return undefined;
  const u = raw.trim().toLowerCase();
  if (u === "ms" || u === "milliseconds" || u === "millisecond") return "ms";
  if (u === "s" || u === "sec" || u === "secs" || u === "second" || u === "seconds") return "s";
  return undefined;
}

function parseRender(raw: unknown): ParsePartResult<RenderSpec> {
  if (raw === undefined || raw === null) return { kind: "ok", value: { kind: "table" } };
  if (!isPlainObject(raw)) {
    return { kind: "error", error: { message: "Invalid widget block: `render` must be an object" } };
  }
  const kind = raw.kind;
  if (typeof kind !== "string") {
    return { kind: "error", error: { message: "Invalid widget block: `render.kind` must be a string" } };
  }
  if (kind === "table") return { kind: "ok", value: { kind: "table" } };
  if (kind === "chart") {
    const chart = raw.chart;
    if (!isPlainObject(chart)) {
      return { kind: "error", error: { message: "Invalid widget block: `render.chart` must be an object" } };
    }
    if (typeof chart.type !== "string" || !(CHART_TYPES as readonly string[]).includes(chart.type)) {
      return {
        kind: "error",
        error: { message: `Invalid widget block: render.chart.type "${String(chart.type)}" is not supported` },
      };
    }
    const type = chart.type as ChartType;
    if (typeof chart.x !== "string" || chart.x.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: `render.chart.x` is required" } };
    }
    let yRaw: string | undefined;
    if (type !== "donut") {
      if (typeof chart.y !== "string" || chart.y.trim() === "") {
        return {
          kind: "error",
          error: { message: `Invalid widget block: render.chart.y is required for ${type}` },
        };
      }
      yRaw = chart.y;
    } else if (chart.y !== undefined) {
      if (typeof chart.y !== "string" || chart.y.trim() === "") {
        return {
          kind: "error",
          error: { message: "Invalid widget block: `render.chart.y` must be a non-empty string" },
        };
      }
      yRaw = chart.y;
    }
    let groupRaw: string | undefined;
    if (type === "stacked-bar") {
      if (typeof chart.group !== "string" || chart.group.trim() === "") {
        return {
          kind: "error",
          error: { message: "Invalid widget block: render.chart.group is required for stacked-bar" },
        };
      }
      groupRaw = chart.group;
    }
    let aggregate: AggregateOp | undefined;
    if (chart.aggregate !== undefined) {
      if (typeof chart.aggregate !== "string" || !AGGREGATE_OPS.includes(chart.aggregate as AggregateOp)) {
        return {
          kind: "error",
          error: {
            message: `Invalid widget block: render.chart.aggregate "${String(chart.aggregate)}" is not supported`,
          },
        };
      }
      aggregate = chart.aggregate as AggregateOp;
    }
    if (type === "donut" && aggregate && aggregate !== "count" && !yRaw) {
      return {
        kind: "error",
        error: {
          message: "Invalid widget block: render.chart.y is required when donut uses aggregate other than count",
        },
      };
    }
    let color: ChartColorKeyword | undefined;
    if (chart.color !== undefined) {
      if (typeof chart.color !== "string") {
        return { kind: "error", error: { message: "Invalid widget block: `render.chart.color` must be a string" } };
      }
      if ((CHART_COLORS as readonly string[]).includes(chart.color)) {
        color = chart.color as ChartColorKeyword;
      } else {
        // Unknown color falls back to the default palette slot rather than failing.
        // eslint-disable-next-line no-console
        console.warn(`[WidgetBlock] Unknown chart color "${chart.color}", falling back to default`);
      }
    }

    let yUnit: ChartYUnit | undefined;
    const yUnitRaw = (chart as Record<string, unknown>).y_unit ?? (chart as Record<string, unknown>).yUnit;
    if (yUnitRaw !== undefined && yUnitRaw !== null) {
      const parsed = parseChartYUnitField(yUnitRaw);
      if (parsed) {
        yUnit = parsed;
      } else {
        // eslint-disable-next-line no-console
        console.warn(`[WidgetBlock] Unknown render.chart.y_unit "${String(yUnitRaw)}", ignoring`);
      }
    }

    return {
      kind: "ok",
      value: {
        kind: "chart",
        chart: {
          type,
          x: compileMaybeExpr(chart.x),
          y: yRaw !== undefined ? compileMaybeExpr(yRaw) : undefined,
          group: groupRaw !== undefined ? compileMaybeExpr(groupRaw) : undefined,
          label: typeof chart.label === "string" ? chart.label : undefined,
          aggregate,
          color,
          yUnit,
        },
      },
    };
  }
  if (kind === "number") {
    const num = raw.number;
    if (!isPlainObject(num)) {
      return { kind: "error", error: { message: "Invalid widget block: `render.number` must be an object" } };
    }
    const aggregate = num.aggregate;
    if (typeof num.aggregate !== "string" || !AGGREGATE_OPS.includes(num.aggregate as AggregateOp)) {
      return {
        kind: "error",
        error: { message: `Invalid widget block: render.number.aggregate "${String(aggregate)}" is not supported` },
      };
    }
    if (typeof num.field !== "string" || num.field.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: `render.number.field` is required" } };
    }
    if (typeof num.label !== "string" || num.label.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: `render.number.label` is required" } };
    }
    let format: NumberSpec["format"] = "number";
    if (num.format !== undefined) {
      if (
        typeof num.format !== "string" ||
        (num.format !== "number" && num.format !== "duration" && num.format !== "percent")
      ) {
        return {
          kind: "error",
          error: { message: `Invalid widget block: render.number.format "${String(num.format)}" is not supported` },
        };
      }
      format = num.format;
    }
    let sparkline = false;
    if (num.sparkline !== undefined) {
      if (typeof num.sparkline !== "boolean") {
        return { kind: "error", error: { message: "Invalid widget block: render.number.sparkline must be a boolean" } };
      }
      sparkline = num.sparkline;
    }
    return {
      kind: "ok",
      value: {
        kind: "number",
        number: {
          field: compileMaybeExpr(num.field),
          aggregate: num.aggregate as AggregateOp,
          label: num.label,
          format,
          sparkline,
        },
      },
    };
  }
  return {
    kind: "error",
    error: { message: `Invalid widget block: render.kind "${kind}" is not supported` },
  };
}

//
// Generic helpers
//

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return value != null && typeof value === "object" && !Array.isArray(value);
}

function stringifyCell(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function getByPath(obj: unknown, path: string): unknown {
  let cur: unknown = obj;
  for (const part of path.split(".")) {
    if (!isPlainObject(cur)) return undefined;
    cur = cur[part];
  }
  return cur;
}

const SHOW_RE = /^\s*([\w.-]+)\s*(==|!=)\s*"([^"]*)"\s*$/;

/**
 * Evaluate a `show` condition.
 *
 * - `undefined` (no `show:` authored) → button is visible.
 * - `MaybeExpr.kind === "expr"` → CEL: any truthy non-undefined result shows the button. Runtime / parse errors fail-closed (button hidden).
 * - `MaybeExpr.kind === "literal"` → legacy simple comparator (`field == "x"` / `field != "x"`). Anything that doesn't match the comparator regex fails-closed.
 */
function evalShow(condition: MaybeExpr | undefined, row: Record<string, unknown>, env: ExprEnv): boolean {
  if (!condition) return true;
  if (condition.kind === "expr") {
    const result = evalRowFieldExpr(condition, row, env, getByPath);
    if (result === undefined) return false;
    return Boolean(result);
  }
  const literal = condition.value;
  const m = literal.match(SHOW_RE);
  if (!m) return false;
  const [, path, op, expected] = m;
  const actual = stringifyCell(getByPath(row, path));
  return op === "==" ? actual === expected : actual !== expected;
}

function setPath(obj: Record<string, unknown>, path: string, value: unknown): void {
  const parts = path.split(".");
  let cur: Record<string, unknown> = obj;
  for (let i = 0; i < parts.length - 1; i++) {
    const k = parts[i];
    if (!isPlainObject(cur[k])) cur[k] = {};
    cur = cur[k] as Record<string, unknown>;
  }
  cur[parts[parts.length - 1]] = value;
}

function buildFillPayload(
  fill: Record<string, CompiledTemplate> | undefined,
  row: Record<string, unknown>,
  env: ExprEnv,
): Record<string, unknown> {
  // Trigger payload values are always strings (current contract). The CEL
  // template's evaluated result is stringified so authors get type-safe
  // `{{ ... }}` interpolation without changing what the trigger receives.
  const payload: Record<string, unknown> = {};
  if (!fill) return payload;
  for (const [path, template] of Object.entries(fill)) {
    setPath(payload, path, evalTemplateExpr(template, row, env, stringifyCell));
  }
  return payload;
}

//
// Filter engine
//

function evalCondition(values: Record<string, unknown>, cond: FilterCondition, env: ExprEnv): boolean {
  // `field` follows the legacy convention: a literal string is a dot-path
  // into the row; a CEL `{{ ... }}` is evaluated against the row.
  const raw = evalRowFieldExpr(cond.field, values, env, getByPath);
  const has = raw !== undefined;
  const val = raw == null ? "" : typeof raw === "string" ? raw : stringifyCell(raw);

  if (cond.op === "exists") return val !== "";
  if (cond.op === "not_exists") return val === "";

  if (!has) return false;

  // `value`, on the other hand, has always been a literal scalar to compare
  // against. Only `{{ ... }}` opts into CEL; a bare string is the raw
  // expected value and is NOT looked up against the row.
  let expected: string;
  if (cond.value.kind === "literal") {
    expected = cond.value.value;
  } else {
    const expectedRaw = evalRowFieldExpr(cond.value, values, env, getByPath);
    expected = expectedRaw == null ? "" : typeof expectedRaw === "string" ? expectedRaw : stringifyCell(expectedRaw);
  }

  switch (cond.op) {
    case "eq":
      return val === expected;
    case "neq":
      return val !== expected;
    case "contains":
      return val.includes(expected);
    case "not_contains":
      return !val.includes(expected);
    case "gt":
    case "lt": {
      const a = parseFloat(val);
      const b = parseFloat(expected);
      if (Number.isNaN(a) || Number.isNaN(b)) return false;
      return cond.op === "gt" ? a > b : a < b;
    }
  }
}

function applyWhereOnRows<T extends Record<string, unknown>>(rows: T[], where: FilterCondition[], env: ExprEnv): T[] {
  return rows.filter((row) => where.every((cond) => evalCondition(row, cond, env)));
}

//
// Executions data path
//

interface ExecutionsRow extends Record<string, unknown> {
  id: string;
  root: CanvasesCanvasEventWithExecutions;
  status: WidgetRunStatus;
  duration: number | null;
  executions: CanvasesCanvasNodeExecutionRef[];
  node: Record<string, CanvasesCanvasNodeExecutionRef>;
}

function normalizeStatus(aggregate: string): WidgetRunStatus {
  switch (aggregate) {
    case "running":
    case "queued":
      return "running";
    case "completed":
      return "passed";
    case "error":
      return "failed";
    case "cancelled":
      return "cancelled";
    default:
      return "running";
  }
}

function deriveDurationSeconds(event: CanvasesCanvasEventWithExecutions): number | null {
  const startMs = event.createdAt ? new Date(event.createdAt).getTime() : NaN;
  if (!Number.isFinite(startMs)) return null;

  const isStillRunning = (event.executions ?? []).some(
    (e) => e.state === "STATE_STARTED" || e.state === "STATE_PENDING",
  );
  let endMs: number;
  if (isStillRunning) {
    endMs = Date.now();
  } else {
    let latest = startMs;
    for (const exec of event.executions ?? []) {
      const ts = exec.updatedAt ? new Date(exec.updatedAt).getTime() : NaN;
      if (Number.isFinite(ts) && ts > latest) latest = ts;
    }
    endMs = latest;
  }
  const seconds = (endMs - startMs) / 1000;
  return Number.isFinite(seconds) && seconds >= 0 ? seconds : null;
}

function indexExecutionsByNodeId(
  executions: CanvasesCanvasNodeExecutionRef[],
): Record<string, CanvasesCanvasNodeExecutionRef> {
  const result: Record<string, CanvasesCanvasNodeExecutionRef> = {};
  for (const exec of executions) {
    if (!exec.nodeId) continue;
    const existing = result[exec.nodeId];
    if (!existing) {
      result[exec.nodeId] = exec;
      continue;
    }
    // Keep the latest by updatedAt (fallback createdAt).
    const incoming = new Date(exec.updatedAt || exec.createdAt || 0).getTime();
    const current = new Date(existing.updatedAt || existing.createdAt || 0).getTime();
    if (incoming >= current) result[exec.nodeId] = exec;
  }
  return result;
}

function eventToRow(event: CanvasesCanvasEventWithExecutions): ExecutionsRow {
  const executions = event.executions ?? [];
  const queueItems = event.queueItems ?? [];
  const aggregate = getAggregateStatus(executions);
  return {
    id: event.id || "",
    root: event,
    status: normalizeStatus(aggregate),
    duration: deriveDurationSeconds(event),
    executions,
    node: indexExecutionsByNodeId(executions),
  };
}

interface UseExecutionRowsOpts {
  trigger?: string;
  status?: WidgetRunStatus;
  limit: number;
  enabled: boolean;
}

interface ExecutionRowsResult {
  rows: ExecutionsRow[];
  isLoading: boolean;
  isError: boolean;
  error: unknown;
}

function useWidgetExecutionRows(canvasId: string, opts: UseExecutionRowsOpts): ExecutionRowsResult {
  const query = useInfiniteCanvasEvents(canvasId, opts.enabled);

  const rows = useMemo<ExecutionsRow[]>(() => {
    if (!query.data) return [];
    const events: CanvasesCanvasEventWithExecutions[] = [];
    for (const page of query.data.pages) {
      const items = page?.events ?? [];
      events.push(...items);
    }
    let mapped = events.map(eventToRow);
    if (opts.trigger) {
      mapped = mapped.filter((row) => row.root.nodeId === opts.trigger);
    }
    if (opts.status) {
      mapped = mapped.filter((row) => row.status === opts.status);
    }
    return mapped.slice(0, opts.limit);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [query.data, opts.trigger, opts.status, opts.limit]);

  return {
    rows,
    isLoading: opts.enabled && query.isLoading,
    isError: opts.enabled && query.isError,
    error: query.error,
  };
}

//
// Cell renderers
//

function truncate(value: string, max: number): string {
  if (value.length <= max) return value;
  return value.slice(0, Math.max(0, max - 1)) + "…";
}

const STATUS_BADGE_CLASS: Record<WidgetRunStatus, string> = {
  running: "bg-blue-100 text-blue-700 border-blue-950/20",
  passed: "bg-emerald-100 text-emerald-700 border-emerald-950/20",
  failed: "bg-red-100 text-red-700 border-red-950/20",
  cancelled: "bg-slate-100 text-slate-600 border-slate-950/20",
};

const BADGE_PILL_BASE = "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none";
const DEFAULT_BADGE_PILL = `${BADGE_PILL_BASE} bg-emerald-100 text-emerald-700 border-emerald-950/20`;

function renderCell(value: unknown, format: Format): React.ReactNode {
  const text = stringifyCell(value);
  if (text === "") return null;

  if (typeof format === "object" && format.kind === "linkLabel") {
    return <CellLink href={text} display={format.label} />;
  }

  switch (format) {
    case "plain":
      return text;
    case "link":
      return <CellLink href={text} display={truncate(text, 40)} />;
    case "relative":
      return <CellRelative raw={text} />;
    case "date":
      return <CellDate raw={text} />;
    case "badge": {
      // Use a status-aware palette when the value matches a known run status; fall back to the default emerald pill otherwise.
      const knownStatus = (WIDGET_RUN_STATUSES as readonly string[]).includes(text);
      const className = knownStatus
        ? `${BADGE_PILL_BASE} ${STATUS_BADGE_CLASS[text as WidgetRunStatus]}`
        : DEFAULT_BADGE_PILL;
      return <span className={className}>{text}</span>;
    }
    case "code":
      return (
        <code className="rounded border border-slate-300 bg-slate-100 px-1.5 py-0.5 font-mono text-[0.85em] text-slate-800">
          {text}
        </code>
      );
    default:
      return text;
  }
}

function CellLink({ href, display }: { href: string; display: string }) {
  const appsPanelLinks = useContext(AppsPanelMarkdownLinksContext);
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className={cn(
        "inline-flex items-center gap-0.5 underline",
        appsPanelLinks ? "text-sky-600 hover:text-sky-700" : "text-blue-600 hover:text-blue-700",
      )}
    >
      {display}
      <ExternalLink className="inline h-3 w-3 shrink-0" />
    </a>
  );
}

function resolveDate(raw: string): Date | null {
  const trimmed = raw.trim();
  if (trimmed === "") return null;
  if (/^-?\d+(\.\d+)?$/.test(trimmed)) {
    const n = Number(trimmed);
    const ms = Math.abs(n) < 1e12 ? n * 1000 : n;
    const d = new Date(ms);
    return isNaN(d.getTime()) ? null : d;
  }
  const d = new Date(trimmed);
  return isNaN(d.getTime()) ? null : d;
}

function CellRelative({ raw }: { raw: string }) {
  const date = resolveDate(raw);
  if (!date) return <>{raw}</>;
  return <span title={formatAbsoluteUtc(date)}>{formatRelativeTime(date.toISOString(), false)}</span>;
}

function CellDate({ raw }: { raw: string }) {
  const date = resolveDate(raw);
  if (!date) return <>{raw}</>;
  return <span>{formatAbsoluteUtc(date)}</span>;
}

function formatAbsoluteUtc(date: Date): string {
  return date.toISOString().replace("T", " ").slice(0, 16) + " UTC";
}

//
// Action button (shared outline style — white/bordered, matches app Button outline)
//

const ICON_MAP: Record<ActionIcon, React.ComponentType<{ className?: string }>> = {
  trash: Trash2,
  play: Play,
  refresh: RefreshCw,
  stop: Square,
  "external-link": ExternalLink,
};

/** Normalize widget action labels for keyword matching (NBSP / ZWSP / bidi marks). */
function normalizeWidgetActionLabel(label: string): string {
  return label.replace(/[\uFEFF\u200B-\u200D\u2060\u200E\u200F]/g, "").trim();
}

/** Strong semantic icons from button wording (runs before explicit YAML `icon:`). */
function iconFromActionLabel(label: string): React.ComponentType<{ className?: string }> | null {
  const normalized = normalizeWidgetActionLabel(label).toLowerCase();
  const words = normalized.split(/[\s/]+/).filter(Boolean);
  if (words.includes("approve")) return Check;
  if (words.includes("reject") || words.includes("deny")) return X;
  if (words.includes("rollback")) return RotateCcw;
  if (words.includes("reapply")) return ArrowRight;
  return null;
}

/** Pick Trigger / Approve / Cancel / … icons: label keywords → YAML `icon` → kind/variant defaults. */
function resolvedWidgetActionIcon(action: ActionSpec): React.ComponentType<{ className?: string }> | null {
  const fromLabel = iconFromActionLabel(action.label);
  if (fromLabel) return fromLabel;

  if (action.icon) {
    const mapped = ICON_MAP[action.icon];
    if (mapped) return mapped;
  }

  if (action.kind === "approve") return Check;
  if (action.kind === "cancel") return X;

  if (action.kind === "trigger") {
    if (action.variant === "danger") return RotateCcw;
    if (action.variant === "primary") return ArrowRight;
  }

  return null;
}

function findRunningExecution(executions: CanvasesCanvasNodeExecutionRef[]): CanvasesCanvasNodeExecutionRef | null {
  // Most recent (by updatedAt fallback createdAt) execution still running.
  let best: CanvasesCanvasNodeExecutionRef | null = null;
  let bestTs = -Infinity;
  for (const exec of executions) {
    if (exec.state !== "STATE_STARTED" && exec.state !== "STATE_PENDING") continue;
    const ts = new Date(exec.updatedAt || exec.createdAt || 0).getTime();
    if (ts >= bestTs) {
      best = exec;
      bestTs = ts;
    }
  }
  return best;
}

function findApprovalExecution(
  executions: CanvasesCanvasNodeExecutionRef[],
  nodeId: string,
): CanvasesCanvasNodeExecutionRef | null {
  // The execution at the given node that's currently running (i.e. waiting for input).
  for (const exec of executions) {
    if (exec.nodeId === nodeId && exec.state === "STATE_STARTED") return exec;
  }
  return null;
}

function actionId(action: ActionSpec): string {
  if (action.kind === "trigger") return action.trigger;
  if (action.kind === "approve" || action.kind === "push-through") return action.node;
  return "cancel";
}

function ActionButtons({
  row,
  actions,
  nodeRefs,
  env,
}: {
  row: Record<string, unknown>;
  actions: ActionSpec[];
  nodeRefs?: NodeChipContext;
  env: ExprEnv;
}) {
  return (
    <div className="flex flex-wrap items-center gap-2">
      {actions.map((action, i) => (
        <ActionButton
          key={`${action.kind}-${actionId(action)}-${i}`}
          row={row}
          action={action}
          nodeRefs={nodeRefs}
          env={env}
        />
      ))}
    </div>
  );
}

interface ResolvedAction {
  testIdSuffix: string;
  /** Whether the button should be rendered at all (after `show` and kind-specific resolution). */
  visible: boolean;
  /** Tooltip when disabled-but-visible (e.g. trigger slug missing). */
  tooltip?: string;
  /** The function that performs the actual API call. */
  fire?: () => Promise<void>;
}

function ActionButton({
  row,
  action,
  nodeRefs,
  env,
}: {
  row: Record<string, unknown>;
  action: ActionSpec;
  nodeRefs?: NodeChipContext;
  env: ExprEnv;
}) {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isFiring, setIsFiring] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const passesShow = evalShow(action.show, row, env);
  const resolved = resolveAction(action, row, nodeRefs, env);

  if (!passesShow || !resolved.visible) return null;

  const Icon = resolvedWidgetActionIcon(action);
  const disabled = !resolved.fire || isFiring;

  const fire = async () => {
    if (!resolved.fire) return;
    setIsFiring(true);
    setError(null);
    try {
      await resolved.fire();
      showSuccessToast(`Triggered: ${action.label}`);
    } catch (err) {
      const message = err instanceof Error ? err.message : String(err);
      setError(message);
    } finally {
      setIsFiring(false);
    }
  };

  const handleClick = () => {
    if (disabled) return;
    if (action.confirm) {
      setIsConfirmOpen(true);
      return;
    }
    void fire();
  };

  const handleConfirmFire = async () => {
    setIsConfirmOpen(false);
    await fire();
  };

  const testIdSuffix = resolved.testIdSuffix;

  return (
    <div className="inline-flex flex-col items-start gap-0.5">
      <Button
        type="button"
        variant="outline"
        size="xs"
        onClick={handleClick}
        disabled={disabled}
        title={resolved.tooltip}
        data-testid={`canvas-widget-block-action-${testIdSuffix}`}
        data-variant={action.variant}
        data-kind={action.kind}
      >
        {isFiring ? <Loader2 className="animate-spin" /> : Icon ? <Icon /> : null}
        {action.label}
      </Button>
      {error ? (
        <span
          className="max-w-[18rem] text-[11px] text-red-600"
          data-testid={`canvas-widget-block-action-error-${testIdSuffix}`}
        >
          {error}
        </span>
      ) : null}
      {action.confirm ? (
        <Dialog open={isConfirmOpen} onOpenChange={setIsConfirmOpen}>
          <DialogContent>
            <DialogHeader>
              <DialogTitle>{action.label}</DialogTitle>
              <DialogDescription>{evalTemplateExpr(action.confirm, row, env, stringifyCell)}</DialogDescription>
            </DialogHeader>
            <DialogFooter>
              <Button variant="outline" onClick={() => setIsConfirmOpen(false)}>
                Cancel
              </Button>
              <Button
                variant={action.variant === "danger" ? "destructive" : "default"}
                onClick={handleConfirmFire}
                data-testid={`canvas-widget-block-action-confirm-${testIdSuffix}`}
              >
                Confirm
              </Button>
            </DialogFooter>
          </DialogContent>
        </Dialog>
      ) : null}
    </div>
  );
}

function resolveAction(
  action: ActionSpec,
  row: Record<string, unknown>,
  nodeRefs: NodeChipContext | undefined,
  env: ExprEnv,
): ResolvedAction {
  if (action.kind === "trigger") {
    const nodeId = nodeRefs?.nodeIds?.[action.trigger];
    const onEmit = nodeRefs?.onEmitEvent;
    if (!nodeId) {
      return {
        testIdSuffix: action.trigger,
        visible: true,
        tooltip: `Trigger "${action.trigger}" not found on canvas`,
      };
    }
    if (!onEmit) {
      return { testIdSuffix: action.trigger, visible: true, tooltip: "Action not available in this view" };
    }
    return {
      testIdSuffix: action.trigger,
      visible: true,
      fire: () =>
        onEmit({ nodeSlug: action.trigger, channel: "default", data: buildFillPayload(action.fill, row, env) }),
    };
  }

  // Execution-scoped actions need a row with `executions`.
  const executions = (row.executions as CanvasesCanvasNodeExecutionRef[] | undefined) ?? [];
  const onAction = nodeRefs?.onExecutionAction;

  if (action.kind === "cancel") {
    const exec = findRunningExecution(executions);
    if (!exec || !exec.id || !exec.nodeId) return { testIdSuffix: "cancel", visible: false };
    if (!onAction) return { testIdSuffix: "cancel", visible: true, tooltip: "Action not available in this view" };
    const targetNodeId = exec.nodeId;
    const targetExecutionId = exec.id;
    return {
      testIdSuffix: "cancel",
      visible: true,
      fire: () => onAction({ kind: "cancel", nodeId: targetNodeId, executionId: targetExecutionId }),
    };
  }

  // approve / push-through
  const exec = findApprovalExecution(executions, action.node);
  if (!exec || !exec.id) return { testIdSuffix: `${action.kind}-${action.node}`, visible: false };
  if (!onAction) {
    return {
      testIdSuffix: `${action.kind}-${action.node}`,
      visible: true,
      tooltip: "Action not available in this view",
    };
  }
  const targetExecutionId = exec.id;
  const targetNodeId = action.node;
  return {
    testIdSuffix: `${action.kind}-${action.node}`,
    visible: true,
    fire: () => onAction({ kind: action.kind, nodeId: targetNodeId, executionId: targetExecutionId }),
  };
}

//
// Renderers
//

interface BaseRowsRenderProps {
  rows: Record<string, unknown>[];
  columns: ColumnSpec[];
  actions?: ActionSpec[];
  nodeRefs?: NodeChipContext;
  env: ExprEnv;
}

function TableRenderer({ rows, columns, actions, nodeRefs, env }: BaseRowsRenderProps) {
  const hasActions = !!actions && actions.length > 0;
  return (
    <div data-testid="canvas-widget-block" className="my-2 overflow-x-auto border-t border-slate-200 shadow-none">
      <table className="min-w-full border-collapse text-left text-xs">
        <thead>
          <tr>
            {columns.map((column, i) => (
              <th
                key={`${column.label}-${i}`}
                className="border-b border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-gray-600"
              >
                {column.label}
              </th>
            ))}
            {hasActions ? (
              <th className="border-b border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-gray-600">
                Actions
              </th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {rows.map((row, i) => (
            <tr key={(row.id as string | undefined) ?? i}>
              {columns.map((column, j) => (
                <td key={`${column.label}-${j}`} className="border-b border-slate-100 px-3 py-1.5 align-top">
                  {renderCell(evalRowFieldExpr(column.field, row, env, getByPath), column.format)}
                </td>
              ))}
              {hasActions ? (
                <td className="border-b border-slate-100 px-3 py-1.5 align-top">
                  <ActionButtons row={row} actions={actions!} nodeRefs={nodeRefs} env={env} />
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

//
// Chart palette
//

const CHART_PALETTE = [
  "var(--chart-1)",
  "var(--chart-2)",
  "var(--chart-3)",
  "var(--chart-4)",
  "var(--chart-5)",
] as const;

const COLOR_KEYWORD_TO_VAR: Record<ChartColorKeyword, string> = {
  // Mapped semantically against the calm chart palette in App.css
  // (sky primary / emerald / violet / amber / muted sky-gray). `blue` is
  // kept as an alias for the primary slot (sky-toned). `red` resolves to
  // `--chart-red` so it stays a true red even though the cycling palette
  // no longer includes one.
  blue: "var(--chart-1)",
  sky: "var(--chart-1)",
  green: "var(--chart-2)",
  yellow: "var(--chart-4)",
  gray: "var(--chart-5)",
  red: "var(--chart-red)",
};

const CHART_ANIMATION_MS = 300;
const CHART_HEIGHT = 240;
const SPARKLINE_HEIGHT = 40;

/** In-svg breathing room so axes / curves aren’t clipped in launchpad panels */
const WIDGET_CARTESIAN_MARGIN = { top: 12, right: 18, left: 16, bottom: 28 };
const WIDGET_PIE_MARGIN = { top: 12, right: 12, bottom: 12, left: 12 };
const WIDGET_SPARKLINE_MARGIN = { top: 8, right: 10, left: 6, bottom: 8 };

//
// Aggregate helper (reused by NumberRenderer + chart shapers).
//

function applyAggregate(values: number[], op: AggregateOp): number {
  if (op === "count") return values.length;
  if (values.length === 0) return NaN;
  if (op === "avg") return values.reduce((a, b) => a + b, 0) / values.length;
  if (op === "sum") return values.reduce((a, b) => a + b, 0);
  if (op === "min") return Math.min(...values);
  if (op === "max") return Math.max(...values);
  return NaN;
}

//
// Chart axis timestamps — short labels for ISO / epoch x-values; otherwise unchanged.
//

const ISO_DATE_ONLY_RE = /^\d{4}-\d{2}-\d{2}$/;

function tryParseChartTimestamp(raw: string): Date | null {
  const s = raw.trim();
  if (!s) return null;
  if (/^\d{10}$/.test(s)) {
    const d = new Date(Number(s) * 1000);
    return Number.isNaN(d.getTime()) ? null : d;
  }
  if (/^\d{12}$|^\d{13}$/.test(s)) {
    const d = new Date(Number(s));
    return Number.isNaN(d.getTime()) ? null : d;
  }
  const d = new Date(s);
  return Number.isNaN(d.getTime()) ? null : d;
}

/** Compact locale-aware labels, e.g. `May 7` or `May 7, 2:30 PM`. */
function formatChartAxisTick(raw: string): string {
  const trimmed = raw.trim();
  const d = tryParseChartTimestamp(trimmed);
  if (!d) return raw;

  const dateTimeCompact: Intl.DateTimeFormatOptions = {
    month: "short",
    day: "numeric",
    hour: "numeric",
    minute: "2-digit",
  };
  const dateCompact: Intl.DateTimeFormatOptions = { month: "short", day: "numeric" };

  if (/^\d{10}$/.test(trimmed) || /^\d{12}$|^\d{13}$/.test(trimmed)) {
    return new Intl.DateTimeFormat(undefined, dateTimeCompact).format(d);
  }
  if (ISO_DATE_ONLY_RE.test(trimmed)) {
    return new Intl.DateTimeFormat(undefined, dateCompact).format(d);
  }
  if (/T00:00:00(\.\d+)?(Z|[+-]00:00)?$/i.test(trimmed)) {
    return new Intl.DateTimeFormat(undefined, dateCompact).format(d);
  }
  return new Intl.DateTimeFormat(undefined, dateTimeCompact).format(d);
}

function widgetChartXTickFormatter(value: string | number): string {
  return formatChartAxisTick(String(value));
}

function WidgetChartTooltipContent(props: ComponentProps<typeof ChartTooltipContent>) {
  return <ChartTooltipContent {...props} labelFormatter={(lbl) => formatChartAxisTick(String(lbl ?? ""))} />;
}

//
// Data shapers — convert widget rows into the shape Recharts expects.
//

function coerceNumber(raw: unknown): number {
  if (typeof raw === "number") return raw;
  return Number(stringifyCell(raw));
}

/** Values are already in seconds (after ms→s conversion when applicable). */
function formatSecondsCompact(sec: number): string {
  if (!Number.isFinite(sec)) return "";
  const a = Math.abs(sec);
  if (a >= 100) return `${Math.round(sec)}s`;
  if (a >= 10) return `${sec.toFixed(1)}s`;
  return `${sec.toFixed(2)}s`;
}

function finalizeChartDisplayY(y: number, yUnit: ChartYUnit | undefined, aggregate?: AggregateOp): number {
  if (!Number.isFinite(y)) return y;
  if (yUnit === "ms" && aggregate !== "count") return y / 1000;
  return y;
}

function inferChartYUnitFromRows(
  chart: ChartSpec,
  rows: Record<string, unknown>[],
  env: ExprEnv,
): ChartYUnit | undefined {
  if (chart.yUnit !== undefined || !chart.y) return undefined;
  const lbl = chart.label?.toLowerCase() ?? "";
  if (!/\bduration\b/.test(lbl)) return undefined;
  let maxAbs = -Infinity;
  for (const row of rows) {
    const y = coerceNumber(evalRowFieldExpr(chart.y, row, env, getByPath));
    if (Number.isFinite(y)) maxAbs = Math.max(maxAbs, Math.abs(y));
  }
  if (maxAbs === -Infinity) return undefined;
  return maxAbs >= 2500 ? "ms" : undefined;
}

function resolveChartSpecForYAxis(chart: ChartSpec, rows: Record<string, unknown>[], env: ExprEnv): ChartSpec {
  const inferred = inferChartYUnitFromRows(chart, rows, env);
  const yUnit = chart.yUnit ?? inferred;
  return yUnit === chart.yUnit ? chart : { ...chart, yUnit };
}

function widgetChartYTickFormatter(chart: ChartSpec): (v: number | string) => string {
  return (v) => {
    const n = Number(v);
    if (!Number.isFinite(n)) return String(v);
    if (chart.yUnit === "ms" || chart.yUnit === "s") return formatSecondsCompact(n);
    return Number.isInteger(n) ? n.toLocaleString() : n.toLocaleString(undefined, { maximumFractionDigits: 3 });
  };
}

interface XYPoint {
  x: string;
  y: number;
}

function shapeXY(rows: Record<string, unknown>[], chart: ChartSpec, env: ExprEnv): XYPoint[] {
  const points: Array<[string, number]> = [];
  for (const row of rows) {
    const x = stringifyCell(evalRowFieldExpr(chart.x, row, env, getByPath));
    if (x === "") continue;
    const y = coerceNumber(evalRowFieldExpr(chart.y!, row, env, getByPath));
    if (!Number.isFinite(y)) continue;
    points.push([x, y]);
  }
  const { yUnit } = chart;
  if (!chart.aggregate) {
    return points.map(([x, y]) => ({ x, y: finalizeChartDisplayY(y, yUnit, undefined) }));
  }
  const groups = new Map<string, number[]>();
  for (const [x, y] of points) {
    const arr = groups.get(x) ?? [];
    arr.push(y);
    groups.set(x, arr);
  }
  const out: XYPoint[] = [];
  for (const [x, ys] of groups) {
    out.push({ x, y: finalizeChartDisplayY(applyAggregate(ys, chart.aggregate!), yUnit, chart.aggregate) });
  }
  return out;
}

interface StackedShape {
  data: Array<Record<string, string | number>>;
  groups: string[];
}

function shapeStacked(rows: Record<string, unknown>[], chart: ChartSpec, env: ExprEnv): StackedShape {
  const op: AggregateOp = chart.aggregate ?? "sum";
  // Map of x -> group -> values[]
  const buckets = new Map<string, Map<string, number[]>>();
  const xOrder: string[] = [];
  const groupOrder: string[] = [];
  for (const row of rows) {
    const x = stringifyCell(evalRowFieldExpr(chart.x, row, env, getByPath));
    if (x === "") continue;
    const g = stringifyCell(evalRowFieldExpr(chart.group!, row, env, getByPath));
    if (g === "") continue;
    let y = 0;
    if (op !== "count") {
      const yRaw = coerceNumber(evalRowFieldExpr(chart.y!, row, env, getByPath));
      if (!Number.isFinite(yRaw)) continue;
      y = yRaw;
    }
    if (!buckets.has(x)) {
      buckets.set(x, new Map());
      xOrder.push(x);
    }
    const inner = buckets.get(x)!;
    if (!inner.has(g)) {
      inner.set(g, []);
      if (!groupOrder.includes(g)) groupOrder.push(g);
    }
    inner.get(g)!.push(y);
  }
  const data: Array<Record<string, string | number>> = [];
  for (const x of xOrder) {
    const inner = buckets.get(x)!;
    const row: Record<string, string | number> = { x };
    for (const g of groupOrder) {
      const values = inner.get(g);
      row[g] = values ? finalizeChartDisplayY(applyAggregate(values, op), chart.yUnit, op) : 0;
    }
    data.push(row);
  }
  return { data, groups: groupOrder };
}

interface DonutSlice {
  name: string;
  value: number;
}

function shapeDonut(rows: Record<string, unknown>[], chart: ChartSpec, env: ExprEnv): DonutSlice[] {
  const op: AggregateOp = chart.aggregate ?? "count";
  const buckets = new Map<string, number[]>();
  const order: string[] = [];
  for (const row of rows) {
    const name = stringifyCell(evalRowFieldExpr(chart.x, row, env, getByPath));
    if (name === "") continue;
    let y = 0;
    if (op !== "count") {
      const yRaw = coerceNumber(evalRowFieldExpr(chart.y!, row, env, getByPath));
      if (!Number.isFinite(yRaw)) continue;
      y = yRaw;
    }
    if (!buckets.has(name)) {
      buckets.set(name, []);
      order.push(name);
    }
    buckets.get(name)!.push(y);
  }
  const out: DonutSlice[] = [];
  for (const name of order) {
    const values = buckets.get(name)!;
    const raw = applyAggregate(values, op);
    const value = finalizeChartDisplayY(raw, chart.yUnit, op);
    if (Number.isFinite(value)) out.push({ name, value });
  }
  return out;
}

function collectSparklineSeries(
  rows: Record<string, unknown>[],
  number: NumberSpec,
  env: ExprEnv,
): Array<{ i: number; y: number }> {
  const out: Array<{ i: number; y: number }> = [];
  let i = 0;
  for (const row of rows) {
    const raw = evalRowFieldExpr(number.field, row, env, getByPath);
    const y = typeof raw === "number" ? raw : Number(stringifyCell(raw));
    if (!Number.isFinite(y)) continue;
    out.push({ i: i++, y });
  }
  return out;
}

//
// Chart renderers
//

function ChartEmpty({ testId, fill }: { testId: string; fill?: boolean }) {
  return (
    <div
      data-testid={testId}
      className={
        fill
          ? "flex h-full w-full items-center justify-center rounded border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-xs text-slate-500"
          : "my-2 flex items-center justify-center rounded border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-xs text-slate-500"
      }
    >
      No data
    </div>
  );
}

function ChartRenderer({
  rows,
  chart,
  fill,
  env,
}: {
  rows: Record<string, unknown>[];
  chart: ChartSpec;
  fill?: boolean;
  env: ExprEnv;
}) {
  if (chart.type === "donut") return <DonutInner rows={rows} chart={chart} fill={fill} env={env} />;
  if (chart.type === "stacked-bar") return <StackedBarInner rows={rows} chart={chart} fill={fill} env={env} />;
  return <SingleSeriesInner rows={rows} chart={chart} fill={fill} env={env} />;
}

// In fill mode the wrapping div uses `h-full flex flex-col` so the chart
// container can flex-grow into the panel. Outside fill mode (markdown body
// with mixed content) the original `my-2` block-flow wrapper is preserved
// so chart blocks coexist with surrounding text.
const fillWrapClass = "flex h-full w-full flex-col";
const fillChartClass = "aspect-auto h-full w-full flex-1";

function SingleSeriesInner({
  rows,
  chart,
  fill,
  env,
}: {
  rows: Record<string, unknown>[];
  chart: ChartSpec;
  fill?: boolean;
  env: ExprEnv;
}) {
  const areaFillGradientId = useId().replace(/:/g, "");
  const resolvedChart = useMemo(() => resolveChartSpecForYAxis(chart, rows, env), [chart, rows, env]);
  const data = useMemo(() => shapeXY(rows, resolvedChart, env), [rows, resolvedChart, env]);
  const yTickFormatter = useMemo(() => widgetChartYTickFormatter(resolvedChart), [resolvedChart]);
  const colorVar = chart.color ? COLOR_KEYWORD_TO_VAR[chart.color] : "var(--chart-1)";
  const seriesKey = "y";
  const seriesLabel = chart.label ?? "Value";
  const config: ChartConfig = useMemo(
    () => ({ [seriesKey]: { label: seriesLabel, color: colorVar } }),
    [seriesLabel, colorVar],
  );

  if (data.length === 0) return <ChartEmpty testId="canvas-widget-block-chart-empty" fill={fill} />;

  return (
    <div data-testid="canvas-widget-block-chart" data-chart-type={chart.type} className={fill ? fillWrapClass : "my-2"}>
      <ChartContainer
        config={config}
        className={fill ? fillChartClass : "aspect-auto"}
        style={fill ? undefined : { height: CHART_HEIGHT }}
      >
        {chart.type === "bar" ? (
          <BarChart data={data} margin={WIDGET_CARTESIAN_MARGIN}>
            <CartesianGrid vertical={false} strokeDasharray="3 3" />
            <XAxis
              dataKey="x"
              tickLine={false}
              axisLine={false}
              interval="preserveStartEnd"
              tickFormatter={widgetChartXTickFormatter}
            />
            <YAxis tickLine={false} axisLine={false} width={44} tickFormatter={yTickFormatter} />
            <ChartTooltip content={<WidgetChartTooltipContent />} />
            <Bar
              dataKey={seriesKey}
              fill={colorVar}
              radius={[4, 4, 0, 0]}
              isAnimationActive
              animationDuration={CHART_ANIMATION_MS}
            />
          </BarChart>
        ) : chart.type === "line" ? (
          <LineChart data={data} margin={WIDGET_CARTESIAN_MARGIN}>
            <CartesianGrid vertical={false} strokeDasharray="3 3" />
            <XAxis
              dataKey="x"
              tickLine={false}
              axisLine={false}
              interval="preserveStartEnd"
              tickFormatter={widgetChartXTickFormatter}
            />
            <YAxis tickLine={false} axisLine={false} width={44} tickFormatter={yTickFormatter} />
            <ChartTooltip content={<WidgetChartTooltipContent />} />
            <Line
              dataKey={seriesKey}
              type="monotone"
              stroke={colorVar}
              strokeWidth={2}
              dot={data.length === 1}
              isAnimationActive
              animationDuration={CHART_ANIMATION_MS}
            />
          </LineChart>
        ) : (
          <AreaChart data={data} margin={WIDGET_CARTESIAN_MARGIN}>
            <defs>
              <linearGradient id={areaFillGradientId} x1="0" y1="0" x2="0" y2="1">
                <stop offset="0%" stopColor={colorVar} stopOpacity={0.42} />
                <stop offset="100%" stopColor={colorVar} stopOpacity={0} />
              </linearGradient>
            </defs>
            <CartesianGrid vertical={false} strokeDasharray="3 3" />
            <XAxis
              dataKey="x"
              tickLine={false}
              axisLine={false}
              interval="preserveStartEnd"
              tickFormatter={widgetChartXTickFormatter}
            />
            <YAxis tickLine={false} axisLine={false} width={44} tickFormatter={yTickFormatter} />
            <ChartTooltip content={<WidgetChartTooltipContent />} />
            <Area
              dataKey={seriesKey}
              type="monotone"
              stroke={colorVar}
              fill={`url(#${areaFillGradientId})`}
              fillOpacity={1}
              strokeWidth={2}
              isAnimationActive
              animationDuration={CHART_ANIMATION_MS}
            />
          </AreaChart>
        )}
      </ChartContainer>
    </div>
  );
}

function StackedBarInner({
  rows,
  chart,
  fill,
  env,
}: {
  rows: Record<string, unknown>[];
  chart: ChartSpec;
  fill?: boolean;
  env: ExprEnv;
}) {
  const resolvedChart = useMemo(() => resolveChartSpecForYAxis(chart, rows, env), [chart, rows, env]);
  const { data, groups } = useMemo(() => shapeStacked(rows, resolvedChart, env), [rows, resolvedChart, env]);
  const yTickFormatter = useMemo(() => widgetChartYTickFormatter(resolvedChart), [resolvedChart]);
  const config: ChartConfig = useMemo(() => {
    const cfg: ChartConfig = {};
    groups.forEach((g, i) => {
      cfg[g] = { label: g, color: CHART_PALETTE[i % CHART_PALETTE.length] };
    });
    return cfg;
  }, [groups]);

  if (data.length === 0 || groups.length === 0) {
    return <ChartEmpty testId="canvas-widget-block-chart-empty" fill={fill} />;
  }

  return (
    <div
      data-testid="canvas-widget-block-chart"
      data-chart-type="stacked-bar"
      className={fill ? fillWrapClass : "my-2"}
    >
      <ChartContainer
        config={config}
        className={fill ? fillChartClass : "aspect-auto"}
        style={fill ? undefined : { height: CHART_HEIGHT }}
      >
        <BarChart data={data} margin={WIDGET_CARTESIAN_MARGIN}>
          <CartesianGrid vertical={false} strokeDasharray="3 3" />
          <XAxis
            dataKey="x"
            tickLine={false}
            axisLine={false}
            interval="preserveStartEnd"
            tickFormatter={widgetChartXTickFormatter}
          />
          <YAxis tickLine={false} axisLine={false} width={44} tickFormatter={yTickFormatter} />
          <ChartTooltip content={<WidgetChartTooltipContent />} />
          {groups.map((g, i) => (
            <Bar
              key={g}
              dataKey={g}
              stackId="a"
              fill={CHART_PALETTE[i % CHART_PALETTE.length]}
              radius={i === groups.length - 1 ? [4, 4, 0, 0] : [0, 0, 0, 0]}
              isAnimationActive
              animationDuration={CHART_ANIMATION_MS}
            />
          ))}
        </BarChart>
      </ChartContainer>
    </div>
  );
}

function DonutInner({
  rows,
  chart,
  fill,
  env,
}: {
  rows: Record<string, unknown>[];
  chart: ChartSpec;
  fill?: boolean;
  env: ExprEnv;
}) {
  const resolvedChart = useMemo(() => resolveChartSpecForYAxis(chart, rows, env), [chart, rows, env]);
  const data = useMemo(() => shapeDonut(rows, resolvedChart, env), [rows, resolvedChart, env]);
  const config: ChartConfig = useMemo(() => {
    const cfg: ChartConfig = {};
    data.forEach((slice, i) => {
      cfg[slice.name] = { label: slice.name, color: CHART_PALETTE[i % CHART_PALETTE.length] };
    });
    return cfg;
  }, [data]);

  if (data.length === 0) return <ChartEmpty testId="canvas-widget-block-chart-empty" fill={fill} />;

  return (
    <div data-testid="canvas-widget-block-chart" data-chart-type="donut" className={fill ? fillWrapClass : "my-2"}>
      <ChartContainer
        config={config}
        className={fill ? fillChartClass : "aspect-auto"}
        style={fill ? undefined : { height: CHART_HEIGHT }}
      >
        <PieChart margin={WIDGET_PIE_MARGIN}>
          <ChartTooltip content={<WidgetChartTooltipContent nameKey="name" />} />
          <Pie
            data={data}
            dataKey="value"
            nameKey="name"
            innerRadius={50}
            outerRadius={80}
            paddingAngle={2}
            isAnimationActive
            animationDuration={CHART_ANIMATION_MS}
          >
            {data.map((slice, i) => (
              <Cell key={slice.name} fill={CHART_PALETTE[i % CHART_PALETTE.length]} />
            ))}
          </Pie>
        </PieChart>
      </ChartContainer>
    </div>
  );
}

//
// Number renderer (with optional sparkline).
//

function NumberRenderer({
  rows,
  number,
  fill,
  env,
}: {
  rows: Record<string, unknown>[];
  number: NumberSpec;
  fill?: boolean;
  env: ExprEnv;
}) {
  const value = useMemo(() => computeNumberAggregate(rows, number, env), [rows, number, env]);
  const display = formatNumberValue(value, number.format);
  const sparkPoints = useMemo(
    () => (number.sparkline ? collectSparklineSeries(rows, number, env) : null),
    [rows, number, env],
  );
  // In fill mode the block stretches to the panel height; content aligns to the
  // top with a large headline and optional sparkline below the label.
  return (
    <div
      data-testid="canvas-widget-block-number"
      className={
        fill ? "flex h-full w-full flex-col items-start justify-start gap-2" : "my-2 flex flex-col items-start gap-1"
      }
    >
      <span
        className={
          fill ? "text-7xl font-normal tracking-tight text-slate-800" : "text-4xl font-semibold text-slate-800"
        }
      >
        {display}
      </span>
      <span className="text-sm font-medium text-slate-500">{number.label}</span>
      {sparkPoints && sparkPoints.length > 1 ? <SparklineInner points={sparkPoints} fill={fill} /> : null}
    </div>
  );
}

const SPARKLINE_CONFIG: ChartConfig = {
  y: { label: "value", color: "var(--chart-1)" },
};

function SparklineInner({ points, fill }: { points: Array<{ i: number; y: number }>; fill?: boolean }) {
  const sparkFillGradientId = useId().replace(/:/g, "");
  // In fill mode the sparkline gets a slightly larger fixed height since it
  // sits inside a much taller card; otherwise keep the compact 40px sparkline
  // used inline with markdown content.
  const height = fill ? 76 : SPARKLINE_HEIGHT;
  const strokeColor = "var(--chart-1)";
  return (
    <div data-testid="canvas-widget-block-number-sparkline" className={fill ? "mt-3 w-full" : "mt-2 w-full"}>
      <ChartContainer config={SPARKLINE_CONFIG} className="aspect-auto" style={{ height }}>
        <AreaChart data={points} margin={WIDGET_SPARKLINE_MARGIN}>
          <defs>
            <linearGradient id={sparkFillGradientId} x1="0" y1="0" x2="0" y2="1">
              <stop offset="0%" stopColor={strokeColor} stopOpacity={0.4} />
              <stop offset="100%" stopColor={strokeColor} stopOpacity={0} />
            </linearGradient>
          </defs>
          <Area
            dataKey="y"
            type="monotone"
            stroke={strokeColor}
            fill={`url(#${sparkFillGradientId})`}
            fillOpacity={1}
            strokeWidth={1.5}
            isAnimationActive={false}
          />
        </AreaChart>
      </ChartContainer>
    </div>
  );
}

function computeNumberAggregate(rows: Record<string, unknown>[], number: NumberSpec, env: ExprEnv): number {
  if (number.aggregate === "count") return rows.length;
  const values: number[] = [];
  for (const row of rows) {
    const raw = evalRowFieldExpr(number.field, row, env, getByPath);
    const n = typeof raw === "number" ? raw : Number(stringifyCell(raw));
    if (Number.isFinite(n)) values.push(n);
  }
  return applyAggregate(values, number.aggregate);
}

function formatNumberValue(value: number, format: NumberSpec["format"]): string {
  if (!Number.isFinite(value)) return "—";
  if (format === "duration") return formatDurationSeconds(value);
  if (format === "percent") return `${(value * 100).toFixed(1)}%`;
  return value.toLocaleString();
}

function formatDurationSeconds(value: number): string {
  const s = Math.round(value);
  if (s < 60) return `${s}s`;
  const minutes = Math.floor(s / 60);
  const remSeconds = s % 60;
  if (minutes < 60) return remSeconds === 0 ? `${minutes}m` : `${minutes}m ${remSeconds}s`;
  const hours = Math.floor(minutes / 60);
  const remMinutes = minutes % 60;
  return remMinutes === 0 ? `${hours}h` : `${hours}h ${remMinutes}m`;
}

//
// Component
//

export function WidgetBlock({ body, canvasId, nodeRefs, fill }: WidgetBlockProps) {
  const parsed = useMemo(() => parseWidgetBody(body), [body]);
  const isOk = parsed.kind === "ok";
  const widget = isOk ? parsed.widget : null;
  // `now` is computed once per render so every CEL expression in this render
  // sees a consistent view of the clock. Custom function bindings live on
  // `env.functions`; they don't change per render but are still recreated so
  // tests can inject mocks via `buildEnv` overrides if needed in the future.
  const env = useMemo<ExprEnv>(() => buildEnv(), [body]); // eslint-disable-line react-hooks/exhaustive-deps

  const memoryEnabled = isOk && widget!.source === "memory";
  const memoryQuery = useCanvasMemoryEntries(canvasId, memoryEnabled);

  const executionsEnabled = isOk && widget!.source === "executions";
  const executionsQuery = useWidgetExecutionRows(canvasId, {
    trigger: executionsEnabled && widget!.source === "executions" ? widget!.trigger : undefined,
    status: executionsEnabled && widget!.source === "executions" ? widget!.status : undefined,
    limit: executionsEnabled && widget!.source === "executions" ? widget!.limit : DEFAULT_LIMIT,
    enabled: executionsEnabled,
  });

  if (parsed.kind === "error") {
    return <WidgetBlockError message={parsed.error.message} body={body} />;
  }

  const w = parsed.widget;

  const isLoading = w.source === "memory" ? memoryQuery.isLoading : executionsQuery.isLoading;
  if (isLoading) return <WidgetBlockSkeleton />;

  const isError = w.source === "memory" ? memoryQuery.isError : executionsQuery.isError;
  if (isError) {
    const err = w.source === "memory" ? memoryQuery.error : executionsQuery.error;
    const message = err instanceof Error ? err.message : "Unknown error";
    const prefix = w.source === "memory" ? "Failed to load memory" : "Failed to load runs";
    return <WidgetBlockError message={`${prefix}: ${message}`} body={body} />;
  }

  // Build the (filtered, ordered) rows with a uniform `Record<string, unknown>` shape.
  const baseRows: Record<string, unknown>[] =
    w.source === "memory"
      ? (memoryQuery.data ?? [])
          .filter((entry) => entry.namespace === w.namespace)
          .map<Record<string, unknown>>((entry) => ({
            id: entry.id,
            namespace: entry.namespace,
            // For memory rows, hoist `values.*` to the row root so existing
            // simple-key columns (`field: pr_number`) keep working unchanged.
            ...(isPlainObject(entry.values) ? entry.values : {}),
          }))
      : executionsQuery.rows;

  const filtered = w.where ? applyWhereOnRows(baseRows, w.where, env) : baseRows;

  if (filtered.length === 0 && w.render.kind === "table") {
    const emptyLabel = w.source === "memory" ? `"${w.namespace}"` : "this widget";
    return (
      <div
        data-testid="canvas-widget-block-empty"
        className="my-2 flex items-center justify-center rounded border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-xs text-slate-500"
      >
        {w.source === "memory" ? <>No entries in {emptyLabel}</> : <>No runs found</>}
      </div>
    );
  }

  if (w.render.kind === "chart") {
    return <ChartRenderer rows={filtered} chart={w.render.chart} fill={fill} env={env} />;
  }
  if (w.render.kind === "number") {
    return <NumberRenderer rows={filtered} number={w.render.number} fill={fill} env={env} />;
  }

  const columns: ColumnSpec[] = w.columns ?? autoColumns(w.source, filtered);

  return <TableRenderer rows={filtered} columns={columns} actions={w.actions} nodeRefs={nodeRefs} env={env} />;
}

function autoColumns(source: "memory" | "executions", rows: Record<string, unknown>[]): ColumnSpec[] {
  if (source === "memory") {
    // Auto-derived from the union of memory `values` keys (excluding the row meta keys we hoist).
    const set = new Set<string>();
    for (const row of rows) {
      for (const key of Object.keys(row)) {
        if (key === "id" || key === "namespace") continue;
        set.add(key);
      }
    }
    return Array.from(set)
      .sort((a, b) => a.localeCompare(b))
      .map<ColumnSpec>((field) => ({ label: field, field: { kind: "literal", value: field }, format: "plain" }));
  }
  // Executions auto-columns: a sensible default trio.
  return [
    { label: "Run", field: { kind: "literal", value: "root.id" }, format: "plain" },
    { label: "Status", field: { kind: "literal", value: "status" }, format: "badge" },
    { label: "Started", field: { kind: "literal", value: "root.createdAt" }, format: "relative" },
  ];
}

function WidgetBlockSkeleton() {
  return (
    <div
      data-testid="canvas-widget-block-skeleton"
      className="my-2 overflow-hidden rounded border border-slate-200"
      aria-busy="true"
      aria-live="polite"
    >
      <table className="min-w-full border-collapse text-left text-xs">
        <thead>
          <tr>
            {[0, 1, 2].map((i) => (
              <th key={i} className="border-b border-slate-200 bg-slate-50 px-3 py-1.5">
                <span className="block h-3 w-20 animate-pulse rounded bg-slate-200" />
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {[0, 1, 2].map((row) => (
            <tr key={row}>
              {[0, 1, 2].map((col) => (
                <td key={col} className="border-b border-slate-100 px-3 py-1.5">
                  <span className="block h-3 w-24 animate-pulse rounded bg-slate-100" />
                </td>
              ))}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function WidgetBlockError({ message, body }: { message: string; body: string }) {
  return (
    <div
      data-testid="canvas-widget-block-error"
      className="my-2 rounded-md border border-red-200 bg-red-50 p-3 text-xs text-red-700"
    >
      <div className="mb-1 flex items-center gap-1.5 font-semibold">
        <AlertTriangle className="h-3.5 w-3.5" />
        {message}
      </div>
      <details className="mt-2">
        <summary className="cursor-pointer text-[11px] text-red-500">Show source</summary>
        <pre className="mt-1 overflow-x-auto rounded bg-white p-2 text-[11px] text-gray-700">{body}</pre>
      </details>
    </div>
  );
}

// `CanvasMemoryEntry` re-export removed — only used internally now.
export type { CanvasMemoryEntry };
