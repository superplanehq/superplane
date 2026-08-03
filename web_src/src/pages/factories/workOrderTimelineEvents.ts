import type { FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { formatWorkOrderResult } from "./workOrderPresentation";
import { getExecutionStepTimestamp, groupWorkOrderExecutionsByLine } from "./workOrderExecutions";

export type WorkOrderTimelineEventKind = "created" | "dispatched";

export interface WorkOrderTimelineStep {
  id: string;
  stepName: string;
  at: string;
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

export interface WorkOrderDispatchBatch {
  id: string;
  lineId: string;
  lineName: string;
  at: string;
  executions: FactoriesWorkOrderExecution[];
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
      steps: batch.executions.map((execution) => ({
        id: execution.id ?? `${batch.id}-${execution.step ?? ""}-${execution.createdAt ?? ""}`,
        stepName: execution.step?.trim() || "Unnamed step",
        at: getExecutionStepTimestamp(execution),
        execution,
      })),
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

export function describeWorkOrderClosed(order: FactoriesWorkOrder): string | null {
  if (order.state !== "STATE_CLOSED") {
    return null;
  }

  const result = formatWorkOrderResult(order.result);
  return result ? `Closed as ${result.toLowerCase()}` : "Closed";
}
