import type {
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderExecutionResult,
  FactoriesWorkOrderExecutionState,
} from "@/api-client";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { UNKNOWN_ORG_USER_NAME } from "@/lib/orgUserDisplay";
import type {
  UserNameLookup,
  WorkOrderTimelineEvent,
  WorkOrderTimelineStep,
  WorkOrderTimelineViewModel,
} from "./workOrderTimelineEvents";

const UNKNOWN_MEMBER_LABEL = UNKNOWN_ORG_USER_NAME;

interface EventUserRef {
  id?: string;
}

interface EventLineRef {
  id?: string;
  name?: string;
}

interface EventRunRef {
  id?: string;
  state?: string;
  result?: string;
}

interface EventAppRef {
  id?: string;
}

interface LineStepExecutionPayload {
  stepName?: string;
  line?: EventLineRef;
  app?: EventAppRef;
  run?: EventRunRef;
}

interface EventPayload extends LineStepExecutionPayload {
  user?: EventUserRef;
  assigned?: EventUserRef[];
  unassigned?: EventUserRef[];
  order?: {
    result?: string;
  };
  result?: string;
}

interface TimelineBuildState {
  events: WorkOrderTimelineEvent[];
  dispatchBatchByLine: Map<string, WorkOrderTimelineEvent>;
}

interface DispatchBatchContext {
  timelineEvents: WorkOrderTimelineEvent[];
  dispatchBatchByLine: Map<string, WorkOrderTimelineEvent>;
}

interface DispatchBatchRequest {
  lineId: string;
  lineName: string;
  stepName: string;
  at: string;
  isCreatedEvent: boolean;
}

interface StepFromExecutionEventInput {
  payload: LineStepExecutionPayload;
  runId: string;
  stepName: string;
  at: string;
  eventType: StepExecutionEventType;
  batch: WorkOrderTimelineEvent;
}

type StepExecutionEventType = "step.execution.created" | "step.execution.finished";

const WORK_ORDER_EVENT_TYPE_ORDER: Record<string, number> = {
  "order.opened": 10,
  "order.assignees.updated": 20,
  "step.execution.created": 30,
  "step.execution.finished": 40,
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
    case "order.assignees.updated":
      appendAssigneesUpdatedEvent(state.events, index, payload, at, resolveUserName);
      return;
    case "step.execution.created":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.created");
      return;
    case "step.execution.finished":
      appendStepExecutionEvent(toDispatchBatchContext(state), payload, at, "step.execution.finished");
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
  events.push({
    id: `opened-${index}`,
    kind: "created",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
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

function appendClosedEvent(
  events: WorkOrderTimelineEvent[],
  index: number,
  payload: EventPayload,
  at: string,
  resolveUserName?: UserNameLookup,
): void {
  const closedResult = payload.result ?? payload.order?.result;
  const result = formatWorkOrderResult(
    closedResult === "completed" ? "RESULT_COMPLETED" : closedResult === "rejected" ? "RESULT_REJECTED" : undefined,
  );

  events.push({
    id: `closed-${index}`,
    kind: "closed",
    at,
    actorUserId: payload.user?.id,
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    title: result ? `closed as ${result.toLowerCase()}` : "closed this work order",
  });
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

function appendStepExecutionEvent(
  ctx: DispatchBatchContext,
  payload: LineStepExecutionPayload,
  at: string,
  eventType: StepExecutionEventType,
): void {
  const line = payload.line;
  const runId = payload.run?.id;
  if (!line?.id || !runId) {
    return;
  }

  const stepName = payload.stepName?.trim() || "Unnamed step";
  const batch = findOrCreateDispatchBatch(ctx, {
    lineId: line.id,
    lineName: line.name?.trim() || "Unnamed line",
    stepName,
    at,
    isCreatedEvent: eventType === "step.execution.created",
  });

  const step = buildStepFromExecutionEvent({
    payload,
    runId,
    stepName,
    at,
    eventType,
    batch,
  });
  upsertTimelineStep(batch, step);
}

function buildStepFromExecutionEvent(input: StepFromExecutionEventInput): WorkOrderTimelineStep {
  const { payload, runId, stepName, at, eventType, batch } = input;
  const stepId = `run-${runId}`;
  const existingStep = batch.steps?.find((item) => item.id === stepId);

  if (eventType === "step.execution.created" && existingStep?.finishedAt) {
    return existingStep;
  }

  const { startedAt, finishedAt } = resolveStepTiming(existingStep, at, eventType === "step.execution.finished");

  return {
    id: stepId,
    stepName,
    at: finishedAt ?? startedAt,
    startedAt,
    finishedAt,
    execution: executionFromStepPayload(payload, startedAt, at, eventType),
  };
}

function resolveStepTiming(
  existingStep: WorkOrderTimelineStep | undefined,
  at: string,
  isFinishedEvent: boolean,
): { startedAt: string; finishedAt?: string } {
  const startedAt = existingStep?.startedAt ?? at;
  const finishedAt = isFinishedEvent ? at : existingStep?.finishedAt;
  return { startedAt, finishedAt };
}

function upsertTimelineStep(batch: WorkOrderTimelineEvent, step: WorkOrderTimelineStep): void {
  const existingIndex = batch.steps?.findIndex((item) => item.id === step.id) ?? -1;
  if (existingIndex >= 0 && batch.steps) {
    batch.steps[existingIndex] = step;
    return;
  }

  batch.steps = [...(batch.steps ?? []), step];
}

function findOrCreateDispatchBatch(ctx: DispatchBatchContext, request: DispatchBatchRequest): WorkOrderTimelineEvent {
  let batch = ctx.dispatchBatchByLine.get(request.lineId);

  if (batch && request.isCreatedEvent && shouldStartNewDispatchBatch(batch, request.stepName)) {
    batch = undefined;
  }

  if (!batch) {
    batch = createDispatchBatchEvent(request.lineId, request.lineName, request.at);
    ctx.timelineEvents.push(batch);
    ctx.dispatchBatchByLine.set(request.lineId, batch);
  }

  return batch;
}

function shouldStartNewDispatchBatch(batch: WorkOrderTimelineEvent, stepName: string): boolean {
  if (!batch.steps?.length) {
    return false;
  }

  return stepName === batch.steps[0]?.stepName;
}

function createDispatchBatchEvent(lineId: string, lineName: string, at: string): WorkOrderTimelineEvent {
  return {
    id: `dispatch-${lineId}-${at}`,
    kind: "dispatched",
    at,
    lineName,
    title: `Dispatched to ${lineName}`,
    steps: [],
  };
}

function executionFromStepPayload(
  payload: LineStepExecutionPayload,
  startedAt: string,
  at: string,
  eventType: StepExecutionEventType,
): FactoriesWorkOrderExecution {
  const run = payload.run;
  const isFinished = eventType === "step.execution.finished";

  return {
    id: run?.id,
    step: payload.stepName,
    state: mapExecutionState(run?.state, eventType),
    result: isFinished ? mapExecutionResult(run?.result) : "RESULT_UNKNOWN",
    createdAt: startedAt,
    updatedAt: at,
    line: payload.line?.id
      ? {
          id: payload.line.id,
          name: payload.line.name,
        }
      : undefined,
    run:
      run?.id && payload.app?.id
        ? {
            id: run.id,
            appId: payload.app.id,
          }
        : undefined,
  };
}

function mapExecutionState(
  state: string | undefined,
  eventType: StepExecutionEventType,
): FactoriesWorkOrderExecutionState {
  switch (state) {
    case "running":
    case "started":
      return "STATE_STARTED";
    case "pending":
      return "STATE_PENDING";
    case "cancelling":
      return "STATE_CANCELLING";
    case "finished":
      return "STATE_FINISHED";
    default:
      return eventType === "step.execution.finished" ? "STATE_FINISHED" : "STATE_PENDING";
  }
}

function mapExecutionResult(result: string | undefined): FactoriesWorkOrderExecutionResult {
  switch (result) {
    case "passed":
      return "RESULT_PASSED";
    case "failed":
      return "RESULT_FAILED";
    case "cancelled":
      return "RESULT_CANCELLED";
    default:
      return "RESULT_UNKNOWN";
  }
}
