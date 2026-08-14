import type { FactoriesFactoryLine } from "@/api-client";
import type { WorkOrderFilterDimension } from "./useWorkOrderListState";
import { UNASSIGNED_FILTER_VALUE, type WorkOrderFilters, type WorkOrderListEntry } from "./workOrderListModel";
import { WORK_ORDER_DISPLAY_STATUSES, getWorkOrderDisplayStatusMeta } from "./workOrderProgress";

/** One selectable value in a Filter submenu. */
export interface WorkOrderFilterOption {
  value: string;
  label: string;
  /** Status dot class, only set for the status dimension. */
  dot?: string;
}

/** One applied filter, rendered as a removable chip under the title bar. */
export interface WorkOrderFilterChip {
  dimension: WorkOrderFilterDimension;
  value: string;
  label: string;
}

export function buildStatusFilterOptions(): WorkOrderFilterOption[] {
  return WORK_ORDER_DISPLAY_STATUSES.map((status) => {
    const meta = getWorkOrderDisplayStatusMeta(status);
    return { value: status, label: meta.filterLabel, dot: meta.dotClassName };
  });
}

export function buildLineFilterOptions(lines: FactoriesFactoryLine[]): WorkOrderFilterOption[] {
  return lines
    .filter((line): line is FactoriesFactoryLine & { id: string } => Boolean(line.id))
    .map((line) => ({ value: line.id, label: line.name?.trim() || "Untitled line" }));
}

/** People on at least one work order, sorted by name, with No Owner first. */
export function buildAssigneeFilterOptions(entries: WorkOrderListEntry[]): WorkOrderFilterOption[] {
  const byId = new Map<string, string>();
  for (const entry of entries) {
    for (const assignee of entry.order.assignees ?? []) {
      if (assignee.id) {
        byId.set(assignee.id, assignee.name?.trim() || "Unknown");
      }
    }
  }

  const people = [...byId.entries()]
    .map(([value, label]) => ({ value, label }))
    .sort((a, b) => a.label.localeCompare(b.label));

  return [{ value: UNASSIGNED_FILTER_VALUE, label: "No Owner" }, ...people];
}

/**
 * Chip labels for every applied filter. Reads the labels back from the same
 * option lists the menu renders, so a chip can never disagree with the menu
 * entry that created it.
 */
export function buildWorkOrderFilterChips(
  filters: WorkOrderFilters,
  options: { lines: WorkOrderFilterOption[]; assignees: WorkOrderFilterOption[] },
): WorkOrderFilterChip[] {
  const lineLabels = toLabelMap(options.lines);
  const assigneeLabels = toLabelMap(options.assignees);

  return [
    ...filters.statuses.map((status) => ({
      dimension: "statuses" as const,
      value: status,
      label: `Status is ${getWorkOrderDisplayStatusMeta(status).filterLabel}`,
    })),
    ...filters.lineIds.map((lineId) => ({
      dimension: "lineIds" as const,
      value: lineId,
      label: `Line is ${lineLabels.get(lineId) ?? "Unknown line"}`,
    })),
    ...filters.assigneeIds.map((assigneeId) => ({
      dimension: "assigneeIds" as const,
      value: assigneeId,
      label:
        assigneeId === UNASSIGNED_FILTER_VALUE
          ? "No Owner"
          : `Owner is ${assigneeLabels.get(assigneeId) ?? "Unknown"}`,
    })),
  ];
}

function toLabelMap(options: WorkOrderFilterOption[]): Map<string, string> {
  return new Map(options.map((option) => [option.value, option.label]));
}
