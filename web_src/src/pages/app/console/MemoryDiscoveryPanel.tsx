import { useMemo, useState } from "react";
import { ChevronDown, Loader2 } from "lucide-react";

import { Button } from "@/components/ui/button";
import { useCanvasMemoryEntries, type CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";

import { useMemoryCatalog } from "./widget/useMemoryCatalog";

interface MemoryDiscoveryPanelProps {
  canvasId: string | undefined;
  selectedNamespace: string;
  onSelectNamespace: (namespace: string) => void;
}

const PREVIEW_ROW_LIMIT = 20;

/**
 * Surfaces namespaces and entry counts discovered from live canvas memory so
 * authors don't have to guess namespace strings. When a namespace is already
 * selected, the panel collapses into a read-only preview of the live entries
 * for that namespace so authors can sanity-check the data shape before
 * configuring columns and row actions.
 */
export function MemoryDiscoveryPanel({ canvasId, selectedNamespace, onSelectNamespace }: MemoryDiscoveryPanelProps) {
  const { namespaces, isLoading, isEmpty } = useMemoryCatalog(canvasId, selectedNamespace);

  if (isLoading) {
    return (
      <div className="flex items-center gap-2 rounded-md border border-dashed border-slate-200 bg-slate-50/80 px-3 py-2 text-xs text-slate-500 dark:border-gray-600 dark:bg-gray-800/80 dark:text-gray-400">
        <Loader2 className="h-3.5 w-3.5 animate-spin" />
        Scanning canvas memory…
      </div>
    );
  }

  if (isEmpty) {
    return (
      <div
        className="rounded-md border border-dashed border-amber-200 bg-amber-50/60 px-3 py-2 text-xs text-amber-800 dark:border-amber-800 dark:bg-amber-950/30 dark:text-amber-200"
        data-testid="memory-discovery-empty"
      >
        No canvas memory entries yet. Components write to memory during runs; once data exists, namespaces and fields
        will appear here.
      </div>
    );
  }

  const selected = namespaces.find((ns) => ns.namespace === selectedNamespace);

  return (
    <div className="space-y-2 rounded-lg bg-slate-100 px-3 py-2 dark:bg-gray-800" data-testid="memory-discovery-panel">
      {!selected ? (
        <>
          <p className="text-xs font-medium text-slate-700 dark:text-gray-300">
            This data exists in your canvas memory:
          </p>
          <div className="flex flex-wrap gap-1.5">
            {namespaces.map((ns) => (
              <Button
                key={ns.namespace}
                type="button"
                size="sm"
                variant="outline"
                className="h-7 text-xs"
                onClick={() => onSelectNamespace(ns.namespace)}
                data-testid={`memory-namespace-${ns.namespace}`}
              >
                {ns.namespace}
                <span className="ml-1 opacity-70">({ns.count})</span>
              </Button>
            ))}
          </div>
        </>
      ) : (
        <MemoryNamespacePreview canvasId={canvasId} namespace={selected.namespace} count={selected.count} />
      )}
    </div>
  );
}

function MemoryNamespacePreview({
  canvasId,
  namespace,
  count,
}: {
  canvasId: string | undefined;
  namespace: string;
  count: number;
}) {
  const [open, setOpen] = useState(false);
  const query = useCanvasMemoryEntries(canvasId ?? "", Boolean(canvasId));

  const entries = useMemo(() => {
    if (!query.data) return [];
    return query.data.filter((entry) => entry.namespace === namespace);
  }, [query.data, namespace]);

  return (
    <Collapsible open={open} onOpenChange={setOpen} data-testid="memory-namespace-preview">
      <CollapsibleTrigger
        className="group flex w-full items-center gap-2 rounded-md px-2 py-1.5 text-left text-xs text-slate-700 hover:bg-slate-100 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-slate-300 dark:text-gray-300 dark:hover:bg-gray-700 dark:focus-visible:ring-gray-600"
        data-testid={`memory-namespace-toggle-${namespace}`}
        aria-label={`Toggle ${namespace} preview`}
      >
        <ChevronDown
          aria-hidden="true"
          className="size-3.5 shrink-0 text-slate-500 transition-transform duration-150 group-data-[state=closed]:-rotate-90 dark:text-gray-400"
        />
        <span className="flex-1 truncate font-mono text-[11px]">{namespace}</span>
        <span className="shrink-0 text-[11px] text-slate-500 dark:text-gray-400">
          {count} {count === 1 ? "entry" : "entries"}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>
        <MemoryNamespaceTable entries={entries} isLoading={query.isLoading} />
      </CollapsibleContent>
    </Collapsible>
  );
}

function MemoryNamespaceTable({ entries, isLoading }: { entries: CanvasMemoryEntry[]; isLoading: boolean }) {
  if (isLoading) {
    return (
      <div className="flex items-center gap-2 px-2 py-2 text-[11px] text-slate-500 dark:text-gray-400">
        <Loader2 className="h-3 w-3 animate-spin" />
        Loading memory entries…
      </div>
    );
  }

  if (entries.length === 0) {
    return <div className="px-2 py-2 text-[11px] text-slate-500 dark:text-gray-400">No entries.</div>;
  }

  const limited = entries.slice(0, PREVIEW_ROW_LIMIT);
  const remaining = entries.length - limited.length;
  const objectRows = limited.map((entry) => entry.values).filter(isRecord) as Record<string, unknown>[];

  if (objectRows.length === limited.length) {
    const columns = collectColumns(objectRows);
    return (
      <div className="mt-1 max-h-56 overflow-auto rounded-md border border-slate-200 bg-white dark:border-gray-600 dark:bg-gray-900">
        <table className="w-full text-[11px]">
          <thead className="sticky top-0 bg-slate-50 dark:bg-gray-800">
            <tr>
              {columns.map((column) => (
                <th
                  key={column}
                  className="border-b border-slate-200 px-2 py-1 text-left font-semibold text-slate-600 dark:border-gray-600 dark:text-gray-300"
                >
                  {column}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {limited.map((entry, index) => {
              const row = objectRows[index]!;
              return (
                <tr key={entry.id || index} className="border-b border-slate-100 last:border-0 dark:border-gray-800">
                  {columns.map((column) => (
                    <td
                      key={`${index}-${column}`}
                      className="truncate px-2 py-1 align-top font-mono text-slate-700 dark:text-gray-300"
                      title={formatCell(row[column])}
                    >
                      {formatCell(row[column])}
                    </td>
                  ))}
                </tr>
              );
            })}
          </tbody>
        </table>
        {remaining > 0 ? (
          <p className="px-2 py-1 text-[10px] text-slate-500 dark:text-gray-400">and {remaining} more…</p>
        ) : null}
      </div>
    );
  }

  return (
    <div className="mt-1 max-h-56 overflow-auto rounded-md border border-slate-200 bg-white dark:border-gray-600 dark:bg-gray-900">
      <table className="w-full text-[11px]">
        <thead className="sticky top-0 bg-slate-50 dark:bg-gray-800">
          <tr>
            <th className="border-b border-slate-200 px-2 py-1 text-left font-semibold text-slate-600 dark:border-gray-600 dark:text-gray-300">
              Value
            </th>
          </tr>
        </thead>
        <tbody>
          {limited.map((entry, index) => (
            <tr key={entry.id || index} className="border-b border-slate-100 last:border-0 dark:border-gray-800">
              <td
                className="truncate px-2 py-1 align-top font-mono text-slate-700 dark:text-gray-300"
                title={formatCell(entry.values)}
              >
                {formatCell(entry.values)}
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      {remaining > 0 ? <p className="px-2 py-1 text-[10px] text-slate-500">and {remaining} more…</p> : null}
    </div>
  );
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function collectColumns(rows: Record<string, unknown>[]): string[] {
  const set = new Set<string>();
  for (const row of rows) {
    for (const key of Object.keys(row)) set.add(key);
  }
  return Array.from(set);
}

function formatCell(value: unknown): string {
  if (value == null) return "";
  if (typeof value === "string") return value;
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}
