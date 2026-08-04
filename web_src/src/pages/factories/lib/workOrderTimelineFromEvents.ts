import type {
  FactoriesWorkOrderEvent,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderExecutionResult,
  FactoriesWorkOrderExecutionState,
} from "@/api-client";
import { formatWorkOrderResult } from "./workOrderPresentation";
import type {
  UserNameLookup,
  WorkOrderTimelineEvent,
  WorkOrderTimelineStep,
  WorkOrderTimelineViewModel,
} from "./workOrderTimelineEvents";

const UNKNOWN_MEMBER_LABEL = "Unknown member";

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

/** @deprecated Legacy event shape; kept for older stored events. */
interface EventExecutionRef {
  id?: string;
  stepName?: string;
  step_name?: string;
  run?: EventRunRef;
  run_id?: string;
  app_id?: string;
  app_name?: string;
  result?: string;
  state?: string;
}

interface EventPayload extends LineStepExecutionPayload {
  user?: EventUserRef;
  execution?: EventExecutionRef;
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
  closedLabel: string | null;
  closedAt: string | null;
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

interface StepFromLegacyEventInput {
  executionPayload: EventExecutionRef;
  line: EventLineRef | undefined;
  at: string;
  index: number;
  eventType: string;
  batch: WorkOrderTimelineEvent;
}

type StepExecutionEventType = "step.execution.created" | "step.execution.finished";

export function buildWorkOrderTimelineViewFromEvents(
  apiEvents: FactoriesWorkOrderEvent[],
  resolveUserName?: UserNameLookup,
): WorkOrderTimelineViewModel {
  const eventsAsc = [...apiEvents].sort(
    (left, right) => Date.parse(left.timestamp ?? "") - Date.parse(right.timestamp ?? ""),
  );

  const state = createTimelineBuildState();

  for (const [index, apiEvent] of eventsAsc.entries()) {
    applyApiEventToTimeline(state, index, apiEvent, resolveUserName);
  }

  return { events: state.events, closedLabel: state.closedLabel, closedAt: state.closedAt };
}

function createTimelineBuildState(): TimelineBuildState {
  return {
    events: [],
    dispatchBatchByLine: new Map(),
    closedLabel: null,
    closedAt: null,
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
    case "order.dispatched":
      registerExplicitDispatchBatch(state, index, payload, at);
      return;
    case "line.step.started":
    case "line.step.finished":
      appendLegacyStepEvent(toDispatchBatchContext(state), { payload, at, index, eventType: apiEvent.type ?? "" });
      return;
    case "order.closed":
      applyOrderClosedState(state, payload, at);
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
    actorName: resolveUserDisplayName(payload.user?.id, resolveUserName),
    title: describeAssigneesUpdated(payload, resolveUserName),
  });
}

function registerExplicitDispatchBatch(
  state: TimelineBuildState,
  index: number,
  payload: EventPayload,
  at: string,
): void {
  const line = payload.line;
  const lineId = line?.id ?? `line-${index}`;
  const lineName = line?.name?.trim() || "Unnamed line";
  const batch = createDispatchBatchEvent(lineId, lineName, at);

  state.events.push(batch);
  state.dispatchBatchByLine.set(lineId, batch);
}

function applyOrderClosedState(state: TimelineBuildState, payload: EventPayload, at: string): void {
  const closedResult = payload.result ?? payload.order?.result;
  const result = formatWorkOrderResult(
    closedResult === "completed" ? "RESULT_COMPLETED" : closedResult === "rejected" ? "RESULT_REJECTED" : undefined,
  );
  state.closedLabel = result ? `Closed as ${result.toLowerCase()}` : "Closed";
  state.closedAt = at;
}

function resolveUserDisplayName(userId: string | undefined, resolveUserName?: UserNameLookup): string | undefined {
  if (!userId) {
    return undefined;
  }

  return resolveUserName?.(userId) ?? UNKNOWN_MEMBER_LABEL;
}

function describeAssigneesUpdated(payload: EventPayload, resolveUserName?: UserNameLookup): string {
  const assignedNames = formatEventUserNames(payload.assigned, resolveUserName);
  const unassignedNames = formatEventUserNames(payload.unassigned, resolveUserName);
  const parts: string[] = [];

  if (assignedNames) {
    parts.push(`assigned ${assignedNames}`);
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

function appendLegacyStepEvent(
  ctx: DispatchBatchContext,
  input: { payload: EventPayload; at: string; index: number; eventType: string },
): void {
  const { payload, at, index, eventType } = input;
  const line = payload.line;
  const executionPayload = payload.execution;
  if (!executionPayload) {
    return;
  }

  const lineId = line?.id ?? "unknown-line";
  const lineName = line?.name?.trim() || "Unnamed line";
  const batch = findOrCreateDispatchBatch(ctx, {
    lineId,
    lineName,
    stepName: legacyStepName(executionPayload),
    at,
    isCreatedEvent: eventType === "line.step.started",
  });

  const step = buildStepFromLegacyEvent({
    executionPayload,
    line,
    at,
    index,
    eventType,
    batch,
  });
  upsertTimelineStep(batch, step);
}

function buildStepFromLegacyEvent(input: StepFromLegacyEventInput): WorkOrderTimelineStep {
  const { executionPayload, line, at, index, eventType, batch } = input;
  const execution = executionFromLegacyPayload(executionPayload, line, at, eventType);
  const stepId = execution.id ?? `step-${index}`;
  const existingStep = batch.steps?.find((item) => item.id === stepId);
  const isFinished = eventType === "line.step.finished";
  const { startedAt, finishedAt } = resolveStepTiming(existingStep, at, isFinished);

  return {
    id: stepId,
    stepName: legacyStepName(executionPayload),
    at: finishedAt ?? startedAt,
    startedAt,
    finishedAt,
    execution: {
      ...execution,
      createdAt: startedAt,
      updatedAt: finishedAt ?? startedAt,
    },
  };
}

function legacyStepName(executionPayload: EventExecutionRef): string {
  return executionPayload.stepName?.trim() || executionPayload.step_name?.trim() || "Unnamed step";
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

function executionFromLegacyPayload(
  executionPayload: EventExecutionRef,
  line: EventLineRef | undefined,
  at: string,
  eventType: string,
): FactoriesWorkOrderExecution {
  const state = mapExecutionState(executionPayload.state ?? executionPayload.run?.state, eventType);
  const result = mapExecutionResult(executionPayload.result ?? executionPayload.run?.result);

  return {
    id: executionPayload.id,
    step: executionPayload.stepName ?? executionPayload.step_name,
    state,
    result,
    createdAt: at,
    updatedAt: at,
    line: line?.id
      ? {
          id: line.id,
          name: line.name,
        }
      : undefined,
    run:
      (executionPayload.run_id ?? executionPayload.run?.id) && executionPayload.app_id
        ? {
            id: executionPayload.run_id ?? executionPayload.run?.id,
            appId: executionPayload.app_id,
            appName: executionPayload.app_name,
          }
        : undefined,
  };
}

function mapExecutionState(state: string | undefined, eventType: string): FactoriesWorkOrderExecutionState {
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
      if (eventType === "line.step.finished" || eventType === "step.execution.finished") {
        return "STATE_FINISHED";
      }
      return "STATE_PENDING";
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
