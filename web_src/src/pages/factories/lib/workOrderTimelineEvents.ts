import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
  SuperplaneUsersUser,
} from "@/api-client";
import { formatDuration } from "@/lib/duration";
import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

export type WorkOrderTimelineEventKind = "created" | "dispatched" | "assigned";
export type UserNameLookup = (userId: string | undefined) => string | undefined;

export interface WorkOrderTimelineStep {
  id: string;
  stepName: string;
  /** Latest meaningful timestamp — finished time when done, otherwise started. */
  at: string;
  startedAt: string;
  finishedAt?: string;
  execution: FactoriesWorkOrderExecution;
}

export interface WorkOrderTimelineEvent {
  id: string;
  kind: WorkOrderTimelineEventKind;
  at: string;
  actorName?: string;
  title: string;
  lineName?: string;
  steps?: WorkOrderTimelineStep[];
}

export interface WorkOrderTimelineViewModel {
  events: WorkOrderTimelineEvent[];
  closedLabel: string | null;
  closedAt: string | null;
}

export function buildWorkOrderUserNameLookup(users: SuperplaneUsersUser[], order: FactoriesWorkOrder): UserNameLookup {
  const namesById = collectOrgUserNames(users);
  addOrderFallbackNames(namesById, order);
  return createUserNameResolver(namesById);
}

export function buildWorkOrderTimelineView(
  apiEvents: FactoriesWorkOrderEvent[] | undefined,
  resolveUserName?: UserNameLookup,
): WorkOrderTimelineViewModel {
  return buildWorkOrderTimelineViewFromEvents(apiEvents ?? [], resolveUserName);
}

export function formatStepExecutionDuration(step: WorkOrderTimelineStep): string | null {
  if (!step.startedAt || !step.finishedAt) {
    return null;
  }

  const durationMs = Date.parse(step.finishedAt) - Date.parse(step.startedAt);
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return null;
  }

  const formatted = formatDuration(durationMs);
  return formatted || null;
}

function collectOrgUserNames(users: SuperplaneUsersUser[]): Map<string, string> {
  const namesById = new Map<string, string>();

  for (const user of users) {
    registerUserName(namesById, user.metadata?.id, user.spec?.displayName?.trim() || user.metadata?.email?.trim());
  }

  return namesById;
}

function addOrderFallbackNames(namesById: Map<string, string>, order: FactoriesWorkOrder): void {
  registerUserName(namesById, order.createdBy?.id, order.createdBy?.name);

  for (const assignee of order.assignees ?? []) {
    registerUserName(namesById, assignee.id, assignee.name);
  }
}

function registerUserName(namesById: Map<string, string>, id: string | undefined, name: string | undefined): void {
  if (id && name?.trim()) {
    namesById.set(id, name.trim());
  }
}

function createUserNameResolver(namesById: Map<string, string>): UserNameLookup {
  return (userId) => {
    if (!userId) {
      return undefined;
    }

    return namesById.get(userId);
  };
}
