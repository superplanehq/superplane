import type { FactoriesWorkOrderEvent, FactoriesWorkOrderResult } from "@/api-client";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { isPullRequestArtifactType, pullRequestFromEventPayload } from "./workOrderPullRequest";
import {
  describeArtifactAdded,
  describeAssigneesUpdated,
  describePullRequestEvent,
  findAutomationStep,
  resolveUserDisplayName,
  toAutomationActor,
} from "./workOrderTimelineFromEvents.helpers";
import type {
  UserNameLookup,
  WorkOrderTimelineEvent,
  WorkOrderTimelineEventKind,
  WorkOrderTimelineStepComment,
  WorkOrderTimelineViewModel,
} from "./workOrderTimelineEvents";
import {
  appendStepExecutionEvent,
  type DispatchBatchContext,
  type LineStepExecutionPayload,
} from "./workOrderTimelineStepBuilder";

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
  data?: Record<string, unknown>;
}

interface EventPullRequestPayload {
  id?: string;
  provider?: string;
  repository?: string;
  number?: number | string;
  url?: string;
  title?: string;
  state?: string;
}

interface EventCheckPayload {
  name?: string;
  score?: number;
  maxScore?: number;
  format?: "fraction" | "percent" | "boolean";
  previousScore?: number;
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
  pullRequest?: EventPullRequestPayload;
  check?: EventCheckPayload;
}

interface TimelineBuildState {
  events: WorkOrderTimelineEvent[];
  dispatchBatchByLine: Map<string, WorkOrderTimelineEvent>;
}

const WORK_ORDER_EVENT_TYPE_ORDER: Record<string, number> = {
  "order.status.updated": 15,
  "order.assignees.updated": 20,
  "step.execution.queued": 25,
  "step.execution.created": 30,
  "step.execution.finished": 40,
  "order.comment.added": 45,
  "order.check.reported": 46,
  "order.artifact.added": 47,
  "order.pull_request.added": 48,
  "order.pull_request.updated": 49,
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
    case "order.status.updated":
      appendStatusUpdatedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "order.assignees.updated":
      appendAssigneesUpdatedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "step.execution.queued":
      appendStepQueuedEvent(state.events, index, payload, at);
      return;
    case "step.execution.created":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.created");
      return;
    case "step.execution.finished":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.finished");
      return;
    case "order.comment.added":
      appendCommentEvent(state, index, payload, at, resolveUserName);
      return;
    case "order.artifact.added":
      appendArtifactEvent(state, index, payload, at, resolveUserName);
      return;
    case "order.pull_request.added":
      appendPullRequestEvent(state, {
        index,
        payload,
        at,
        kind: "pullRequestAdded",
        resolveUserName,
      });
      return;
    case "order.pull_request.updated":
      appendPullRequestEvent(state, {
        index,
        payload,
        at,
        kind: "pullRequestUpdated",
        resolveUserName,
      });
      return;
    case "order.check.reported":
      appendCheckReportedEvent(state, index, payload, at);
  }
}

function toDispatchBatchContext(state: TimelineBuildState): DispatchBatchContext {
  return {
    timelineEvents: state.events,
    dispatchBatchByLine: state.dispatchBatchByLine,
  };
}

// The step is at its max parallelism: the work order waits in the step
// queue until a run completes and frees a slot.
function appendStepQueuedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
): void {
  const stepName = payload.stepName?.trim() || "Unnamed step";
  const lineName = payload.line?.name?.trim();
  events.push({
    id: `step-queued-${index}`,
    kind: "queued",
    at,
    lineId: payload.line?.id,
    lineName,
    title: `Queued at ${stepName} — waiting for a free slot`,
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

// Derive the visual lifecycle kind from the authoritative status transition.
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

  const kind = deriveStatusEventKind(fromState, toState);
  events.push({
    id: `${kind}-${index}`,
    kind,
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: toAutomationActor(payload.automation),
    sourceRunId: payload.run?.id,
    sourceAppId: payload.app?.id,
    statusChange: { fromState, toState, fromResult, toResult },
    title: describeStatusTransition(fromState, toState, toResult),
  });
}

function deriveStatusEventKind(fromState: string, toState: string): WorkOrderTimelineEventKind {
  if (!fromState) {
    return "created";
  }
  if (toState === "closed") {
    return "closed";
  }
  return "statusChanged";
}

function describeStatusTransition(fromState: string, toState: string, toResult: string): string {
  if (!fromState) {
    return "created this work order";
  }
  if (toState === "closed") {
    const result = formatWorkOrderResult(CLOSED_RESULT_TO_PROTO[toResult] ?? undefined);
    return result ? `closed as ${result.toLowerCase()}` : "closed this work order";
  }
  if (fromState === "draft" && toState === "open") {
    return "opened this work order";
  }
  if (fromState === "open" && toState === "draft") {
    return "moved this work order back to Draft";
  }
  if (fromState === "closed" && toState === "open") {
    return "reopened this work order";
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
  state: TimelineBuildState,
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
  const sourceRunId = payload.run?.id;
  const sourceAppId = automationActor?.appId ?? payload.app?.id;
  const stepComment: WorkOrderTimelineStepComment = { body };
  const step = findAutomationStep(state, automationActor);
  if (step) {
    step.comments = [...(step.comments ?? []), stepComment];
    return;
  }

  state.events.push({
    id: `comment-${index}`,
    kind: "commented",
    at,
    actorUserId: author.userId ?? payload.user?.id,
    actorName: resolveUserDisplayName(author.userId ?? payload.user?.id, resolveUserName),
    actorAutomation: automationActor,
    sourceRunId,
    sourceAppId,
    comment: {
      body,
      authorKind: author.kind,
      automation: automationActor,
    },
    title: "commented",
  });
}

function appendArtifactEvent(
  state: TimelineBuildState,
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const artifact = payload.artifact;
  if (!artifact?.type || isPullRequestArtifactType(artifact.type)) {
    return;
  }

  const timelineArtifact = {
    id: artifact.id,
    type: artifact.type,
    data: artifact.data,
  };
  const automationActor = toAutomationActor(payload.automation);
  const step = findAutomationStep(state, automationActor);
  if (step) {
    step.artifacts = [...(step.artifacts ?? []), timelineArtifact];
    return;
  }

  state.events.push({
    id: `artifact-${index}`,
    kind: "artifactAdded",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: automationActor,
    artifact: timelineArtifact,
    title: describeArtifactAdded(artifact),
  });
}

function appendPullRequestEvent(
  state: TimelineBuildState,
  args: {
    index: number;
    payload: EventPayload;
    at: string;
    kind: "pullRequestAdded" | "pullRequestUpdated";
    resolveUserName?: UserNameLookup;
  },
): void {
  const { index, payload, at, kind, resolveUserName } = args;
  if (!payload.pullRequest) {
    return;
  }

  const pullRequest = pullRequestFromEventPayload(payload.pullRequest);
  const automationActor = toAutomationActor(payload.automation);
  const step = findAutomationStep(state, automationActor);
  if (step) {
    step.pullRequests = [...(step.pullRequests ?? []), pullRequest];
    return;
  }

  state.events.push({
    id: `pull-request-${kind}-${index}`,
    kind,
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    actorAutomation: automationActor,
    sourceRunId: payload.run?.id,
    sourceAppId: automationActor?.appId ?? payload.app?.id,
    pullRequest,
    title: describePullRequestEvent(kind, pullRequest),
  });
}

// Check reports stay top-level: they come from dedicated automations
// (risk review, coverage), not from a dispatched line step.
function appendCheckReportedEvent(state: TimelineBuildState, index: number, payload: EventPayload, at: string): void {
  const check = payload.check;
  if (!check?.name || check.score === undefined || check.maxScore === undefined) {
    return;
  }

  const automationActor = toAutomationActor(payload.automation);
  // Same wording rule as CheckReportedEventBody: a re-report with an
  // unchanged score still reads as "reported".
  const isRescore = check.previousScore !== undefined && check.previousScore !== check.score;
  state.events.push({
    id: `check-${index}`,
    kind: "checkReported",
    at,
    actorAutomation: automationActor,
    sourceRunId: payload.run?.id,
    sourceAppId: automationActor?.appId ?? payload.app?.id,
    check: {
      name: check.name,
      score: check.score,
      maxScore: check.maxScore,
      format: check.format,
      previousScore: check.previousScore,
    },
    title: isRescore ? `re-scored ${check.name}` : `reported ${check.name}`,
  });
}

const CLOSED_RESULT_TO_PROTO: Record<string, FactoriesWorkOrderResult> = {
  completed: "RESULT_COMPLETED",
  rejected: "RESULT_REJECTED",
  failed: "RESULT_FAILED",
};
