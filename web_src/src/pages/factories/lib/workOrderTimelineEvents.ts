import type {
  FactoriesWorkOrder,
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
  SuperplaneUsersUser,
} from "@/api-client";
import { formatDuration } from "@/lib/duration";
import {
  buildOrgUserDisplayMap,
  createOrgUserDisplayLookup,
  resolveOrgUserDisplay,
  type OrgUserDisplay,
  type OrgUserDisplayLookup,
} from "@/lib/orgUserDisplay";
import { buildWorkOrderTimelineViewFromEvents } from "./workOrderTimelineFromEvents";

export type WorkOrderTimelineEventKind =
  | "created"
  | "dispatched"
  | "assigned"
  | "statusChanged"
  | "commented"
  | "artifactAdded"
  | "closed";
export type UserNameLookup = (userId: string | undefined) => string | undefined;
export type { OrgUserDisplayLookup };

export interface WorkOrderTimelineStep {
  id: string;
  stepName: string;
  /** Latest meaningful timestamp — finished time when done, otherwise started. */
  at: string;
  startedAt: string;
  finishedAt?: string;
  execution: FactoriesWorkOrderExecution;
}

export interface WorkOrderTimelineAssigneeChange {
  assignedUserIds: string[];
  unassignedUserIds: string[];
}

export interface WorkOrderTimelineStatusChange {
  fromState?: string;
  toState: string;
  fromResult?: string;
  toResult?: string;
}

export interface WorkOrderTimelineCommentAutomation {
  nodeId?: string;
  nodeName?: string;
  appId?: string;
  appName?: string;
}

export interface WorkOrderTimelineComment {
  body: string;
  authorKind?: string;
  automation?: WorkOrderTimelineCommentAutomation;
}

export interface WorkOrderTimelineArtifact {
  id?: string;
  type: string;
  url?: string;
  title?: string;
  body?: string;
}

export interface WorkOrderTimelineEvent {
  id: string;
  kind: WorkOrderTimelineEventKind;
  at: string;
  actorUserId?: string;
  actorName?: string;
  assigneeChange?: WorkOrderTimelineAssigneeChange;
  statusChange?: WorkOrderTimelineStatusChange;
  comment?: WorkOrderTimelineComment;
  artifact?: WorkOrderTimelineArtifact;
  title: string;
  lineName?: string;
  steps?: WorkOrderTimelineStep[];
}

export interface WorkOrderTimelineViewModel {
  events: WorkOrderTimelineEvent[];
}

export function buildWorkOrderUserNameLookup(users: SuperplaneUsersUser[], order: FactoriesWorkOrder): UserNameLookup {
  const resolveUserDisplay = buildWorkOrderUserDisplayLookup(users, order);
  return (userId) => resolveUserDisplay(userId)?.name;
}

export function buildWorkOrderUserDisplayLookup(
  users: SuperplaneUsersUser[],
  order: FactoriesWorkOrder,
): OrgUserDisplayLookup {
  const usersById = buildOrgUserDisplayMap(users);
  addOrderFallbackNames(usersById, order);
  return createOrgUserDisplayLookup(usersById);
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

function addOrderFallbackNames(usersById: Map<string, OrgUserDisplay>, order: FactoriesWorkOrder): void {
  registerOrderUserFallback(usersById, order.createdBy?.id, order.createdBy?.name);

  for (const assignee of order.assignees ?? []) {
    registerOrderUserFallback(usersById, assignee.id, assignee.name);
  }
}

function registerOrderUserFallback(
  usersById: Map<string, OrgUserDisplay>,
  id: string | undefined,
  name: string | undefined,
): void {
  if (!id || usersById.has(id) || !name?.trim()) {
    return;
  }

  usersById.set(id, resolveOrgUserDisplay(new Map(), id, name)!);
}
