import type { RunStatusFilter } from "@/ui/Runs/runPresentation";
import { cn } from "@/lib/utils";
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
      <span className="min-w-0 truncate text-[11px] font-medium uppercase tracking-wide text-gray-500">Runs</span>
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
