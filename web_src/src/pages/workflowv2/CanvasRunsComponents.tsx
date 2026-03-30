import { Loader2 } from "lucide-react";
import { cn } from "@/lib/utils";
import type { RunsStatusFilter } from "./canvasRunsUtils";

export function RunsFilterBar({
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

export function LoadMoreButton({
  isFetchingNextPage,
  onLoadMore,
  loadedCount,
  totalCount,
}: {
  isFetchingNextPage?: boolean;
  onLoadMore?: () => void;
  loadedCount: number;
  totalCount: number;
}) {
  return (
    <div className="px-4 pt-2 pb-8 text-center">
      <button
        type="button"
        onClick={onLoadMore}
        disabled={isFetchingNextPage}
        className="text-xs font-medium text-slate-500 hover:text-slate-700 disabled:text-gray-400 transition-colors"
      >
        {isFetchingNextPage ? (
          <span className="inline-flex items-center gap-1">
            <Loader2 className="h-3 w-3 animate-spin" />
            Loading...
          </span>
        ) : (
          `Load more (${loadedCount} of ${totalCount})`
        )}
      </button>
    </div>
  );
}
