/**
 * Presentational status + trigger filter sections shared by the canvas
 * runs sidebar popover and the console run-datasource editor. Extracted
 * so the two surfaces stay visually identical and the filter vocabulary
 * (status labels, checkbox rows, empty state, clear affordance) has a
 * single source of truth.
 */

import { Checkbox } from "@/components/ui/checkbox";
import { cn } from "@/lib/utils";
import { RunNodeIcon, RUN_NODE_ICON_SIZE } from "@/ui/Runs/RunNodeIcon";
import { RUN_STATUS_FILTER_OPTIONS, type RunStatusFilter } from "@/ui/Runs/runPresentation";

export interface TriggerOption {
  id: string;
  name: string;
  iconSrc?: string;
  iconSlug?: string;
}

/**
 * Compact header rendered above a filter section — title on the left, a
 * disabled-when-empty "Clear" affordance on the right.
 */
export function RunFilterHeader({
  title,
  hasFilter,
  onClear,
  className,
}: {
  title: string;
  hasFilter: boolean;
  onClear: () => void;
  className?: string;
}) {
  return (
    <div className={cn("flex items-center justify-between px-3 py-2", className)}>
      <span className="text-[12px] font-medium text-gray-700 dark:text-gray-300">{title}</span>
      <button
        type="button"
        onClick={onClear}
        disabled={!hasFilter}
        className={cn(
          "text-[11px]",
          hasFilter
            ? "text-sky-600 hover:text-sky-800 dark:text-indigo-300 dark:hover:text-indigo-200"
            : "text-gray-400 dark:text-gray-500",
        )}
      >
        Clear
      </button>
    </div>
  );
}

/**
 * Multi-select checkbox list of the four run-status categories. Empty
 * `selectedStatuses` means "all statuses" — the callers rely on that
 * convention rather than pre-selecting every option.
 */
export function RunStatusFilterSection({
  selectedStatuses,
  onToggleStatus,
  onClearStatuses,
}: {
  selectedStatuses: Set<RunStatusFilter>;
  onToggleStatus: (status: RunStatusFilter) => void;
  onClearStatuses: () => void;
}) {
  const hasStatusFilter = selectedStatuses.size > 0;
  return (
    <>
      <RunFilterHeader title="Filter by status" hasFilter={hasStatusFilter} onClear={onClearStatuses} />
      <div className="py-1">
        {RUN_STATUS_FILTER_OPTIONS.map((option) => (
          <label
            key={option.id}
            className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
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
    </>
  );
}

/**
 * Multi-select checkbox list of every trigger node currently on the
 * canvas. Renders a graceful empty state when no trigger nodes exist so
 * authors understand why the list is empty.
 */
export function RunTriggerFilterSection({
  triggerOptions,
  selectedTriggerIds,
  onToggleTrigger,
  onClearTriggers,
  headerClassName,
}: {
  triggerOptions: TriggerOption[];
  selectedTriggerIds: Set<string>;
  onToggleTrigger: (triggerId: string) => void;
  onClearTriggers: () => void;
  /** Extra classes for the header (e.g. add a top border when stacked below the status section). */
  headerClassName?: string;
}) {
  const hasTriggerFilter = selectedTriggerIds.size > 0;
  return (
    <>
      <RunFilterHeader
        title="Filter by trigger"
        hasFilter={hasTriggerFilter}
        onClear={onClearTriggers}
        className={headerClassName}
      />
      <div className="max-h-64 overflow-y-auto py-1">
        {triggerOptions.length === 0 ? (
          <div className="px-3 py-4 text-center text-[11px] text-gray-400 dark:text-gray-500">
            No triggers in this canvas
          </div>
        ) : (
          triggerOptions.map((option) => (
            <label
              key={option.id}
              className="flex cursor-pointer items-center gap-2 px-3 py-1.5 text-[12px] text-gray-700 hover:bg-gray-50 dark:text-gray-300 dark:hover:bg-gray-800"
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
                className="shrink-0 text-gray-500 dark:text-gray-400"
              />
              <span className="min-w-0 truncate">{option.name}</span>
            </label>
          ))
        )}
      </div>
    </>
  );
}
