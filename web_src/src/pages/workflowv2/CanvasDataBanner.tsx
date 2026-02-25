interface CanvasDataEntry {
  key: string;
  value: unknown;
  updatedAt: string;
}

interface CanvasDataViewProps {
  entries: CanvasDataEntry[];
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

export function CanvasDataView({ entries }: CanvasDataViewProps) {
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
              <div key={entry.key} className="grid grid-cols-[240px_1fr] gap-4 px-4 py-2 text-sm">
                <div className="font-mono text-gray-700">{entry.key}</div>
                <div className="font-mono text-gray-600 break-all">{formatValue(entry.value)}</div>
              </div>
            ))}
          </div>
        )}
      </div>
    </div>
  );
}
