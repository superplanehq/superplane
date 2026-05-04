import { useMemo } from "react";
import * as yaml from "js-yaml";
import { AlertTriangle, ExternalLink } from "lucide-react";

import { useCanvasMemoryEntries, type CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { formatRelativeTime } from "@/lib/timezone";

//
// Authored as a fenced code block inside Launchpad markdown panels:
//
//   ```query
//   source: memory
//   namespace: environments
//   columns:                # optional — auto-detected from values keys when omitted
//     - label: PR
//       field: pr_number
//     - label: URL
//       field: url
//       format: link:Open
//   where:                  # optional — all conditions ANDed
//     - field: url
//       op: exists
//   ```
//
// v2 supports `source: memory` only. Render path:
//   parsed -> namespace filter -> where filter -> columns (explicit or auto)
//

export interface QueryBlockProps {
  body: string;
  canvasId: string;
}

type Format = "plain" | "link" | { kind: "linkLabel"; label: string } | "relative" | "date" | "badge" | "code";

interface ColumnSpec {
  label: string;
  field: string;
  format: Format;
}

const FILTER_OPS = ["eq", "neq", "contains", "not_contains", "gt", "lt", "exists", "not_exists"] as const;
type FilterOp = (typeof FILTER_OPS)[number];

interface FilterCondition {
  field: string;
  op: FilterOp;
  value: string;
}

interface ParsedQuery {
  source: "memory";
  namespace: string;
  columns?: ColumnSpec[];
  where?: FilterCondition[];
}

interface QueryParseError {
  message: string;
}

type ParseResult = { kind: "ok"; query: ParsedQuery } | { kind: "error"; error: QueryParseError };

function parseQueryBody(body: string): ParseResult {
  let parsed: unknown;
  try {
    parsed = yaml.load(body);
  } catch (err) {
    const message = err instanceof Error ? err.message : String(err);
    return { kind: "error", error: { message: `Invalid query block: ${message}` } };
  }

  if (parsed == null || typeof parsed !== "object" || Array.isArray(parsed)) {
    return {
      kind: "error",
      error: { message: "Invalid query block: expected an object with `source` and `namespace`" },
    };
  }

  const obj = parsed as Record<string, unknown>;
  const source = obj.source;
  const namespace = obj.namespace;

  if (typeof source !== "string" || source.trim() === "") {
    return { kind: "error", error: { message: "Invalid query block: missing `source`" } };
  }
  if (source !== "memory") {
    return {
      kind: "error",
      error: { message: `Invalid query block: unsupported source "${source}" (only "memory" is supported)` },
    };
  }
  if (typeof namespace !== "string" || namespace.trim() === "") {
    return { kind: "error", error: { message: "Invalid query block: missing `namespace`" } };
  }

  const columnsResult = parseColumns(obj.columns);
  if (columnsResult.kind === "error") return columnsResult;

  const whereResult = parseWhere(obj.where);
  if (whereResult.kind === "error") return whereResult;

  return {
    kind: "ok",
    query: {
      source: "memory",
      namespace,
      columns: columnsResult.value,
      where: whereResult.value,
    },
  };
}

type ParsePartResult<T> = { kind: "ok"; value: T | undefined } | { kind: "error"; error: QueryParseError };

function parseColumns(raw: unknown): ParsePartResult<ColumnSpec[]> {
  if (raw === undefined || raw === null) return { kind: "ok", value: undefined };
  if (!Array.isArray(raw)) {
    return { kind: "error", error: { message: "Invalid query block: `columns` must be a list" } };
  }
  if (raw.length === 0) {
    return { kind: "error", error: { message: "Invalid query block: `columns` must not be empty" } };
  }

  const result: ColumnSpec[] = [];
  for (let i = 0; i < raw.length; i++) {
    const item = raw[i];
    if (!isPlainObject(item)) {
      return { kind: "error", error: { message: `Invalid query block: columns[${i}] must be an object` } };
    }
    const label = item.label;
    const field = item.field;
    const formatRaw = item.format;
    if (typeof label !== "string" || label.trim() === "") {
      return { kind: "error", error: { message: `Invalid query block: columns[${i}] missing \`label\`` } };
    }
    if (typeof field !== "string" || field.trim() === "") {
      return { kind: "error", error: { message: `Invalid query block: columns[${i}] missing \`field\`` } };
    }
    if (formatRaw !== undefined && typeof formatRaw !== "string") {
      return {
        kind: "error",
        error: { message: `Invalid query block: columns[${i}].format must be a string` },
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
  // Per spec: unknown format does NOT fail the block. Warn once and fall back to plain.
  // eslint-disable-next-line no-console
  console.warn(`[QueryBlock] Unknown format "${raw}", falling back to plain text`);
  return "plain";
}

function parseWhere(raw: unknown): ParsePartResult<FilterCondition[]> {
  if (raw === undefined || raw === null) return { kind: "ok", value: undefined };
  if (!Array.isArray(raw)) {
    return { kind: "error", error: { message: "Invalid query block: `where` must be a list" } };
  }
  if (raw.length === 0) {
    return { kind: "error", error: { message: "Invalid query block: `where` must not be empty" } };
  }

  const result: FilterCondition[] = [];
  for (let i = 0; i < raw.length; i++) {
    const item = raw[i];
    if (!isPlainObject(item)) {
      return { kind: "error", error: { message: `Invalid query block: where[${i}] must be an object` } };
    }
    const field = item.field;
    const op = item.op;
    const value = item.value;
    if (typeof field !== "string" || field.trim() === "") {
      return { kind: "error", error: { message: `Invalid query block: where[${i}] missing \`field\`` } };
    }
    if (typeof op !== "string" || op.trim() === "") {
      return { kind: "error", error: { message: `Invalid query block: where[${i}] missing \`op\`` } };
    }
    if (!isFilterOp(op)) {
      return { kind: "error", error: { message: `Unknown filter operator: "${op}"` } };
    }
    const opNeedsValue = op !== "exists" && op !== "not_exists";
    if (opNeedsValue) {
      if (value === undefined || value === null) {
        return { kind: "error", error: { message: `Invalid query block: where[${i}] missing \`value\`` } };
      }
      if (typeof value !== "string" && typeof value !== "number" && typeof value !== "boolean") {
        return {
          kind: "error",
          error: { message: `Invalid query block: where[${i}].value must be a scalar` },
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

//
// Filter engine
//

function evalCondition(values: Record<string, unknown>, cond: FilterCondition): boolean {
  const has = Object.prototype.hasOwnProperty.call(values, cond.field);
  const raw = has ? values[cond.field] : undefined;
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

function applyWhere(entries: CanvasMemoryEntry[], where: FilterCondition[]): CanvasMemoryEntry[] {
  return entries.filter((entry) => {
    const values = isPlainObject(entry.values) ? entry.values : {};
    return where.every((cond) => evalCondition(values, cond));
  });
}

//
// Cell renderers
//

function truncate(value: string, max: number): string {
  if (value.length <= max) return value;
  return value.slice(0, Math.max(0, max - 1)) + "…";
}

const BADGE_PILL_CLASS =
  "inline-flex items-center rounded-full border px-2 py-0.5 text-[11px] font-medium leading-none bg-emerald-100 text-emerald-700 border-emerald-200";

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
    case "badge":
      return <span className={BADGE_PILL_CLASS}>{text}</span>;
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
  // Numeric string -> Unix epoch seconds.
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
// Component
//

export function QueryBlock({ body, canvasId }: QueryBlockProps) {
  const parsed = useMemo(() => parseQueryBody(body), [body]);

  // Always call the hook; gate it via `enabled` so invalid blocks don't
  // generate API traffic. Order of hooks must stay stable across renders.
  const enabled = parsed.kind === "ok";
  const memoryQuery = useCanvasMemoryEntries(canvasId, enabled);

  if (parsed.kind === "error") {
    return <QueryBlockError message={parsed.error.message} body={body} />;
  }

  const { namespace, columns: explicitColumns, where } = parsed.query;

  if (memoryQuery.isLoading) {
    return <QueryBlockSkeleton />;
  }

  if (memoryQuery.isError) {
    const message = memoryQuery.error instanceof Error ? memoryQuery.error.message : "Unknown error";
    return <QueryBlockError message={`Failed to load memory: ${message}`} body={body} />;
  }

  const inNamespace = (memoryQuery.data ?? []).filter((entry) => entry.namespace === namespace);
  const filtered = where ? applyWhere(inNamespace, where) : inNamespace;

  if (filtered.length === 0) {
    return (
      <div
        data-testid="canvas-query-block-empty"
        className="my-2 flex items-center justify-center rounded border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-xs text-slate-500"
      >
        No entries in &quot;{namespace}&quot;
      </div>
    );
  }

  const columns: ColumnSpec[] = explicitColumns ?? autoColumns(filtered);

  return (
    <div data-testid="canvas-query-block" className="my-2 overflow-x-auto rounded border border-slate-200">
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
          </tr>
        </thead>
        <tbody>
          {filtered.map((entry) => {
            const values = isPlainObject(entry.values) ? entry.values : {};
            return (
              <tr key={entry.id}>
                {columns.map((column, i) => (
                  <td key={`${column.field}-${i}`} className="border-b border-slate-100 px-3 py-1.5 align-top">
                    {renderCell(values[column.field], column.format)}
                  </td>
                ))}
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}

function autoColumns(entries: CanvasMemoryEntry[]): ColumnSpec[] {
  return collectColumnFields(entries).map<ColumnSpec>((field) => ({
    label: field,
    field,
    format: "plain",
  }));
}

function collectColumnFields(entries: CanvasMemoryEntry[]): string[] {
  const set = new Set<string>();
  for (const entry of entries) {
    if (!isPlainObject(entry.values)) continue;
    for (const key of Object.keys(entry.values)) set.add(key);
  }
  return Array.from(set).sort((a, b) => a.localeCompare(b));
}

function QueryBlockSkeleton() {
  return (
    <div
      data-testid="canvas-query-block-skeleton"
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

function QueryBlockError({ message, body }: { message: string; body: string }) {
  return (
    <div
      data-testid="canvas-query-block-error"
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
