import type { FactoriesWorkOrderEvent, FactoriesWorkOrderResult } from "@/api-client";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { UNKNOWN_ORG_USER_NAME } from "@/lib/orgUserDisplay";
import type {
  UserNameLookup,
  WorkOrderTimelineAutomationActor,
  WorkOrderTimelineEvent,
  WorkOrderTimelineViewModel,
} from "./workOrderTimelineEvents";
import {
  appendStepExecutionEvent,
  type DispatchBatchContext,
  type LineStepExecutionPayload,
} from "./workOrderTimelineStepBuilder";

const UNKNOWN_MEMBER_LABEL = UNKNOWN_ORG_USER_NAME;

interface EventUserRef {
  id?: string;
}

interface EventAutomationRefPayload {
  nodeId?: string;
  nodeName?: string;
  appId?: string;
  appName?: string;
  lineId?: string;
  lineName?: string;
  stepIndex?: number;
  stepName?: string;
}

interface EventCommentAuthorPayload {
  kind?: string;
  userId?: string;
  automation?: EventAutomationRefPayload;
}

interface EventArtifactPayload {
  id?: string;
  type?: string;
  url?: string;
  title?: string;
  data?: Record<string, unknown>;
}

interface EventPayload extends LineStepExecutionPayload {
  user?: EventUserRef;
  automation?: EventAutomationRefPayload;
  assigned?: EventUserRef[];
  unassigned?: EventUserRef[];
  order?: {
    result?: string;
  };
  result?: string;
  fromState?: string;
  toState?: string;
  fromResult?: string;
  toResult?: string;
  body?: string;
  author?: EventCommentAuthorPayload;
  artifact?: EventArtifactPayload;
}

interface TimelineBuildState {
  events: WorkOrderTimelineEvent[];
  dispatchBatchByLine: Map<string, WorkOrderTimelineEvent>;
}

const WORK_ORDER_EVENT_TYPE_ORDER: Record<string, number> = {
  "order.opened": 10,
  "order.status.updated": 15,
  "order.assignees.updated": 20,
  "step.execution.created": 30,
  "step.execution.finished": 40,
  "order.comment.added": 45,
  "order.artifact.added": 47,
  "order.closed": 50,
};

function compareWorkOrderEventsChronologically(left: FactoriesWorkOrderEvent, right: FactoriesWorkOrderEvent): number {
  const timeDiff = Date.parse(left.timestamp ?? "") - Date.parse(right.timestamp ?? "");
  if (timeDiff !== 0) {
    return timeDiff;
  }

  const leftOrder = WORK_ORDER_EVENT_TYPE_ORDER[left.type ?? ""] ?? 0;
  const rightOrder = WORK_ORDER_EVENT_TYPE_ORDER[right.type ?? ""] ?? 0;
  if (leftOrder !== rightOrder) {
    return leftOrder - rightOrder;
  }

  return (left.type ?? "").localeCompare(right.type ?? "");
}

export function buildWorkOrderTimelineViewFromEvents(
  apiEvents: FactoriesWorkOrderEvent[],
  resolveUserName?: UserNameLookup,
): WorkOrderTimelineViewModel {
  const eventsAsc = [...apiEvents].sort(compareWorkOrderEventsChronologically);

  const state = createTimelineBuildState();

  for (const [index, apiEvent] of eventsAsc.entries()) {
    applyApiEventToTimeline(state, index, apiEvent, resolveUserName);
  }

  return { events: state.events };
}

function createTimelineBuildState(): TimelineBuildState {
  return {
    events: [],
    dispatchBatchByLine: new Map(),
  };
}

function applyApiEventToTimeline(
  state: TimelineBuildState,
  index: number,
  apiEvent: FactoriesWorkOrderEvent,
  resolveUserName?: UserNameLookup,
): void {
  const payload = (apiEvent.event ?? {}) as EventPayload;
  const at = apiEvent.timestamp ?? "";

  switch (apiEvent.type) {
    case "order.opened":
      appendOpenedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "order.status.updated":
      appendStatusUpdatedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "order.assignees.updated":
      appendAssigneesUpdatedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "step.execution.created":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.created");
      return;
    case "step.execution.finished":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.finished");
      return;
    case "order.comment.added":
      appendCommentEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "order.artifact.added":
      appendArtifactEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "order.closed":
      appendClosedEvent(state.events, index, payload, at, resolveUserName);
  }
}

function toDispatchBatchContext(state: TimelineBuildState): DispatchBatchContext {
  return {
    timelineEvents: state.events,
    dispatchBatchByLine: state.dispatchBatchByLine,
  };
}

function appendOpenedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  // `order.opened` now fires on the first `draft → open` transition, not at
  // creation. Emit it as a status change so we don't duplicate the "created"
  // entry that `order.status.updated ("" → draft)` already produced. The
  // originating run / app are attached for attribution when present.
  events.push({
    id: `opened-${index}`,
    kind: "statusChanged",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: toAutomationActor(payload.automation),
    sourceRunId: payload.run?.id,
    sourceAppId: payload.app?.id,
    statusChange: { fromState: "draft", toState: "open", fromResult: "", toResult: "" },
    title: "opened this work order",
  });
}

function appendAssigneesUpdatedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  events.push({
    id: `assignees-updated-${index}`,
    kind: "assigned",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    assigneeChange: {
      assignedUserIds: (payload.assigned ?? []).map((user) => user.id).filter((id): id is string => Boolean(id)),
      unassignedUserIds: (payload.unassigned ?? []).map((user) => user.id).filter((id): id is string => Boolean(id)),
    },
    title: describeAssigneesUpdated(payload, resolveUserName),
  });
}

function appendStatusUpdatedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const toState = payload.toState ?? "";
  if (!toState) {
    return;
  }

  const fromState = payload.fromState ?? "";
  const toResult = payload.toResult ?? "";
  const fromResult = payload.fromResult ?? "";

  // Creation is emitted as `status.updated` with empty `fromState`.
  if (!fromState) {
    events.push({
      id: `created-${index}`,
      kind: "created",
      at,
      actorUserId: payload.user?.id,
      actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
      actorAutomation: toAutomationActor(payload.automation),
      title: "created this work order",
    });
    return;
  }

  // Skip status.updated when a coarse `order.opened` / `order.closed`
  // event already covers the same change (one entry per transition).
  if (toState === "open" && fromState === "draft") {
    return;
  }
  if (toState === "closed") {
    return;
  }

  events.push({
    id: `status-updated-${index}`,
    kind: "statusChanged",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: toAutomationActor(payload.automation),
    statusChange: { fromState, toState, fromResult, toResult },
    title: describeStatusTransition(fromState, toState),
  });
}

function describeStatusTransition(fromState: string, toState: string): string {
  if (!fromState) {
    return `set status to ${humanizeState(toState)}`;
  }
  if (fromState === "closed" && toState === "open") {
    return `reopened as ${humanizeState(toState)}`;
  }
  return `moved ${humanizeState(fromState)} → ${humanizeState(toState)}`;
}

function humanizeState(state: string): string {
  switch (state) {
    case "draft":
      return "Draft";
    case "open":
      return "Open";
    case "closed":
      return "Closed";
    default:
      return state || "Unknown";
  }
}

function appendCommentEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const body = (payload.body ?? "").trim();
  if (!body) {
    return;
  }

  const author = payload.author ?? {};
  const automationActor = toAutomationActor(author.automation);
  events.push({
    id: `comment-${index}`,
    kind: "commented",
    at,
    actorUserId: author.userId ?? payload.user?.id,
    actorName: resolveUserDisplayName(author.userId ?? payload.user?.id, resolveUserName),
    actorAutomation: automationActor,
    comment: {
      body,
      authorKind: author.kind,
      automation: automationActor,
    },
    title: "commented",
  });
}

function appendArtifactEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const artifact = payload.artifact;
  if (!artifact?.type) {
    return;
  }

  events.push({
    id: `artifact-${index}`,
    kind: "artifactAdded",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: toAutomationActor(payload.automation),
    artifact: {
      id: artifact.id,
      type: artifact.type,
      url: artifact.url,
      title: artifact.title,
      data: artifact.data,
    },
    title: describeArtifactAdded(artifact),
  });
}

// Short-form artifact label ("PR", "note") for event descriptions.
// See timeline/authorLabels.ts for the long form used in bodies.
const ARTIFACT_KIND_SHORT_LABEL: Record<string, string> = {
  pr: "PR",
  markdown: "note",
};

function formatArtifactKindShort(type: string | undefined): string {
  if (!type) {
    return "artifact";
  }
  return ARTIFACT_KIND_SHORT_LABEL[type] ?? "artifact";
}

function describeArtifactAdded(artifact: EventArtifactPayload): string {
  const label = artifact.title?.trim() || artifact.url?.trim();
  const type = formatArtifactKindShort(artifact.type);
  return label ? `attached ${type}: ${label}` : `attached ${type}`;
}

// Event payload result → proto enum the presentation helpers expect.
const CLOSED_RESULT_TO_PROTO: Record<string, FactoriesWorkOrderResult> = {
  completed: "RESULT_COMPLETED",
  rejected: "RESULT_REJECTED",
  failed: "RESULT_FAILED",
};

function appendClosedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const closedResult = payload.result ?? payload.order?.result;
  const result = formatWorkOrderResult(closedResult ? CLOSED_RESULT_TO_PROTO[closedResult] : undefined);

  events.push({
    id: `closed-${index}`,
    kind: "closed",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: toAutomationActor(payload.automation),
    title: result ? `closed as ${result.toLowerCase()}` : "closed this work order",
  });
}

function toAutomationActor(
  payload: EventAutomationRefPayload | undefined,
): WorkOrderTimelineAutomationActor | undefined {
  if (!payload) {
    return undefined;
  }

  const anySet =
    payload.nodeId || payload.nodeName || payload.appId || payload.appName || payload.lineId || payload.lineName;
  if (!anySet) {
    return undefined;
  }

  return {
    nodeId: payload.nodeId,
    nodeName: payload.nodeName,
    appId: payload.appId,
    appName: payload.appName,
    lineId: payload.lineId,
    lineName: payload.lineName,
    stepIndex: payload.stepIndex,
    stepName: payload.stepName,
  };
}

function resolveUserDisplayName(userId: string | undefined, resolveUserName?: UserNameLookup): string | undefined {
  if (!userId) {
    return undefined;
  }

  return resolveUserName?.(userId) ?? UNKNOWN_MEMBER_LABEL;
}

function describeAssigneesUpdated(payload: EventPayload, resolveUserName?: UserNameLookup): string {
  const actorId = payload.user?.id;
  const assignedIds = (payload.assigned ?? []).map((user) => user.id).filter((id): id is string => Boolean(id));
  const assignedOthers = actorId ? assignedIds.filter((userId) => userId !== actorId) : assignedIds;
  const selfAssigned = Boolean(actorId && assignedIds.includes(actorId));
  const unassignedNames = formatEventUserNames(payload.unassigned, resolveUserName);
  const parts: string[] = [];

  if (selfAssigned && assignedOthers.length === 0) {
    parts.push("self-assigned");
  } else if (selfAssigned && assignedOthers.length > 0) {
    const otherNames = formatEventUserNames(
      assignedOthers.map((id) => ({ id })),
      resolveUserName,
    );
    parts.push(otherNames ? `self-assigned and assigned ${otherNames}` : "self-assigned");
  } else {
    const assignedNames = formatEventUserNames(payload.assigned, resolveUserName);
    if (assignedNames) {
      parts.push(`assigned ${assignedNames}`);
    }
  }

  if (unassignedNames) {
    parts.push(`unassigned ${unassignedNames}`);
  }

  if (parts.length === 0) {
    return "updated assignees";
  }

  return parts.join(" and ");
}

function formatEventUserNames(users: EventUserRef[] | undefined, resolveUserName?: UserNameLookup): string | null {
  if (!users?.length) {
    return null;
  }

  const names = users.map((user) => resolveUserDisplayName(user.id, resolveUserName) ?? UNKNOWN_MEMBER_LABEL);
  return formatNameList(names);
}

function formatNameList(names: string[]): string {
  if (names.length === 0) {
    return "";
  }

  if (names.length === 1) {
    return names[0];
  }

  if (names.length === 2) {
    return `${names[0]} and ${names[1]}`;
  }

  return `${names.slice(0, -1).join(", ")}, and ${names[names.length - 1]}`;
}
