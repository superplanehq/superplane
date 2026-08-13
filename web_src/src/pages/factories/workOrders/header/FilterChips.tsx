import { X } from "lucide-react";
import type { WorkOrderListState } from "../../lib/useWorkOrderListState";
import { buildWorkOrderFilterChips, type WorkOrderFilterOption } from "../../lib/workOrderFilterOptions";

interface FilterChipsProps {
  state: WorkOrderListState;
  lineOptions: WorkOrderFilterOption[];
  assigneeOptions: WorkOrderFilterOption[];
}

/** Applied filters, rendered under the title bar only when something is set. */
export function FilterChips({ state, lineOptions, assigneeOptions }: FilterChipsProps) {
  if (state.filterCount === 0) {
    return null;
  }

  const chips = buildWorkOrderFilterChips(state.filters, { lines: lineOptions, assignees: assigneeOptions });

  return (
    <div className="mt-3 flex flex-wrap items-center gap-1.5" data-testid="work-orders-filter-chips">
      {chips.map((chip) => (
        <span
          key={`${chip.dimension}:${chip.value}`}
          className="inline-flex h-7 items-center gap-1 rounded-md border border-border bg-background px-2 text-[12px] text-foreground"
        >
          {chip.label}
          <button
            type="button"
            onClick={() => state.removeFilter(chip.dimension, chip.value)}
            aria-label={`Remove filter ${chip.label}`}
            className="rounded p-0.5 text-muted-foreground hover:text-foreground"
          >
            <X className="size-3" aria-hidden />
          </button>
        </span>
      ))}
      <button
        type="button"
        onClick={state.clearFilters}
        className="px-1.5 text-[12px] text-muted-foreground hover:text-foreground"
        data-testid="work-orders-filter-clear"
      >
        Clear
      </button>
    </div>
  );
}
