import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Checkbox } from "@/ui/checkbox";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { RUN_STATUS_FILTER_OPTIONS, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { RunNodeIcon } from "@/ui/Runs/RunNodeIcon";
import { Filter } from "lucide-react";

export interface TriggerOption {
  id: string;
  name: string;
  iconSrc?: string;
  iconSlug?: string;
}

interface RunFiltersPopoverProps {
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  triggerOptions: TriggerOption[];
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
}

export function RunFiltersPopover({
  selectedStatuses,
  selectedTriggerIds,
  triggerOptions,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
}: RunFiltersPopoverProps) {
  const [isOpen, setIsOpen] = useState(false);
  const hasStatusFilter = selectedStatuses.size > 0;
  const hasTriggerFilter = selectedTriggerIds.size > 0;
  const totalFilters = selectedTriggerIds.size + selectedStatuses.size;

  return (
    <Popover open={isOpen} onOpenChange={setIsOpen}>
      <PopoverTrigger asChild>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          className={cn(
            "relative h-7 w-7 shrink-0 p-0",
            (hasTriggerFilter || hasStatusFilter) && "bg-sky-50 text-sky-700 hover:bg-sky-100",
          )}
          aria-label="Filter runs"
          title="Filter runs"
        >
          <Filter className="h-3.5 w-3.5" />
          {hasTriggerFilter || hasStatusFilter ? (
            <span className="absolute -right-0.5 -top-0.5 flex h-3.5 min-w-3.5 items-center justify-center rounded-full bg-sky-500 px-1 text-[9px] font-semibold text-white">
              {totalFilters}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-64 p-0" sideOffset={6}>
        <div className="flex items-center justify-between border-b border-slate-200 px-3 py-2">
          <span className="text-[12px] font-medium text-gray-700">Filter by status</span>
          <button
            type="button"
            onClick={onClearStatuses}
            disabled={!hasStatusFilter}
            className={cn("text-[11px]", hasStatusFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-300")}
          >
            Clear
          </button>
        </div>
        <div className="py-1">
          {RUN_STATUS_FILTER_OPTIONS.map((option) => (
            <label
              key={option.id}
              className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
            >
              <Checkbox
                checked={selectedStatuses.has(option.id)}
                onCheckedChange={() => onToggleStatus(option.id)}
                className="h-3.5 w-3.5"
              />
              <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", option.dotClassName)} />
              <span className="min-w-0 truncate">{option.label}</span>
            </label>
          ))}
        </div>

        <div className="flex items-center justify-between border-t border-slate-100 px-3 py-2">
          <span className="text-[12px] font-medium text-gray-700">Filter by trigger</span>
          <button
            type="button"
            onClick={onClearTriggers}
            disabled={!hasTriggerFilter}
            className={cn("text-[11px]", hasTriggerFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-300")}
          >
            Clear
          </button>
        </div>
        <div className="max-h-64 overflow-y-auto py-1">
          {triggerOptions.length === 0 ? (
            <div className="px-3 py-4 text-center text-[11px] text-gray-400">No triggers in this canvas</div>
          ) : (
            triggerOptions.map((option) => (
              <label
                key={option.id}
                className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50"
              >
                <Checkbox
                  checked={selectedTriggerIds.has(option.id)}
                  onCheckedChange={() => onToggleTrigger(option.id)}
                  className="h-3.5 w-3.5"
                />
                <RunNodeIcon
                  iconSrc={option.iconSrc}
                  iconSlug={option.iconSlug}
                  alt={option.name}
                  size={12}
                  className="shrink-0 text-gray-400"
                />
                <span className="min-w-0 truncate">{option.name}</span>
              </label>
            ))
          )}
        </div>
      </PopoverContent>
    </Popover>
  );
}
