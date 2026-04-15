import { Button } from "@/components/ui/button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import type { CanvasMemoryEntry } from "@/hooks/useCanvasData";
import { Trash2 } from "lucide-react";

export type CanvasMemoryModalProps = {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

export function CanvasMemoryModal(props: CanvasMemoryModalProps) {
  return (
    <Dialog open={props.open} onOpenChange={props.onOpenChange}>
      <DialogContent size="large" className="flex max-h-[90vh] w-[90vw] h-full flex-col gap-0 overflow-hidden p-0">
        <div className="flex h-full min-h-0 flex-col">
          <div className="flex shrink-0 items-center justify-between border-b border-gray-200 bg-white px-4 py-3">
            <span className="font-mono text-sm text-gray-600">Canvas Memory</span>
          </div>

          <div className="flex min-h-0 flex-1 flex-col overflow-hidden bg-slate-50">
            <CanvasMemoryModalBody
              entries={props.entries}
              isLoading={props.isLoading}
              errorMessage={props.errorMessage}
              onDeleteEntry={props.onDeleteEntry}
              deletingId={props.deletingId}
            />
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

type CanvasMemoryBodyProps = {
  entries: CanvasMemoryEntry[];
  isLoading?: boolean;
  errorMessage?: string;
  onDeleteEntry?: (memoryId: string) => void;
  deletingId?: string;
};

function CanvasMemoryModalBody({ entries, isLoading, errorMessage, onDeleteEntry, deletingId }: CanvasMemoryBodyProps) {
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
        <div key={namespace} className="m-4 border border-slate-300 rounded-md bg-white">
          <div className="px-3 py-2 font-mono text-sm text-gray-600 border-b border-slate-300">
            Namespace: {namespace}
          </div>

          {renderNamespaceTable(values, onDeleteEntry, deletingId)}
        </div>
      ))}
    </div>
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
              <th className="w-12 px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase"></th>
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
    <div className="overflow-x-auto bg-red-500">
      <table className="w-full text-sm">
        <thead>
          <tr className="border-b border-slate-950/15 bg-slate-50">
            <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Value</th>
            <th className="w-12 px-3 py-2 text-right text-xs font-semibold text-gray-600 uppercase"></th>
          </tr>
        </thead>
        <tbody>
          {values.map((entry, index) => (
            <tr key={entry.id || index} className="border-b border-slate-950/15">
              <td className="px-3 py-2 align-middle font-mono text-xs text-gray-700">{formatValue(entry.values)}</td>
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
