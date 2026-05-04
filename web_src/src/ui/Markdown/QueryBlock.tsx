import { useMemo } from "react";
import * as yaml from "js-yaml";
import { AlertTriangle } from "lucide-react";

import { useCanvasMemoryEntries, type CanvasMemoryEntry } from "@/hooks/useCanvasData";

//
// Authored as a fenced code block inside Launchpad markdown panels:
//
//   ```query
//   source: memory
//   namespace: environments
//   ```
//
// v1 only supports `source: memory`. The block is parsed as YAML, validated,
// then rendered as a plain table with columns auto-derived from the union of
// `values` keys across matched memory entries. Failure modes (parse error,
// missing namespace, unsupported source, API error) all render an inline
// error card so the surrounding markdown keeps rendering.
//

export interface QueryBlockProps {
  body: string;
  canvasId: string;
}

interface ParsedQuery {
  source: string;
  namespace: string;
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

  return { kind: "ok", query: { source, namespace } };
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

export function QueryBlock({ body, canvasId }: QueryBlockProps) {
  const parsed = useMemo(() => parseQueryBody(body), [body]);

  // Always call the hook; gate it via `enabled` so invalid blocks don't
  // generate API traffic. Order of hooks must stay stable across renders.
  const enabled = parsed.kind === "ok";
  const memoryQuery = useCanvasMemoryEntries(canvasId, enabled);

  if (parsed.kind === "error") {
    return <QueryBlockError message={parsed.error.message} body={body} />;
  }

  const namespace = parsed.query.namespace;

  if (memoryQuery.isLoading) {
    return <QueryBlockSkeleton />;
  }

  if (memoryQuery.isError) {
    const message = memoryQuery.error instanceof Error ? memoryQuery.error.message : "Unknown error";
    return <QueryBlockError message={`Failed to load memory: ${message}`} body={body} />;
  }

  const entries = (memoryQuery.data ?? []).filter((entry) => entry.namespace === namespace);

  if (entries.length === 0) {
    return (
      <div
        data-testid="canvas-query-block-empty"
        className="my-2 flex items-center justify-center rounded border border-dashed border-slate-200 bg-slate-50/60 px-4 py-6 text-xs text-slate-500"
      >
        No entries in &quot;{namespace}&quot;
      </div>
    );
  }

  const columns = collectColumns(entries);

  return (
    <div data-testid="canvas-query-block" className="my-2 overflow-x-auto rounded border border-slate-200">
      <table className="min-w-full border-collapse text-left text-xs">
        <thead>
          <tr>
            {columns.map((column) => (
              <th
                key={column}
                className="border-b border-slate-200 bg-slate-50 px-3 py-1.5 text-xs font-semibold text-gray-600"
              >
                {column}
              </th>
            ))}
          </tr>
        </thead>
        <tbody>
          {entries.map((entry) => {
            const values = isPlainObject(entry.values) ? entry.values : {};
            return (
              <tr key={entry.id}>
                {columns.map((column) => (
                  <td key={column} className="border-b border-slate-100 px-3 py-1.5 align-top">
                    {stringifyCell(values[column])}
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

function collectColumns(entries: CanvasMemoryEntry[]): string[] {
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
