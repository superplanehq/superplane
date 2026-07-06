import type { CanvasesCanvasRun, SuperplaneComponentsNode as ComponentsNode } from "@/api-client";
import type { RefObject, UIEvent } from "react";
import type { RunStatusFilter, RunStatusKey } from "@/ui/Runs/runPresentation";
import { RunsList } from "./RunsList";
import { RunsToolbar } from "./RunsToolbar";
import type { TriggerOption } from "./RunFiltersPopover";

type DecoratedRun = {
  run: CanvasesCanvasRun;
  triggerName: string;
  title: string;
  status: RunStatusKey;
  triggerNode?: ComponentsNode;
};

interface RunsTabListViewProps {
  isActive: boolean;
  scrollRef: RefObject<HTMLDivElement | null>;
  onScroll: (event: UIEvent<HTMLDivElement>) => void;
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
  hasAnyFilter: boolean;
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  triggerOptions: TriggerOption[];
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
}

export function RunsTabListView({
  isActive,
  scrollRef,
  onScroll,
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
  hasAnyFilter,
  selectedStatuses,
  selectedTriggerIds,
  triggerOptions,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
}: RunsTabListViewProps) {
  return (
    <div
      className={`absolute inset-0 flex min-h-0 min-w-0 flex-col overflow-hidden bg-white transition-transform duration-300 ease-in-out dark:bg-gray-900 ${
        isActive ? "translate-x-0" : "-translate-x-full"
      } ${isActive ? "pointer-events-auto" : "pointer-events-none"}`}
    >
      <RunsToolbar
        selectedStatuses={selectedStatuses}
        selectedTriggerIds={selectedTriggerIds}
        triggerOptions={triggerOptions}
        onToggleStatus={onToggleStatus}
        onClearStatuses={onClearStatuses}
        onToggleTrigger={onToggleTrigger}
        onClearTriggers={onClearTriggers}
      />

      <div
        ref={scrollRef}
        className="min-h-0 min-w-0 flex-1 overflow-x-hidden overflow-y-auto"
        data-testid="runs-sidebar-scroll"
        onScroll={onScroll}
      >
        <RunsList
          runs={runs}
          filteredRuns={filteredRuns}
          orderedRuns={orderedRuns}
          selectedRunId={selectedRunId}
          onSelectRun={onSelectRun}
          componentIconMap={componentIconMap}
          isLoading={isLoading}
          isError={isError}
          onRetry={onRetry}
          onClearFilters={onClearFilters}
        />
      </div>

      {hasAnyFilter && runs.length > 0 ? (
        <div className="flex shrink-0 items-center justify-between gap-2 border-t border-slate-950/15 bg-slate-50 px-3 py-1.5 text-[11px] text-gray-500 dark:border-gray-800/70 dark:bg-gray-900 dark:text-gray-400">
          <span>
            Showing {filteredRuns.length} of {runs.length} loaded
          </span>
          <button
            type="button"
            onClick={onClearFilters}
            className="shrink-0 text-sky-600 hover:text-sky-800 dark:text-indigo-300 dark:hover:text-indigo-200"
          >
            Clear filters
          </button>
        </div>
      ) : null}
    </div>
  );
}
