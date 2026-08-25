import type {
  CanvasesCanvasEvent,
  CanvasesCanvasRun,
  CanvasesListEventExecutionsResponse,
} from "@/api-client";

import type { IntakeAutomationRun } from "./intakeSourceSettingsModel";
import type { LineIntakeAnalyzingTicket, LineIntakeSourceId } from "./lineIntakeModel";

const ACTIVE_EXECUTION_STATES = new Set(["STATE_PENDING", "STATE_STARTED", "STATE_CANCELLING"]);

export function analyzingTicketsFromIntakeRuns(
  sourceId: LineIntakeSourceId,
  analysisNodeId: string,
  events: CanvasesCanvasEvent[],
  executionResponses: Array<CanvasesListEventExecutionsResponse | undefined>,
): LineIntakeAnalyzingTicket[] {
  return events.flatMap((event, index) => {
    const analysis = executionResponses[index]?.executions?.find((execution) => execution.nodeId === analysisNodeId);
    const eventId = event.id?.trim();
    const title = intakeEventTitle(sourceId, event);
    if (!eventId || !title || !analysis?.state || !ACTIVE_EXECUTION_STATES.has(analysis.state)) {
      return [];
    }
    return [
      {
        id: eventId,
        title,
        ...(event.runId ? { runId: event.runId } : {}),
      },
    ];
  });
}

interface IntakeRunContext {
  appId: string;
  sourceId: LineIntakeSourceId;
  analysisNodeId: string;
  createWorkOrderNodeId: string;
}

export function automationRunsFromCanvasRuns(
  context: IntakeRunContext,
  runs: CanvasesCanvasRun[],
  executionResponses: Array<CanvasesListEventExecutionsResponse | undefined>,
  now = new Date(),
): IntakeAutomationRun[] {
  const events = runs.map((run) => ({
    ...run.rootEvent,
    runId: run.id,
    createdAt: run.createdAt ?? run.rootEvent?.createdAt,
  }));
  return automationRunsFromIntakeEvents(context, events, executionResponses, now).map((mappedRun, index) => ({
    ...mappedRun,
    id: runs[index]?.id ?? mappedRun.id,
    runId: runs[index]?.id ?? mappedRun.runId,
  }));
}

export function automationRunsFromIntakeEvents(
  context: IntakeRunContext,
  events: CanvasesCanvasEvent[],
  executionResponses: Array<CanvasesListEventExecutionsResponse | undefined>,
  now = new Date(),
): IntakeAutomationRun[] {
  return events.flatMap((event, index) => {
    const executions = executionResponses[index]?.executions ?? [];
    const analysis = executions.find((execution) => execution.nodeId === context.analysisNodeId);
    const eventId = event.id?.trim();
    const title = intakeEventTitle(context.sourceId, event);
    if (!eventId || !title || analysis?.state !== "STATE_FINISHED") {
      return [];
    }

    const createWorkOrder = executions.find((execution) => execution.nodeId === context.createWorkOrderNodeId);
    const placement =
      analysis.result === "RESULT_FAILED"
        ? ("rejected" as const)
        : createWorkOrder?.result === "RESULT_PASSED"
          ? ("backlog" as const)
          : ("below-threshold" as const);

    return [
      {
        id: eventId,
        appId: context.appId,
        ...(event.runId ? { runId: event.runId } : {}),
        title,
        confidencePct: confidenceScore(analysis.outputs),
        ranMinutesAgo: minutesAgo(event.createdAt, now),
        analyzedMinutesAgo: minutesAgo(analysis.updatedAt, now),
        placement,
      },
    ];
  });
}

function intakeEventTitle(sourceId: LineIntakeSourceId, event: CanvasesCanvasEvent): string | undefined {
  const payload = recordValue(event.data?.data);
  if (sourceId === "github-issues") {
    return stringValue(recordValue(payload?.issue)?.title);
  }
  if (sourceId === "sentry-exceptions") {
    return stringValue(recordValue(recordValue(payload?.data)?.issue)?.title);
  }
  return stringValue(recordValue(payload?.incident)?.title);
}

function recordValue(value: unknown): Record<string, unknown> | undefined {
  return value && typeof value === "object" && !Array.isArray(value) ? (value as Record<string, unknown>) : undefined;
}

function stringValue(value: unknown): string | undefined {
  return typeof value === "string" && value.trim() ? value.trim() : undefined;
}

function confidenceScore(outputs: Record<string, unknown> | undefined): number {
  const score = findResultScore(outputs);
  return score === undefined ? 0 : Math.min(100, Math.max(0, Math.round(score)));
}

function findResultScore(value: unknown): number | undefined {
  if (typeof value === "number" && Number.isFinite(value)) {
    return value;
  }
  if (typeof value === "string" && value.trim() && Number.isFinite(Number(value))) {
    return Number(value);
  }
  if (Array.isArray(value)) {
    return value.reduce<number | undefined>((score, item) => score ?? findResultScore(item), undefined);
  }
  const record = recordValue(value);
  if (!record) {
    return undefined;
  }
  if ("result" in record) {
    const score = findResultScore(record.result);
    if (score !== undefined) {
      return score;
    }
  }
  return Object.values(record).reduce<number | undefined>((score, item) => score ?? findResultScore(item), undefined);
}

function minutesAgo(timestamp: string | undefined, now: Date): number {
  const value = timestamp ? Date.parse(timestamp) : Number.NaN;
  if (!Number.isFinite(value)) {
    return 0;
  }
  return Math.max(0, Math.floor((now.getTime() - value) / 60_000));
}
