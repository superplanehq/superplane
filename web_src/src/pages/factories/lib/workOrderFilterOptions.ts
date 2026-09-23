import type { FactoriesFactoryIntake, FactoriesFactoryLine } from "@/api-client";
import type { WorkOrderFilterDimension } from "./useWorkOrderListState";
import { lineIntakeSourceForApiSource } from "../pages/lineIntakeModel";
import {
  CREATED_MANUALLY,
  INTAKE_PRESENTATION,
  type SplitRunIntakeKind,
} from "../pages/work-order-split-run/splitRunSource";
import {
  MANUAL_FILTER_VALUE,
  UNASSIGNED_FILTER_VALUE,
  WORK_ORDER_FILTER_LABELS,
  WORK_ORDER_FILTER_LABEL_META,
  type WorkOrderFilters,
  type WorkOrderListEntry,
} from "./workOrderListModel";
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

export function buildLabelFilterOptions(): WorkOrderFilterOption[] {
  return WORK_ORDER_FILTER_LABELS.map((label) => ({
    value: label,
    label: WORK_ORDER_FILTER_LABEL_META[label].label,
  }));
}

export function buildLineFilterOptions(lines: FactoriesFactoryLine[]): WorkOrderFilterOption[] {
  return lines
    .filter((line): line is FactoriesFactoryLine & { id: string } => Boolean(line.id))
    .map((line) => ({ value: line.id, label: line.name?.trim() || "Untitled line" }));
}

/** People on at least one task or in extraPeople, sorted by name, with No Owner first. */
export function buildAssigneeFilterOptions(
  entries: WorkOrderListEntry[],
  extraPeople: Array<{ id?: string; name?: string }> = [],
): WorkOrderFilterOption[] {
  const byId = new Map<string, string>();
  for (const person of extraPeople) {
    if (person.id) {
      byId.set(person.id, person.name?.trim() || "Unknown");
    }
  }
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
 * Sources the factory can receive tasks from, plus Created manually.
 * Configured intakes drive the list so a tool with no tasks still appears.
 * Two intakes on the same tool collapse to one entry.
 */
export function buildSourceFilterOptions(
  intakes: FactoriesFactoryIntake[],
  entries: WorkOrderListEntry[],
): WorkOrderFilterOption[] {
  const ids = new Set<string>();
  for (const intake of intakes) {
    const source = lineIntakeSourceForApiSource(intake.source);
    if (source) {
      ids.add(source.id);
    }
  }
  for (const entry of entries) {
    ids.add(entry.sourceId);
  }
  ids.delete(MANUAL_FILTER_VALUE);

  const tools = [...ids]
    .map((value) => ({ value, label: sourceFilterLabel(value) }))
    .sort((a, b) => a.label.localeCompare(b.label));

  return [...tools, { value: MANUAL_FILTER_VALUE, label: CREATED_MANUALLY }];
}

export function sourceFilterLabel(sourceId: string): string {
  if (sourceId === MANUAL_FILTER_VALUE) {
    return CREATED_MANUALLY;
  }
  if (isSplitRunIntakeKind(sourceId)) {
    return INTAKE_PRESENTATION[sourceId].name;
  }
  return sourceId;
}

function isSplitRunIntakeKind(value: string): value is SplitRunIntakeKind {
  return value in INTAKE_PRESENTATION;
}

/**
 * Chip labels for every applied filter. Reads the labels back from the same
 * option lists the menu renders, so a chip can never disagree with the menu
 * entry that created it.
 */
export function buildWorkOrderFilterChips(
  filters: WorkOrderFilters,
  options: {
    lines: WorkOrderFilterOption[];
    sources: WorkOrderFilterOption[];
    assignees: WorkOrderFilterOption[];
  },
): WorkOrderFilterChip[] {
  const lineLabels = toLabelMap(options.lines);
  const sourceLabels = toLabelMap(options.sources);
  const assigneeLabels = toLabelMap(options.assignees);

  return [
    ...filters.statuses.map((status) => ({
      dimension: "statuses" as const,
      value: status,
      label: `Status is ${getWorkOrderDisplayStatusMeta(status).filterLabel}`,
    })),
    ...filters.labels.map((label) => ({
      dimension: "labels" as const,
      value: label,
      label: `Label is ${WORK_ORDER_FILTER_LABEL_META[label].label}`,
    })),
    ...filters.lineIds.map((lineId) => ({
      dimension: "lineIds" as const,
      value: lineId,
      label: `Line is ${lineLabels.get(lineId) ?? "Unknown line"}`,
    })),
    ...filters.sourceIds.map((sourceId) => ({
      dimension: "sourceIds" as const,
      value: sourceId,
      label:
        sourceId === MANUAL_FILTER_VALUE
          ? CREATED_MANUALLY
          : `Source is ${sourceLabels.get(sourceId) ?? sourceFilterLabel(sourceId)}`,
    })),
    ...filters.assigneeIds.map((assigneeId) => ({
      dimension: "assigneeIds" as const,
      value: assigneeId,
      label:
        assigneeId === UNASSIGNED_FILTER_VALUE ? "No Owner" : `Owner is ${assigneeLabels.get(assigneeId) ?? "Unknown"}`,
    })),
  ];
}

function toLabelMap(options: WorkOrderFilterOption[]): Map<string, string> {
  return new Map(options.map((option) => [option.value, option.label]));
}
