import type { FactoriesWorkOrder, FactoriesWorkOrderEvent, FactoriesWorkOrderExecution, SuperplaneUsersUser } from "@/api-client";
import { formatDuration } from "@/lib/duration";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { getExecutionStepTimestamp, groupWorkOrderExecutionsByLine } from "./workOrderExecutions";
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

export interface WorkOrderDispatchBatch {
  id: string;
  lineId: string;
  lineName: string;
  at: string;
  executions: FactoriesWorkOrderExecution[];
}

export function buildWorkOrderUserNameLookup(users: SuperplaneUsersUser[], order: FactoriesWorkOrder): UserNameLookup {
  const namesById = collectOrgUserNames(users);
  addOrderFallbackNames(namesById, order);
  return createUserNameResolver(namesById);
}

export function buildWorkOrderTimelineView(
  order: FactoriesWorkOrder,
  apiEvents?: FactoriesWorkOrderEvent[],
  resolveUserName?: UserNameLookup,
): WorkOrderTimelineViewModel {
  if (apiEvents?.length) {
    return buildWorkOrderTimelineViewFromEvents(apiEvents, resolveUserName);
  }

  return {
    events: buildWorkOrderTimelineEvents(order),
    closedLabel: describeWorkOrderClosed(order),
    closedAt: order.state === "STATE_CLOSED" ? (order.updatedAt ?? null) : null,
  };
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

export function buildWorkOrderTimelineEvents(order: FactoriesWorkOrder): WorkOrderTimelineEvent[] {
  const events: WorkOrderTimelineEvent[] = [];

  if (order.createdAt) {
    events.push({
      id: "created",
      kind: "created",
      at: order.createdAt,
      actorName: order.createdBy?.name?.trim() || "Someone",
      title: "opened this work order",
    });
  }

  for (const batch of groupExecutionsIntoDispatchBatches(order.executions)) {
    events.push({
      id: batch.id,
      kind: "dispatched",
      at: batch.at,
      lineName: batch.lineName,
      title: `Dispatched to ${batch.lineName}`,
      steps: batch.executions.map((execution) => stepFromWorkOrderExecution(execution, batch)),
    });
  }

  return events.sort((left, right) => Date.parse(left.at) - Date.parse(right.at));
}

export function groupExecutionsIntoDispatchBatches(
  executions: FactoriesWorkOrderExecution[] | undefined,
): WorkOrderDispatchBatch[] {
  if (!executions?.length) {
    return [];
  }

  const batches: WorkOrderDispatchBatch[] = [];

  for (const lineGroup of groupWorkOrderExecutionsByLine(executions)) {
    let current: FactoriesWorkOrderExecution[] = [];
    let batchFirstStep: string | null = null;

    for (const execution of lineGroup.executions) {
      const stepName = execution.step?.trim() || "Unnamed step";

      if (current.length > 0 && stepName === batchFirstStep) {
        batches.push(toDispatchBatch(lineGroup.lineId, lineGroup.lineName, current));
        current = [execution];
        batchFirstStep = stepName;
        continue;
      }

      if (current.length === 0) {
        batchFirstStep = stepName;
      }

      current.push(execution);
    }

    if (current.length > 0) {
      batches.push(toDispatchBatch(lineGroup.lineId, lineGroup.lineName, current));
    }
  }

  return batches.sort((left, right) => Date.parse(left.at) - Date.parse(right.at));
}

export function describeWorkOrderClosed(order: FactoriesWorkOrder): string | null {
  if (order.state !== "STATE_CLOSED") {
    return null;
  }

  const result = formatWorkOrderResult(order.result);
  return result ? `Closed as ${result.toLowerCase()}` : "Closed";
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

function stepFromWorkOrderExecution(
  execution: FactoriesWorkOrderExecution,
  batch: WorkOrderDispatchBatch,
): WorkOrderTimelineStep {
  const startedAt = execution.createdAt ?? batch.at;
  const finishedAt = isExecutionFinished(execution) ? getExecutionStepTimestamp(execution) : undefined;

  return {
    id: execution.id ?? `${batch.id}-${execution.step ?? ""}-${execution.createdAt ?? ""}`,
    stepName: execution.step?.trim() || "Unnamed step",
    at: finishedAt ?? startedAt,
    startedAt,
    finishedAt,
    execution,
  };
}

function isExecutionFinished(execution: FactoriesWorkOrderExecution): boolean {
  return execution.state === "STATE_FINISHED" || Boolean(execution.result && execution.result !== "RESULT_UNKNOWN");
}

function toDispatchBatch(
  lineId: string,
  lineName: string,
  executions: FactoriesWorkOrderExecution[],
): WorkOrderDispatchBatch {
  const first = executions[0];

  return {
    id: `${lineId}-${first?.id ?? first?.createdAt ?? lineName}`,
    lineId,
    lineName,
    at: first?.createdAt ?? "",
    executions,
  };
}
