import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { RunStatusKey } from "@/ui/Runs/runPresentation";
import { AlertCircle, Loader2, Rabbit } from "lucide-react";
import { RunRow } from "./RunRow";

type DecoratedRun = {
  run: CanvasesCanvasRun;
  triggerName: string;
  title: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
};

interface RunsListProps {
  runs: CanvasesCanvasRun[];
  filteredRuns: DecoratedRun[];
  orderedRuns: { active: DecoratedRun[]; rest: DecoratedRun[] };
  selectedRunId: string | null;
  onSelectRun: (runId: string) => void;
  componentIconMap: Record<string, string>;
  isLoading?: boolean;
  isError?: boolean;
  onRetry?: () => void;
  onClearFilters: () => void;
}

export function RunsList({
  runs,
  filteredRuns,
  orderedRuns,
  selectedRunId,
  onSelectRun,
  componentIconMap,
  isLoading,
  isError,
  onRetry,
  onClearFilters,
}: RunsListProps) {
  if (isError && runs.length === 0) {
    return (
      <div role="alert" className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-gray-500">
        <AlertCircle className="h-5 w-5 text-red-500" aria-hidden />
        <span>Failed to load runs</span>
        {onRetry ? (
          <button type="button" onClick={onRetry} className="text-[11px] text-sky-600 hover:text-sky-800">
            Try again
          </button>
        ) : null}
      </div>
    );
  }

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-8">
        <Loader2 className="h-5 w-5 animate-spin text-gray-400" />
      </div>
    );
  }

  if (runs.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 px-3 py-6 text-center text-[13px] text-gray-500">
        <Rabbit className="h-4 w-4" aria-hidden />
        <span>No Runs</span>
      </div>
    );
  }

  if (filteredRuns.length === 0) {
    return (
      <div className="flex flex-col items-center gap-2 px-3 py-6 text-center text-xs text-gray-400">
        <span>No runs match your filters</span>
        <button type="button" onClick={onClearFilters} className="text-[11px] text-sky-600 hover:text-sky-800">
          Clear filters
        </button>
      </div>
    );
  }

  const renderRow = (item: DecoratedRun) => (
    <RunRow
      key={item.run.id}
      run={item.run}
      triggerName={item.triggerName}
      title={item.title}
      status={item.status}
      triggerNode={item.triggerNode}
      isSelected={item.run.id === selectedRunId}
      componentIconMap={componentIconMap}
      onSelectRun={onSelectRun}
    />
  );

  return (
    <>
      {orderedRuns.active.map(renderRow)}
      {orderedRuns.rest.map(renderRow)}
      {isError ? (
        <div role="alert" className="flex items-center justify-between gap-2 px-3 py-2 text-[11px] text-red-600">
          <span className="inline-flex items-center gap-1">
            <AlertCircle className="h-3 w-3" aria-hidden />
            Failed to load more runs
          </span>
          {onRetry ? (
            <button type="button" onClick={onRetry} className="text-sky-600 hover:text-sky-800">
              Retry
            </button>
          ) : null}
        </div>
      ) : null}
    </>
  );
}
