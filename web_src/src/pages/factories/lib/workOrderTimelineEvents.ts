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
  comments?: WorkOrderTimelineStepComment[];
  artifacts?: WorkOrderTimelineArtifact[];
  execution: FactoriesWorkOrderExecution;
}

export interface WorkOrderTimelineStepComment {
  body: string;
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

export interface WorkOrderTimelineAutomationActor {
  nodeId?: string;
  nodeName?: string;
  appId?: string;
  appName?: string;
  lineId?: string;
  lineName?: string;
  stepIndex?: number;
  stepName?: string;
}

export interface WorkOrderTimelineComment {
  body: string;
  authorKind?: string;
  automation?: WorkOrderTimelineAutomationActor;
}

export interface WorkOrderTimelineArtifact {
  id?: string;
  type: string;
  data?: Record<string, unknown>;
}

export interface WorkOrderTimelineEvent {
  id: string;
  kind: WorkOrderTimelineEventKind;
  at: string;
  actorUserId?: string;
  actorName?: string;
  actorAutomation?: WorkOrderTimelineAutomationActor;
  sourceRunId?: string;
  sourceAppId?: string;
  assigneeChange?: WorkOrderTimelineAssigneeChange;
  statusChange?: WorkOrderTimelineStatusChange;
  comment?: WorkOrderTimelineComment;
  artifact?: WorkOrderTimelineArtifact;
  title: string;
  lineId?: string;
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
  executions: FactoriesWorkOrderExecution[] = [],
): WorkOrderTimelineViewModel {
  const timeline = buildWorkOrderTimelineViewFromEvents(apiEvents ?? [], resolveUserName);
  const executionsByRunID = new Map(
    executions.flatMap((execution) => (execution.run?.id ? [[execution.run.id, execution]] : [])),
  );

  for (const event of timeline.events) {
    for (const step of event.steps ?? []) {
      const execution = step.execution.id ? executionsByRunID.get(step.execution.id) : undefined;
      if (execution) {
        step.execution = { ...step.execution, ...execution };
      }
    }
  }

  return timeline;
}

export function formatStepExecutionDuration(step: WorkOrderTimelineStep): string | null {
  if (!step.startedAt || !step.finishedAt) {
    return null;
  }

  const durationMs = Date.parse(step.finishedAt) - Date.parse(step.startedAt);
  if (!Number.isFinite(durationMs) || durationMs <= 0) {
    return null;
  }

  const formatted = formatDuration(durationMs, { precision: "second" });
  return formatted || null;
}

function addOrderFallbackNames(usersById: Map<string, OrgUserDisplay>, order: FactoriesWorkOrder): void {
  const creator = order.createdBy?.user;
  registerOrderUserFallback(usersById, creator?.id, creator?.name);

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
