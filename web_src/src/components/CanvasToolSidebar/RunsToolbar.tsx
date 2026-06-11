import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { cn } from "@/lib/utils";
import { Rabbit } from "lucide-react";
import { RunFiltersPopover, type TriggerOption } from "./RunFiltersPopover";
import { RUNS_SIDEBAR_ROW_CLASS } from "./runsSidebarRowLayout";

interface RunsToolbarProps {
  selectedStatuses: Set<RunStatusFilter>;
  selectedTriggerIds: Set<string>;
  triggerOptions: TriggerOption[];
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
}

export function RunsToolbar({
  selectedStatuses,
  selectedTriggerIds,
  triggerOptions,
  onToggleStatus,
  onClearStatuses,
  onToggleTrigger,
  onClearTriggers,
}: RunsToolbarProps) {
  return (
    <div className={cn(RUNS_SIDEBAR_ROW_CLASS, "justify-between pr-1.5")}>
      <span className="flex min-w-0 items-center gap-1.5">
        <Rabbit className="h-3.5 w-3.5 shrink-0 text-gray-500" aria-hidden />
        <span className="min-w-0 truncate text-[13px] font-medium text-gray-800">Runs</span>
      </span>
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
