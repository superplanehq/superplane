import type {
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderExecutionResult,
  FactoriesWorkOrderExecutionState,
} from "@/api-client";
import type { WorkOrderTimelineEvent, WorkOrderTimelineStep } from "./workOrderTimelineEvents";

export type StepExecutionEventType = "step.execution.created" | "step.execution.finished";

export interface LineStepExecutionPayload {
  stepName?: string;
  line?: { id?: string; name?: string };
  app?: { id?: string; name?: string };
  run?: { id?: string; state?: string; result?: string };
}

export interface DispatchBatchContext {
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

export function appendStepExecutionEvent(
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

  const stepName = payload.app?.name?.trim() || payload.stepName?.trim() || "Unnamed step";
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
    comments: existingStep?.comments,
    artifacts: existingStep?.artifacts,
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
    lineId,
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
    step: payload.app?.name?.trim() || payload.stepName,
    state: mapExecutionState(run?.state, eventType),
    result: isFinished ? mapExecutionResult(run?.result) : "RESULT_UNKNOWN",
    createdAt: startedAt,
    updatedAt: at,
    run:
      run?.id && payload.app?.id
        ? {
            id: run.id,
            appId: payload.app.id,
            appName: payload.app.name,
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
