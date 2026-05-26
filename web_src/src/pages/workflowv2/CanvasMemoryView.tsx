import { Button } from "@/components/ui/button";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { useEffectiveLeftSidebarWidth } from "@/stores/sidebarLayoutStore";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/ui/collapsible";
import { ChevronDown, Trash2 } from "lucide-react";
import { useState } from "react";

export type CanvasMemoryViewProps = {
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

export function CanvasMemoryView(props: CanvasMemoryViewProps) {
  const leftOffset = useEffectiveLeftSidebarWidth();

  return (
    <div
      className="absolute bottom-0 top-[5rem] z-10 flex flex-col overflow-hidden bg-slate-50"
      style={{ left: leftOffset, right: 0 }}
      data-testid="memory-overlay"
    >
      <CanvasMemoryViewBody {...props} />
    </div>
  );
}

function CanvasMemoryViewBody({ entries, isLoading, errorMessage, onDeleteEntry, deletingId }: CanvasMemoryViewProps) {
  const groupedEntries = entries.reduce<Record<string, CanvasMemoryEntry[]>>((acc, entry) => {
    const namespace = entry.namespace || "(no namespace)";
    if (!acc[namespace]) {
      acc[namespace] = [];
    }
    acc[namespace].push(entry);
    return acc;
  }, {});

  if (isLoading) {
    return (
      <div className="flex min-h-0 w-full flex-1 items-center justify-center px-4 py-12 text-[13px] text-gray-500">
        Loading memory entries…
      </div>
    );
  }

  if (errorMessage) {
    return (
      <div className="flex min-h-0 w-full flex-1 flex-col items-center justify-center gap-2 px-6 py-12 text-center">
        <p className="text-[13px] font-medium text-red-600">Failed to load memory entries.</p>
        <p className="max-w-md text-xs text-red-500">{errorMessage}</p>
      </div>
    );
  }

  if (entries.length === 0) {
    return <ZeroState />;
  }

  return (
    <div className="min-h-0 w-full min-w-0 flex-1 overflow-auto">
      {Object.entries(groupedEntries).map(([namespace, values]) => (
        <NamespaceSection
          key={namespace}
          namespace={namespace}
          values={values}
          onDeleteEntry={onDeleteEntry}
          deletingId={deletingId}
        />
      ))}
    </div>
  );
}

type NamespaceSectionProps = {
  namespace: string;
  values: CanvasMemoryEntry[];
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

function NamespaceSection({ namespace, values, onDeleteEntry, deletingId }: NamespaceSectionProps) {
  const [isOpen, setIsOpen] = useState(true);

  return (
    <Collapsible
      open={isOpen}
      onOpenChange={setIsOpen}
      className="m-4 border border-slate-300 rounded-md bg-white"
      data-testid={`memory-namespace-section-${namespace}`}
    >
      <CollapsibleTrigger
        className="group flex w-full items-center gap-2 px-3 py-2 text-left font-mono text-sm text-gray-600 border-b border-slate-300 data-[state=closed]:border-b-0 hover:bg-slate-50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring/50"
        data-testid={`memory-namespace-toggle-${namespace}`}
        aria-label={`Toggle ${namespace} namespace`}
      >
        <ChevronDown
          aria-hidden="true"
          className="size-4 shrink-0 text-gray-500 transition-transform duration-150 group-data-[state=closed]:-rotate-90"
        />
        <span className="flex-1 truncate">Namespace: {namespace}</span>
        <span className="shrink-0 text-xs font-normal text-gray-500">
          {values.length} {values.length === 1 ? "item" : "items"}
        </span>
      </CollapsibleTrigger>
      <CollapsibleContent>{renderNamespaceTable(values, onDeleteEntry, deletingId)}</CollapsibleContent>
    </Collapsible>
  );
}

function ZeroState() {
  return (
    <div
      role="status"
      className="flex min-h-0 w-full flex-1 flex-col items-center justify-center gap-4 px-6 py-16 text-center sm:px-10"
    >
      <p className="text-base font-medium text-gray-900">No canvas memory yet</p>
      <p className="max-w-lg text-pretty text-sm leading-relaxed text-gray-500">
        Use memory components on your canvas—for example <span className="font-medium text-gray-700">Add Memory</span>,{" "}
        <span className="font-medium text-gray-700">Read Memory</span>, or{" "}
        <span className="font-medium text-gray-700">Upsert Memory</span>. After a run writes to canvas memory, entries
        will show up here.
      </p>
    </div>
  );
}

function formatValue(value: unknown): string {
  if (typeof value === "string") {
    return value;
  }

  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function collectColumns(items: Record<string, unknown>[]): string[] {
  const set = new Set<string>();
  items.forEach((item) => {
    Object.keys(item).forEach((key) => set.add(key));
  });
  return Array.from(set);
}

function renderNamespaceTable(
  values: CanvasMemoryEntry[],
  onDeleteEntry?: (memoryId: string) => void,
  deletingId?: string,
) {
  if (values.length === 0) {
    return <div className="px-3 py-2 text-xs text-gray-500">No items</div>;
  }

  const showActions = !!onDeleteEntry;
  const objectValues = values.map((entry) => entry.values).filter(isRecord) as Record<string, unknown>[];
  if (objectValues.length === values.length) {
    const columns = collectColumns(objectValues);
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-950/15 bg-slate-50">
              {columns.map((column) => (
                <th key={column} className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">
                  {column}
                </th>
              ))}
              {showActions ? (
                <th className="w-12 px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase"></th>
              ) : null}
            </tr>
          </thead>
          <tbody>
            {values.map((entry, index) => {
              const item = objectValues[index];
              return (
                <tr key={entry.id || index} className="border-b border-slate-950/15">
                  {columns.map((column) => (
                    <td key={`${index}-${column}`} className="px-3 py-2 align-middle font-mono text-xs text-gray-700">
                      {formatValue(item[column])}
                    </td>
                  ))}
                  {showActions ? (
                    <td className="px-3 py-2 text-right align-middle">
                      <Button
                        type="button"
                        variant="ghost"
                        size="icon-sm"
                        disabled={!entry.id || deletingId === entry.id}
                        onClick={() => {
                          if (entry.id) onDeleteEntry?.(entry.id);
                        }}
                        className="text-gray-500 hover:text-red-600"
                        title="Delete entry"
                      >
                        <Trash2 className="h-4 w-4" />
                      </Button>
                    </td>
                  ) : null}
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }

  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-950/15 bg-slate-50">
            <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Value</th>
            {showActions ? (
              <th className="w-12 px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase"></th>
            ) : null}
          </tr>
        </thead>
        <tbody>
          {values.map((entry, index) => (
            <tr key={entry.id || index} className="border-b border-slate-950/15">
              <td className="px-3 py-2 align-middle font-mono text-xs text-gray-700">{formatValue(entry.values)}</td>
              {showActions ? (
                <td className="px-3 py-2 text-right align-middle">
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    disabled={!entry.id || deletingId === entry.id}
                    onClick={() => {
                      if (entry.id) onDeleteEntry?.(entry.id);
                    }}
                    className="text-gray-500 hover:text-red-600"
                    title="Delete entry"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </td>
              ) : null}
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
