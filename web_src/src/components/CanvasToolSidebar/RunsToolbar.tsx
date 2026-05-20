import { InputGroup, InputGroupAddon, InputGroupInput } from "@/components/ui/input-group";
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

      <InputGroup className="h-7 flex-1 border border-slate-200 shadow-none !ring-0 focus-within:!ring-0 focus-within:ring-offset-0 [&_[data-slot=input-group-control]]:!text-[12px]">
        <InputGroupAddon className="!text-[12px]">
          <Search className="h-3.5 w-3.5 text-gray-500" />
        </InputGroupAddon>
        <InputGroupInput
          placeholder="Search runs..."
          value={search}
          onChange={(event) => onSearchChange(event.target.value)}
          className="h-6 !border-0 !text-[12px] shadow-none focus:ring-0 focus-visible:ring-0"
        />
        {hasSearch ? (
          <InputGroupAddon>
            <button
              type="button"
              aria-label="Clear search"
              onClick={() => onSearchChange("")}
              className="rounded p-0.5 text-gray-400 hover:bg-gray-100 hover:text-gray-700"
            >
              <X className="h-3 w-3" />
            </button>
          </InputGroupAddon>
        ) : null}
      </InputGroup>
    </div>
  );
}
