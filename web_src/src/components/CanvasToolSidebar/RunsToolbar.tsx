import { Search, X } from "lucide-react";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { cn } from "@/lib/utils";
import { RunFiltersPopover, type TriggerOption } from "./RunFiltersPopover";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

interface RunsToolbarProps {
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  triggerOptions: TriggerOption[];
  searchQuery: string;
  onSearchChange: (query: string) => void;
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
}

export function RunsToolbar({
  selectedStatuses,
  selectedTriggerIds,
  triggerOptions,
  searchQuery,
  onSearchChange,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
}: RunsToolbarProps) {
  return (
    <div className={cn(RUNS_SIDEBAR_ROW_CLASS, "pr-1.5")}>
      <span className="shrink-0 text-[11px] font-medium uppercase tracking-wide text-gray-500">Runs</span>
      <div className="flex min-w-0 flex-1 items-center gap-1 rounded border border-transparent px-1.5 transition-colors focus-within:border-slate-300">
        <Search className="h-3.5 w-3.5 shrink-0 text-gray-400" />
        <input
          type="text"
          value={searchQuery}
          onChange={(event) => onSearchChange(event.target.value)}
          placeholder="Search runs"
          aria-label="Search runs"
          className="min-w-0 flex-1 bg-transparent text-[12px] text-gray-700 placeholder:text-gray-400 focus:outline-none"
        />
        {searchQuery ? (
          <button
            type="button"
            onClick={() => onSearchChange("")}
            aria-label="Clear search"
            className="shrink-0 rounded p-0.5 text-gray-400 transition-colors hover:bg-gray-100 hover:text-gray-600"
          >
            <X className="h-3 w-3" />
          </button>
        ) : null}
      </div>
      <RunFiltersPopover
        selectedStatuses={selectedStatuses}
        selectedTriggerIds={selectedTriggerIds}
        triggerOptions={triggerOptions}
        onToggleStatus={onToggleStatus}
        onClearStatuses={onClearStatuses}
        onToggleTrigger={onToggleTrigger}
        onClearTriggers={onClearTriggers}
      />
    </div>
  );
}
