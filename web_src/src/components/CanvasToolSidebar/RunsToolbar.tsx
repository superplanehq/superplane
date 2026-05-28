import { Input } from "@/components/ui/input";
import { cn } from "@/lib/utils";
import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { Search, X } from "lucide-react";
import { RunFiltersPopover, type TriggerOption } from "./RunFiltersPopover";

interface RunsToolbarProps {
  search: string;
  onSearchChange: (value: string) => void;
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  triggerOptions: TriggerOption[];
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
}

export function RunsToolbar({
  search,
  onSearchChange,
  selectedStatuses,
  selectedTriggerIds,
  triggerOptions,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
}: RunsToolbarProps) {
  const hasSearch = search.trim().length > 0;

  return (
    <div className="flex shrink-0 items-center gap-1.5 border-b border-slate-200 px-2 py-1.5">
      <RunFiltersPopover
        selectedStatuses={selectedStatuses}
        selectedTriggerIds={selectedTriggerIds}
        triggerOptions={triggerOptions}
        onToggleStatus={onToggleStatus}
        onClearStatuses={onClearStatuses}
        onToggleTrigger={onToggleTrigger}
        onClearTriggers={onClearTriggers}
      />

      <div className="relative min-w-0 flex-1">
        <Search
          className="pointer-events-none absolute left-2.5 top-1/2 size-3.5 -translate-y-1/2 text-gray-400"
          aria-hidden="true"
        />
        <Input
          type="text"
          placeholder="Search runs..."
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          className={cn("h-7 text-xs shadow-none focus:ring-0 focus-visible:ring-0", hasSearch ? "pl-8 pr-8" : "pl-8")}
        />
        {hasSearch ? (
          <button
            type="button"
            aria-label="Clear search"
            onClick={() => onSearchChange("")}
            className="absolute right-2 top-1/2 -translate-y-1/2 rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
          >
            <X className="size-3.5" />
          </button>
        ) : null}
      </div>
    </div>
  );
}
