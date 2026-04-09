import { type RunsStatusFilter } from "@/pages/workflowv2/lib/canvas-runs";
import { cn } from "@/lib/utils";

export function FilterBar({
  statusFilter,
  onFilterChange,
  counts,
}: {
  statusFilter: RunsStatusFilter;
  onFilterChange: (f: RunsStatusFilter) => void;
  counts: { all: number; completed: number; errors: number; running: number; queued: number };
}) {
  const buttons: { key: RunsStatusFilter; label: string; count: number }[] = [
    { key: "all", label: "All", count: counts.all },
    { key: "completed", label: "Completed", count: counts.completed },
    { key: "errors", label: "Errors", count: counts.errors },
    { key: "running", label: "Running", count: counts.running },
    { key: "queued", label: "Queued", count: counts.queued },
  ];
  return (
    <div className="flex items-center gap-1 px-4 py-1.5 border-b border-gray-200">
      {buttons.map((btn) => (
        <button
          key={btn.key}
          type="button"
          onClick={() => onFilterChange(btn.key)}
          className={cn(
            "rounded-md px-2 py-0.5 text-xs font-medium transition-colors",
            statusFilter === btn.key ? "bg-slate-900 text-white" : "text-gray-600 hover:bg-gray-100",
          )}
        >
          {btn.label}
          {btn.count > 0 && (
            <span className={cn("ml-1 tabular-nums", statusFilter === btn.key ? "text-white/70" : "text-gray-400")}>
              {btn.count}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
