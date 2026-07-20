import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Popover, PopoverContent, PopoverTrigger } from "@/ui/popover";
import { RunStatusFilterSection, RunTriggerFilterSection, type TriggerOption } from "@/ui/Runs/RunFilterSections";
import { type RunStatusFilter } from "@/ui/Runs/runPresentation";
import { ListFilter } from "lucide-react";

export type { TriggerOption };

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
            "relative shrink-0 hover:bg-gray-100 dark:hover:bg-gray-800",
            hasTriggerFilter || hasStatusFilter
              ? "text-sky-700 hover:bg-sky-100 dark:text-indigo-300 dark:hover:bg-gray-800"
              : "text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200",
          )}
          aria-label="Filter runs"
          title="Filter runs"
        >
          <ListFilter className="size-3.5 shrink-0" aria-hidden />
          {hasTriggerFilter || hasStatusFilter ? (
            <span className="absolute -right-0.5 -top-0.5 flex h-3 min-w-3 items-center justify-center rounded-full bg-sky-500 px-0.5 text-[8px] font-semibold leading-none text-white dark:bg-indigo-300 dark:text-indigo-950">
              {totalFilters}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent
        align="end"
        className="w-64 border-slate-950/20 bg-white p-0 shadow-md dark:border-gray-800/70 dark:bg-gray-900"
        sideOffset={4}
      >
        <RunStatusFilterSection
          selectedStatuses={selectedStatuses}
          onToggleStatus={onToggleStatus}
          onClearStatuses={onClearStatuses}
        />
        <RunTriggerFilterSection
          triggerOptions={triggerOptions}
          selectedTriggerIds={selectedTriggerIds}
          onToggleTrigger={onToggleTrigger}
          onClearTriggers={onClearTriggers}
          headerClassName="border-t border-slate-950/10 dark:border-gray-800/70"
        />
      </PopoverContent>
    </Popover>
  );
}
