import { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";

interface CanvasMemoryViewProps {
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
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

  const objectValues = values.map((entry) => entry.values).filter(isRecord) as Record<string, unknown>[];
  if (objectValues.length === values.length) {
    const columns = collectColumns(objectValues);
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              {columns.map((column) => (
                <th key={column} className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">
                  {column}
                </th>
              ))}
              <th className="px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase w-12"></th>
            </tr>
          </thead>
          <tbody>
            {values.map((entry, index) => {
              const item = objectValues[index];
              return (
                <tr key={entry.id || index} className="border-b border-slate-100">
                  {columns.map((column) => (
                    <td key={`${index}-${column}`} className="px-3 py-2 font-mono text-xs text-gray-700 align-middle">
                      {formatValue(item[column])}
                    </td>
                  ))}
                  <td className="px-3 py-2 text-right align-middle">
                    <Button
                      type="button"
                      variant="ghost"
                      size="icon-sm"
                      disabled={!onDeleteEntry || !entry.id || deletingId === entry.id}
                      onClick={() => {
                        if (entry.id) onDeleteEntry?.(entry.id);
                      }}
                      className="text-gray-500 hover:text-red-600"
                      title="Delete entry"
                    >
                      <Trash2 className="h-4 w-4" />
                    </Button>
                  </td>
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
          <tr className="border-b border-slate-200 bg-slate-50">
            <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Value</th>
            <th className="px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase w-12"></th>
          </tr>
        </thead>
        <tbody>
          {values.map((entry, index) => (
            <tr key={entry.id || index} className="border-b border-slate-100">
              <td className="px-3 py-2 font-mono text-xs text-gray-700 align-middle">{formatValue(entry.values)}</td>
              <td className="px-3 py-2 text-right align-middle">
                <Button
                  type="button"
                  variant="ghost"
                  size="icon-sm"
                  disabled={!onDeleteEntry || !entry.id || deletingId === entry.id}
                  onClick={() => {
                    if (entry.id) onDeleteEntry?.(entry.id);
                  }}
                  className="text-gray-500 hover:text-red-600"
                  title="Delete entry"
                >
                  <Trash2 className="h-4 w-4" />
                </Button>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}

export function CanvasMemoryView({
  entries,
  isLoading,
  errorMessage,
  onDeleteEntry,
  deletingId,
}: CanvasMemoryViewProps) {
  const groupedEntries = entries.reduce<Record<string, CanvasMemoryEntry[]>>((acc, entry) => {
    const namespace = entry.namespace || "(no namespace)";
    if (!acc[namespace]) {
      acc[namespace] = [];
    }
    acc[namespace].push(entry);
    return acc;
  }, {});

  return (
    <div className="p-4">
      <div className="rounded-lg border border-slate-200 bg-white">
        <div className="border-b border-slate-200 px-4 py-3">
          <h2 className="text-sm font-semibold text-gray-900">Canvas Memory</h2>
          <p className="text-xs text-gray-500">Shared memory entries for this canvas.</p>
        </div>
        {isLoading ? (
          <div className="px-4 py-6 text-sm text-gray-500">Loading memory entries...</div>
        ) : errorMessage ? (
          <div className="px-4 py-6 text-sm text-red-600">
            Failed to load memory entries.
            <div className="mt-1 text-xs text-red-500">{errorMessage}</div>
          </div>
        ) : entries.length === 0 ? (
          <div className="px-4 py-6 text-sm text-gray-500">No memory entries added yet.</div>
        ) : (
          <div className="divide-y divide-slate-200">
            {Object.entries(groupedEntries).map(([namespace, values]) => (
              <div key={namespace}>
                <div className="flex items-center justify-between px-4 py-2 border-b border-slate-100">
                  <div className="font-mono text-sm text-gray-800">{namespace}</div>
                </div>
                {renderNamespaceTable(values, onDeleteEntry, deletingId)}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
