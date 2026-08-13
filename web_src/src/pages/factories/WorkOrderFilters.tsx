import { cn } from "@/lib/utils";
import { factoryFilterPillActiveClassName, factoryFilterPillInactiveClassName } from "./lib/factoryPageStyles";
import type { WorkOrderDisplayStatus, WorkOrderOwnerFilter, WorkOrderStatusFilter } from "./lib/workOrderProgress";
import { getWorkOrderDisplayStatusMeta } from "./lib/workOrderProgress";

// `label` is a fallback for non-display filters (`all`, `active`);
// other pills resolve their label via `getWorkOrderDisplayStatusMeta`.
const STATUS_FILTERS: Array<{ id: WorkOrderStatusFilter; label: string }> = [
  { id: "all", label: "All statuses" },
  { id: "active", label: "Active" },
  { id: "draft", label: "Draft" },
  { id: "open", label: "Open" },
  { id: "running", label: "Running" },
  { id: "failed", label: "Failed" },
  { id: "completed", label: "Completed" },
  { id: "rejected", label: "Rejected" },
];

interface WorkOrderFiltersProps {
  ownerFilter: WorkOrderOwnerFilter;
  statusFilter: WorkOrderStatusFilter;
  onOwnerFilterChange: (value: WorkOrderOwnerFilter) => void;
  onStatusFilterChange: (value: WorkOrderStatusFilter) => void;
}

export function WorkOrderFilters({
  ownerFilter,
  statusFilter,
  onOwnerFilterChange,
  onStatusFilterChange,
}: WorkOrderFiltersProps) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3" data-testid="work-order-filters">
      <div className="flex flex-wrap items-center gap-2">
        <FilterPill active={ownerFilter === "mine"} onClick={() => onOwnerFilterChange("mine")}>
          My Work
        </FilterPill>
        <FilterPill active={ownerFilter === "unassigned"} onClick={() => onOwnerFilterChange("unassigned")}>
          No Owner
        </FilterPill>
        <FilterPill active={ownerFilter === "all"} onClick={() => onOwnerFilterChange("all")}>
          All
        </FilterPill>
      </div>

      <div className="flex flex-wrap items-center gap-2">
        {STATUS_FILTERS.map((filter) => (
          <FilterPill
            key={filter.id}
            active={statusFilter === filter.id}
            onClick={() => onStatusFilterChange(filter.id)}
          >
            {filter.id === "all" || filter.id === "active"
              ? filter.label
              : getWorkOrderDisplayStatusMeta(filter.id as WorkOrderDisplayStatus).filterLabel}
          </FilterPill>
        ))}
      </div>
    </div>
  );
}

function FilterPill({
  active,
  onClick,
  children,
}: {
  active: boolean;
  onClick: () => void;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={cn(
        "rounded-md px-3 py-1 text-sm font-medium transition-colors",
        active ? factoryFilterPillActiveClassName : factoryFilterPillInactiveClassName,
      )}
    >
      {children}
    </button>
  );
}
