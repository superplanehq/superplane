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
            "relative shrink-0 hover:bg-action-neutral-hover",
            hasTriggerFilter || hasStatusFilter
              ? "text-status-info-content hover:bg-status-info-subtle"
              : "text-content-secondary hover:text-content-primary",
          )}
          aria-label="Filter runs"
          title="Filter runs"
        >
          <ListFilter className="size-3.5 shrink-0" aria-hidden />
          {hasTriggerFilter || hasStatusFilter ? (
            <span className="absolute -top-0.5 -right-0.5 flex h-3 min-w-3 items-center justify-center rounded-full bg-status-info px-0.5 text-[8px] leading-none font-semibold text-content-inverse">
              {totalFilters}
            </span>
          ) : null}
        </Button>
      </PopoverTrigger>
      <PopoverContent align="end" className="w-64 border-edge-default bg-surface-overlay p-0 shadow-md" sideOffset={4}>
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
          headerClassName="border-t border-edge-subtle"
        />
      </PopoverContent>
    </Popover>
  );
}
