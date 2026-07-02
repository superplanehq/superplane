import { useState } from "react";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_FILTER_OPTIONS, type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { ListFilter } from "lucide-react";

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
          size="icon-xs"
          className={cn(
            "relative shrink-0 hover:bg-gray-100",
            hasTriggerFilter || hasStatusFilter ? "text-sky-700 hover:bg-sky-100" : "text-gray-500 hover:text-gray-700",
          )}
          aria-label="Filter runs"
          title="Filter runs"
        >
          <ListFilter className="size-3.5 shrink-0" aria-hidden />
          {hasTriggerFilter || hasStatusFilter ? (
            <span className="absolute -right-0.5 -top-0.5 flex h-3 min-w-3 items-center justify-center rounded-full bg-sky-500 px-0.5 text-[8px] font-semibold leading-none text-white">
              {totalFilters}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 border-slate-950/20 bg-white p-0 shadow-md" sideOffset={4}>
        <div className="flex items-center justify-between px-3 py-2">
          <span className="text-[12px] font-medium text-gray-700">Filter by status</span>
          <button
            type="button"
            onClick={onClearStatuses}
            disabled={!hasStatusFilter}
            className={cn("text-[11px]", hasStatusFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-400")}
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
                onChange={() => onToggleStatus(option.id)}
                className="size-3.5"
              />
              <span className={cn("inline-block h-2 w-2 shrink-0 rounded-full", option.dotClassName)} />
              <span className="min-w-0 truncate">{option.label}</span>
            </label>
          ))}
        </div>

        <div className="flex items-center justify-between px-3 py-2">
          <span className="text-[12px] font-medium text-gray-700">Filter by trigger</span>
          <button
            type="button"
            onClick={onClearTriggers}
            disabled={!hasTriggerFilter}
            className={cn("text-[11px]", hasTriggerFilter ? "text-sky-600 hover:text-sky-800" : "text-gray-400")}
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
                  onChange={() => onToggleTrigger(option.id)}
                  className="size-3.5"
                />
                <RunNodeIcon
                  iconSrc={option.iconSrc}
                  iconSlug={option.iconSlug}
                  alt={option.name}
                  size={RUN_NODE_ICON_SIZE}
                  className="shrink-0 text-gray-500"
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
