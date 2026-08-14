import type { WorkOrderListEntry, WorkOrderScope } from "../../lib/workOrderListModel";
import type { WorkOrderDisplayStatus } from "../../lib/workOrderProgress";
import type { FactoryMission } from "./missionMocks";

export type MissionDisplayStatus = "draft" | "active" | "completed" | "closed";
export type MissionCloseReason = "completed" | "closed";

export interface MissionProgress {
  completed: number;
  total: number;
  label: string;
  percent: number;
}

export interface MissionRailItem {
  mission: FactoryMission;
  progress: MissionProgress;
  status: MissionDisplayStatus;
}

const TERMINAL_WORK_ORDER_STATUSES: WorkOrderDisplayStatus[] = ["completed", "failed", "cancelled"];

export const MISSION_STATUS_META: Record<
  MissionDisplayStatus,
  { label: string; textClass: string; dotClass: string; className: string }
> = {
  draft: {
    label: "Draft",
    textClass: "text-[color:var(--status-draft-fg)]",
    dotClass: "bg-[color:var(--status-draft-dot)]",
    className:
      "border-[color:var(--status-draft-border)] bg-[color:var(--status-draft-bg)] text-[color:var(--status-draft-fg)]",
  },
  active: {
    label: "Active",
    textClass: "text-[color:var(--status-running-fg)]",
    dotClass: "bg-[color:var(--status-running-dot)]",
    className:
      "border-[color:var(--status-running-border)] bg-[color:var(--status-running-bg)] text-[color:var(--status-running-fg)]",
  },
  completed: {
    label: "Completed",
    textClass: "text-[color:var(--status-completed-fg)]",
    dotClass: "bg-[color:var(--status-completed-dot)]",
    className:
      "border-[color:var(--status-completed-border)] bg-[color:var(--status-completed-bg)] text-[color:var(--status-completed-fg)]",
  },
  closed: {
    label: "Closed",
    textClass: "text-[color:var(--status-cancelled-fg)]",
    dotClass: "bg-[color:var(--status-cancelled-dot)]",
    className:
      "border-[color:var(--status-cancelled-border)] bg-[color:var(--status-cancelled-bg)] text-[color:var(--status-cancelled-fg)]",
  },
};

export function applyMissionFilter(
  entries: WorkOrderListEntry[],
  missionId: string | null,
  missionByWorkOrderId: Record<string, string>,
): WorkOrderListEntry[] {
  if (!missionId) {
    return entries;
  }
  return entries.filter((entry) => missionByWorkOrderId[entry.id] === missionId);
}

export function findMissionById(missions: FactoryMission[], missionId: string): FactoryMission | null {
  return missions.find((mission) => mission.id === missionId) ?? null;
}

/** Returns a new map. A work order belongs to one mission, or to none. */
export function assignWorkOrderToMission(
  missionByWorkOrderId: Record<string, string>,
  workOrderId: string,
  missionId: string | null,
): Record<string, string> {
  const next = { ...missionByWorkOrderId };
  if (!missionId) {
    delete next[workOrderId];
    return next;
  }
  next[workOrderId] = missionId;
  return next;
}

export function resolveMissionForWorkOrder(
  missions: FactoryMission[],
  missionByWorkOrderId: Record<string, string>,
  workOrderId: string,
): FactoryMission | null {
  const missionId = missionByWorkOrderId[workOrderId];
  if (!missionId) {
    return null;
  }
  return findMissionById(missions, missionId);
}

function membersForMission(
  entries: WorkOrderListEntry[],
  missionId: string,
  missionByWorkOrderId: Record<string, string>,
): WorkOrderListEntry[] {
  return entries.filter((entry) => missionByWorkOrderId[entry.id] === missionId);
}

export function resolveMissionProgress(
  entries: WorkOrderListEntry[],
  missionId: string,
  missionByWorkOrderId: Record<string, string>,
): MissionProgress {
  const members = membersForMission(entries, missionId, missionByWorkOrderId);
  const completed = members.filter((entry) => entry.displayStatus === "completed").length;
  const total = members.length;
  const percent = total === 0 ? 0 : Math.round((completed / total) * 100);
  return {
    completed,
    total,
    label: `${completed} of ${total} completed`,
    percent,
  };
}

/**
 * Status comes only from member work orders.
 * Empty → draft. Any unfinished work → active. Only finished work → completed.
 */
export function resolveMissionStatus(
  entries: WorkOrderListEntry[],
  missionId: string,
  missionByWorkOrderId: Record<string, string>,
): MissionDisplayStatus {
  const members = membersForMission(entries, missionId, missionByWorkOrderId);
  if (members.length === 0) {
    return "draft";
  }
  if (members.every((entry) => TERMINAL_WORK_ORDER_STATUSES.includes(entry.displayStatus))) {
    return "completed";
  }
  return "active";
}

/** Manual complete or close wins over the work-order derived status. */
export function resolveMissionDisplayStatus(
  derived: MissionDisplayStatus,
  closeReason?: MissionCloseReason,
): MissionDisplayStatus {
  if (closeReason === "completed") {
    return "completed";
  }
  if (closeReason === "closed") {
    return "closed";
  }
  return derived;
}

export function isMissionHiddenFromActive(
  item: MissionRailItem,
  closedByMissionId: Record<string, MissionCloseReason>,
): boolean {
  return item.status === "completed" || item.status === "closed" || Boolean(closedByMissionId[item.mission.id]);
}

/** Follows the Work Orders page scope. Active hides finished missions. */
export function applyMissionScope(
  items: MissionRailItem[],
  scope: WorkOrderScope,
  closedByMissionId: Record<string, MissionCloseReason>,
  currentUserId?: string,
): MissionRailItem[] {
  if (scope === "my") {
    if (!currentUserId) {
      return [];
    }
    return items.filter((item) => item.mission.assignee.id === currentUserId);
  }
  if (scope === "active") {
    return items.filter((item) => !isMissionHiddenFromActive(item, closedByMissionId));
  }
  return items;
}

/** Progress on each card uses every work order in the mission, not the current filters. */
export function buildMissionRailItems(
  missions: FactoryMission[],
  entries: WorkOrderListEntry[],
  missionByWorkOrderId: Record<string, string>,
  closedByMissionId: Record<string, MissionCloseReason> = {},
): MissionRailItem[] {
  return missions.map((mission) => ({
    mission,
    progress: resolveMissionProgress(entries, mission.id, missionByWorkOrderId),
    status: resolveMissionDisplayStatus(
      resolveMissionStatus(entries, mission.id, missionByWorkOrderId),
      closedByMissionId[mission.id],
    ),
  }));
}

export const DEFAULT_VISIBLE_MISSION_COUNT = 4;

export function sortMissionItems(items: MissionRailItem[]): MissionRailItem[] {
  return [...items].sort((left, right) => {
    if (right.progress.total !== left.progress.total) {
      return right.progress.total - left.progress.total;
    }
    return left.mission.name.localeCompare(right.mission.name);
  });
}

/** Prefer missions that have work. Cap the default list so the page stays quiet. */
export function visibleMissionItems(
  items: MissionRailItem[],
  expanded: boolean,
): { visible: MissionRailItem[]; hiddenCount: number } {
  const sorted = sortMissionItems(items);
  if (expanded || sorted.length <= DEFAULT_VISIBLE_MISSION_COUNT) {
    return { visible: sorted, hiddenCount: 0 };
  }

  const withWork = sorted.filter((item) => item.progress.total > 0);
  const preferred = withWork.length > 0 ? withWork : sorted;
  const visible = preferred.slice(0, DEFAULT_VISIBLE_MISSION_COUNT);
  return { visible, hiddenCount: sorted.length - visible.length };
}
