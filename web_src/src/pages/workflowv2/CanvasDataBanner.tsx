import { Trash2 } from "lucide-react";
import { Button } from "@/components/ui/button";

interface CanvasDataEntry {
  key: string;
  value: unknown;
  updatedAt: string;
}

interface CanvasDataViewProps {
  entries: CanvasDataEntry[];
  onDeleteEntry?: (key: string) => void;
  deletingKey?: string;
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

function renderValueTable(value: unknown) {
  if (Array.isArray(value)) {
    if (value.length === 0) {
      return <div className="px-3 py-2 text-xs text-gray-500">No items</div>;
    }

    const objectItems = value.filter(isRecord) as Record<string, unknown>[];
    if (objectItems.length === value.length) {
      const columns = collectColumns(objectItems);
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
              </tr>
            </thead>
            <tbody>
              {objectItems.map((item, index) => (
                <tr key={index} className="border-b border-slate-100">
                  {columns.map((column) => (
                    <td key={`${index}-${column}`} className="px-3 py-2 font-mono text-xs text-gray-700 align-top">
                      {formatValue(item[column])}
                    </td>
                  ))}
                </tr>
              ))}
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
              <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Item</th>
            </tr>
          </thead>
          <tbody>
            {value.map((item, index) => (
              <tr key={index} className="border-b border-slate-100">
                <td className="px-3 py-2 font-mono text-xs text-gray-700">{formatValue(item)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  if (isRecord(value)) {
    const rows = Object.entries(value);
    return (
      <div className="overflow-x-auto">
        <table className="w-full text-sm">
          <thead>
            <tr className="border-b border-slate-200 bg-slate-50">
              <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Field</th>
              <th className="px-3 py-2 text-left text-xs font-semibold text-gray-600 uppercase">Value</th>
            </tr>
          </thead>
          <tbody>
            {rows.map(([field, fieldValue]) => (
              <tr key={field} className="border-b border-slate-100">
                <td className="px-3 py-2 font-mono text-xs text-gray-700">{field}</td>
                <td className="px-3 py-2 font-mono text-xs text-gray-700">{formatValue(fieldValue)}</td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    );
  }

  return <div className="px-3 py-2 font-mono text-xs text-gray-700">{formatValue(value)}</div>;
}

export function CanvasDataView({ entries, onDeleteEntry, deletingKey }: CanvasDataViewProps) {
  return (
    <div className="p-4">
      <div className="rounded-lg border border-slate-200 bg-white">
        <div className="border-b border-slate-200 px-4 py-3">
          <h2 className="text-sm font-semibold text-gray-900">Canvas Data</h2>
          <p className="text-xs text-gray-500">Shared key/value storage for this canvas.</p>
        </div>
        {entries.length === 0 ? (
          <div className="px-4 py-6 text-sm text-gray-500">No key/value pairs set yet.</div>
        ) : (
          <div className="divide-y divide-slate-200">
            {entries.map((entry) => (
              <div key={entry.key}>
                <div className="flex items-center justify-between px-4 py-2 border-b border-slate-100">
                  <div className="font-mono text-sm text-gray-800">{entry.key}</div>
                  <Button
                    type="button"
                    variant="ghost"
                    size="icon-sm"
                    disabled={deletingKey === entry.key || !onDeleteEntry}
                    onClick={() => onDeleteEntry?.(entry.key)}
                    className="text-gray-500 hover:text-red-600"
                    title="Delete key"
                  >
                    <Trash2 className="h-4 w-4" />
                  </Button>
                </div>
                {renderValueTable(entry.value)}
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
