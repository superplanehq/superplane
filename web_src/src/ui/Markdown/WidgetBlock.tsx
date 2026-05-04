import { useMemo, useState } from "react";
import * as yaml from "js-yaml";
import { AlertTriangle, ExternalLink, Loader2, Play, RefreshCw, Square, Trash2 } from "lucide-react";

import { useCanvasMemoryEntries, useInfiniteCanvasEvents, type CanvasMemoryEntry } from "@/hooks/useCanvasData";
import type { CanvasesCanvasEventWithExecutions, CanvasesCanvasNodeExecutionRef } from "@/api-client";
import { getAggregateRunStatus } from "@/pages/workflowv2/lib/canvas-runs";
import { formatRelativeTime } from "@/lib/timezone";
import { showSuccessToast } from "@/lib/toast";
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
import { MermaidDiagram } from "./MermaidDiagram";

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
//     - label: Open
//       kind: trigger
//       trigger: my-trigger
//   render:                   # optional — table | chart | number; defaults to table
//     kind: chart
//     chart: { type: bar, x: pr_number, y: created_at }
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
}

//
// Column / format types (unchanged from increments 2/3).
//

type Format = "plain" | "link" | { kind: "linkLabel"; label: string } | "relative" | "date" | "badge" | "code";

interface ColumnSpec {
  label: string;
  field: string;
  format: Format;
}

//
// Filter types (unchanged from increment 3).
//

const FILTER_OPS = ["eq", "neq", "contains", "not_contains", "gt", "lt", "exists", "not_exists"] as const;
type FilterOp = (typeof FILTER_OPS)[number];

interface FilterCondition {
  field: string;
  op: FilterOp;
  value: string;
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
  confirm?: string;
  show?: string;
}

interface TriggerActionSpec extends ActionBase {
  kind: "trigger";
  trigger: string;
  template?: string;
  fill?: Record<string, string>;
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

interface ChartSpec {
  type: "bar" | "line";
  x: string;
  y: string;
  label?: string;
  aggregate?: AggregateOp;
}

interface NumberSpec {
  field: string;
  aggregate: AggregateOp;
  label: string;
  format: "number" | "duration" | "percent";
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
    result.push({ label, field, format: parseFormat(formatRaw) });
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
      field,
      op,
      value: opNeedsValue ? String(value) : "",
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
      if ((ACTION_VARIANTS as readonly string[]).includes(variantRaw)) {
        variant = variantRaw as ActionVariant;
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
      confirm: typeof confirm === "string" ? confirm : undefined,
      show: typeof showRaw === "string" ? showRaw : undefined,
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
      let parsedFill: Record<string, string> | undefined;
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
          parsedFill[path] = value;
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
    if (chart.type !== "bar" && chart.type !== "line") {
      return { kind: "error", error: { message: 'Invalid widget block: `render.chart.type` must be "bar" or "line"' } };
    }
    if (typeof chart.x !== "string" || chart.x.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: `render.chart.x` is required" } };
    }
    if (typeof chart.y !== "string" || chart.y.trim() === "") {
      return { kind: "error", error: { message: "Invalid widget block: `render.chart.y` is required" } };
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
    return {
      kind: "ok",
      value: {
        kind: "chart",
        chart: {
          type: chart.type,
          x: chart.x,
          y: chart.y,
          label: typeof chart.label === "string" ? chart.label : undefined,
          aggregate,
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
    return {
      kind: "ok",
      value: {
        kind: "number",
        number: { field: num.field, aggregate: num.aggregate as AggregateOp, label: num.label, format },
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

function evalShow(condition: string | undefined, row: Record<string, unknown>): boolean {
  if (!condition) return true;
  const m = condition.match(SHOW_RE);
  if (!m) return false; // fail-closed: malformed expression hides the button
  const [, path, op, expected] = m;
  const actual = stringifyCell(getByPath(row, path));
  return op === "==" ? actual === expected : actual !== expected;
}

function interpolate(template: string, values: Record<string, unknown>): string {
  return template.replace(/\{\{([\w.-]+)\}\}/g, (_, key) => stringifyCell(getByPath(values, key)));
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
  fill: Record<string, string> | undefined,
  row: Record<string, unknown>,
): Record<string, unknown> {
  const payload: Record<string, unknown> = {};
  if (!fill) return payload;
  for (const [path, template] of Object.entries(fill)) {
    setPath(payload, path, interpolate(template, row));
  }
  return payload;
}

//
// Filter engine
//

function evalCondition(values: Record<string, unknown>, cond: FilterCondition): boolean {
  // Allow filters to use dot-paths so executions widgets can filter on nested fields too.
  const raw = getByPath(values, cond.field);
  const has = raw !== undefined;
  const val = raw == null ? "" : typeof raw === "string" ? raw : stringifyCell(raw);

  if (cond.op === "exists") return val !== "";
  if (cond.op === "not_exists") return val === "";

  if (!has) return false;

  switch (cond.op) {
    case "eq":
      return val === cond.value;
    case "neq":
      return val !== cond.value;
    case "contains":
      return val.includes(cond.value);
    case "not_contains":
      return !val.includes(cond.value);
    case "gt":
    case "lt": {
      const a = parseFloat(val);
      const b = parseFloat(cond.value);
      if (Number.isNaN(a) || Number.isNaN(b)) return false;
      return cond.op === "gt" ? a > b : a < b;
    }
  }
}

function applyWhereOnRows<T extends Record<string, unknown>>(rows: T[], where: FilterCondition[]): T[] {
  return rows.filter((row) => where.every((cond) => evalCondition(row, cond)));
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
  const aggregate = getAggregateRunStatus(executions, queueItems.length > 0);
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
  running: "bg-blue-100 text-blue-700 border-blue-200",
  passed: "bg-emerald-100 text-emerald-700 border-emerald-200",
  failed: "bg-red-100 text-red-700 border-red-200",
  cancelled: "bg-slate-100 text-slate-600 border-slate-200",
};

const BADGE_PILL_BASE = "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none";
const DEFAULT_BADGE_PILL = `${BADGE_PILL_BASE} bg-emerald-100 text-emerald-700 border-emerald-200`;

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
  return (
    <a
      href={href}
      target="_blank"
      rel="noopener noreferrer"
      className="inline-flex items-center gap-0.5 text-blue-600 underline"
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
// Action button
//

const VARIANT_CLASSES: Record<ActionVariant, string> = {
  default: "border-slate-300 bg-white text-slate-700 hover:bg-slate-50",
  danger: "border-red-300 bg-red-50 text-red-700 hover:bg-red-100",
  primary: "border-blue-300 bg-blue-50 text-blue-700 hover:bg-blue-100",
};

const ICON_MAP: Record<ActionIcon, React.ComponentType<{ className?: string }>> = {
  trash: Trash2,
  play: Play,
  refresh: RefreshCw,
  stop: Square,
  "external-link": ExternalLink,
};

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
}: {
  row: Record<string, unknown>;
  actions: ActionSpec[];
  nodeRefs?: NodeChipContext;
}) {
  return (
    <div className="flex flex-wrap items-center gap-1">
      {actions.map((action, i) => (
        <ActionButton key={`${action.kind}-${actionId(action)}-${i}`} row={row} action={action} nodeRefs={nodeRefs} />
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
}: {
  row: Record<string, unknown>;
  action: ActionSpec;
  nodeRefs?: NodeChipContext;
}) {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isFiring, setIsFiring] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const passesShow = evalShow(action.show, row);
  const resolved = resolveAction(action, row, nodeRefs);

  if (!passesShow || !resolved.visible) return null;

  const Icon = action.icon ? ICON_MAP[action.icon] : null;
  const baseClass =
    "inline-flex items-center gap-1 rounded border px-2 py-1 text-xs font-medium transition disabled:cursor-not-allowed disabled:opacity-60";
  const variantClass = VARIANT_CLASSES[action.variant];
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
      <button
        type="button"
        onClick={handleClick}
        disabled={disabled}
        title={resolved.tooltip}
        data-testid={`canvas-widget-block-action-${testIdSuffix}`}
        data-variant={action.variant}
        data-kind={action.kind}
        className={`${baseClass} ${variantClass}`}
      >
        {isFiring ? <Loader2 className="h-3 w-3 animate-spin" /> : Icon ? <Icon className="h-3 w-3" /> : null}
        <span>{action.label}</span>
      </button>
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
              <DialogDescription>{interpolate(action.confirm, row)}</DialogDescription>
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

function resolveAction(action: ActionSpec, row: Record<string, unknown>, nodeRefs?: NodeChipContext): ResolvedAction {
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
      fire: () => onEmit({ nodeSlug: action.trigger, channel: "default", data: buildFillPayload(action.fill, row) }),
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
}

function TableRenderer({ rows, columns, actions, nodeRefs }: BaseRowsRenderProps) {
  const hasActions = !!actions && actions.length > 0;
  return (
    <div data-testid="canvas-widget-block" className="my-2 overflow-x-auto rounded border border-slate-200">
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
                <td key={`${column.field}-${j}`} className="border-b border-slate-100 px-3 py-1.5 align-top">
                  {renderCell(getByPath(row, column.field), column.format)}
                </td>
              ))}
              {hasActions ? (
                <td className="border-b border-slate-100 px-3 py-1.5 align-top">
                  <ActionButtons row={row} actions={actions!} nodeRefs={nodeRefs} />
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

function ChartRenderer({ rows, chart }: { rows: Record<string, unknown>[]; chart: ChartSpec }) {
  const code = useMemo(() => buildXyChartSource(rows, chart), [rows, chart]);
  return (
    <div data-testid="canvas-widget-block-chart">
      <MermaidDiagram code={code} />
    </div>
  );
}

function aggregatePoints(points: Array<[string, number]>, op: AggregateOp): Array<[string, number]> {
  const groups = new Map<string, number[]>();
  for (const [x, y] of points) {
    const arr = groups.get(x) ?? [];
    arr.push(y);
    groups.set(x, arr);
  }
  const out: Array<[string, number]> = [];
  for (const [x, ys] of groups) {
    out.push([x, applyAggregate(ys, op)]);
  }
  return out;
}

function applyAggregate(values: number[], op: AggregateOp): number {
  if (op === "count") return values.length;
  if (values.length === 0) return NaN;
  if (op === "avg") return values.reduce((a, b) => a + b, 0) / values.length;
  if (op === "sum") return values.reduce((a, b) => a + b, 0);
  if (op === "min") return Math.min(...values);
  if (op === "max") return Math.max(...values);
  return NaN;
}

function buildXyChartSource(rows: Record<string, unknown>[], chart: ChartSpec): string {
  const points: Array<[string, number]> = [];
  for (const row of rows) {
    const xRaw = stringifyCell(getByPath(row, chart.x));
    if (xRaw === "") continue;
    const yRaw = getByPath(row, chart.y);
    const y = typeof yRaw === "number" ? yRaw : Number(stringifyCell(yRaw));
    if (!Number.isFinite(y)) continue;
    points.push([xRaw, y]);
  }
  const grouped = chart.aggregate ? aggregatePoints(points, chart.aggregate) : points;
  const xLabels = grouped.map(([x]) => `"${escapeMermaidString(x)}"`).join(", ");
  const yValues = grouped.map(([, y]) => formatChartNumber(y)).join(", ");
  const series = chart.type === "line" ? `line [${yValues}]` : `bar [${yValues}]`;
  const titleLine = chart.label ? `  title "${escapeMermaidString(chart.label)}"` : undefined;
  const yAxisLine = chart.label ? `  y-axis "${escapeMermaidString(chart.label)}"` : undefined;
  const lines = ["xychart-beta", titleLine, `  x-axis [${xLabels}]`, yAxisLine, `  ${series}`].filter(Boolean);
  return lines.join("\n");
}

function escapeMermaidString(value: string): string {
  return value.replace(/"/g, "'");
}

function formatChartNumber(value: number): string {
  if (!Number.isFinite(value)) return "0";
  return Number.isInteger(value) ? String(value) : value.toFixed(2);
}

function NumberRenderer({ rows, number }: { rows: Record<string, unknown>[]; number: NumberSpec }) {
  const value = useMemo(() => computeNumberAggregate(rows, number), [rows, number]);
  const display = formatNumberValue(value, number.format);
  return (
    <div
      data-testid="canvas-widget-block-number"
      className="my-2 flex flex-col items-start gap-1 rounded border border-slate-200 bg-white px-4 py-3"
    >
      <span className="text-3xl font-semibold text-slate-800">{display}</span>
      <span className="text-xs uppercase tracking-wide text-slate-500">{number.label}</span>
    </div>
  );
}

function computeNumberAggregate(rows: Record<string, unknown>[], number: NumberSpec): number {
  if (number.aggregate === "count") return rows.length;
  const values: number[] = [];
  for (const row of rows) {
    const raw = getByPath(row, number.field);
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

export function WidgetBlock({ body, canvasId, nodeRefs }: WidgetBlockProps) {
  const parsed = useMemo(() => parseWidgetBody(body), [body]);
  const isOk = parsed.kind === "ok";
  const widget = isOk ? parsed.widget : null;

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

  const filtered = w.where ? applyWhereOnRows(baseRows, w.where) : baseRows;

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
    return <ChartRenderer rows={filtered} chart={w.render.chart} />;
  }
  if (w.render.kind === "number") {
    return <NumberRenderer rows={filtered} number={w.render.number} />;
  }

  const columns: ColumnSpec[] = w.columns ?? autoColumns(w.source, filtered);

  return <TableRenderer rows={filtered} columns={columns} actions={w.actions} nodeRefs={nodeRefs} />;
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
      .map<ColumnSpec>((field) => ({ label: field, field, format: "plain" }));
  }
  // Executions auto-columns: a sensible default trio.
  return [
    { label: "Run", field: "root.id", format: "plain" },
    { label: "Status", field: "status", format: "badge" },
    { label: "Started", field: "root.createdAt", format: "relative" },
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
