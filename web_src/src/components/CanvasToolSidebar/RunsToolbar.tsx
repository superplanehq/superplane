import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import type { ChangeEvent } from "react";
import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import { Search, X } from "lucide-react";
import { RunFiltersPopover, type TriggerOption } from "./RunFiltersPopover";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

interface RunsToolbarProps {
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  searchQuery: string;
  triggerOptions: TriggerOption[];
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
  onSearchQueryChange: (query: string) => void;
}

export function RunsToolbar({
  selectedStatuses,
  selectedTriggerIds,
  searchQuery,
  triggerOptions,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
  onSearchQueryChange,
}: RunsToolbarProps) {
  const handleSearchChange = (event: ChangeEvent<HTMLInputElement>) => {
    onSearchQueryChange(event.target.value);
  };

  return (
    <div className={cn(RUNS_SIDEBAR_ROW_CLASS, "gap-2 pr-1.5")}>
      <span className="shrink-0 text-[11px] font-medium uppercase tracking-wide text-gray-500 dark:text-gray-400">
        Runs
      </span>
      <div className="relative min-w-0 flex-1">
        <Search className="pointer-events-none absolute left-2 top-1/2 size-3.5 -translate-y-1/2 text-gray-400 dark:text-gray-500" />
        <Input
          type="text"
          value={searchQuery}
          onChange={handleSearchChange}
          aria-label="Search runs"
          placeholder="Search runs"
          className="h-7 rounded-none border-0 bg-transparent pl-7 pr-6 text-[13px] shadow-none placeholder:text-gray-400 focus:border-0 dark:bg-transparent dark:placeholder:text-gray-500"
        />
        {searchQuery && (
          <button
            type="button"
            aria-label="Clear search runs"
            onClick={() => onSearchQueryChange("")}
            className="absolute right-1 top-1/2 flex size-5 -translate-y-1/2 cursor-pointer items-center justify-center rounded text-gray-400 hover:text-gray-600 dark:text-gray-500 dark:hover:text-gray-300"
          >
            <X className="size-3.5" strokeWidth={1.5} />
          </button>
        )}
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
