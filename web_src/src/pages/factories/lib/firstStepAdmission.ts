import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoriesWorkOrderExecution } from "@/api-client";
import { lineStepParallelism } from "./factoryLineFormShared";

const FIRST_STEP_INDEX = 0;

export type FirstStepAdmission = {
  line: FactoriesFactoryLine;
  workOrders: FactoriesWorkOrder[];
  firstStepMaxParallelism?: number;
  /** Organization or installation cap for this factory. */
  factoryMaxParallelTasks?: number;
  /**
   * Starts that are in flight on the client and do not yet occupy an
   * execution slot in `workOrders`.
   */
  reservedStarts?: number;
};

/** Pending, running, or cancelling executions occupy a step slot. Queue items do not. */
export function isInFlightStepExecution(execution: Pick<FactoriesWorkOrderExecution, "state">): boolean {
  return (
    execution.state === "STATE_PENDING" || execution.state === "STATE_STARTED" || execution.state === "STATE_CANCELLING"
  );
}

export function countQueuedItemsAtStep(workOrders: FactoriesWorkOrder[], lineId: string, stepIndex: number): number {
  let count = 0;
  for (const order of workOrders) {
    for (const dispatch of order.lineDispatches ?? []) {
      if (dispatch.line?.id !== lineId) {
        continue;
      }
      if (dispatch.queueItem && (dispatch.queueItem.stepIndex ?? 0) === stepIndex) {
        count += 1;
      }
    }
  }
  return count;
}

export function countInFlightExecutionsAtStep(
  workOrders: FactoriesWorkOrder[],
  lineId: string,
  stepIndex: number,
): number {
  let count = 0;
  for (const order of workOrders) {
    for (const dispatch of order.lineDispatches ?? []) {
      if (dispatch.line?.id !== lineId) {
        continue;
      }
      for (const execution of dispatch.stepExecutions ?? []) {
        if ((execution.stepIndex ?? 0) !== stepIndex) {
          continue;
        }
        if (isInFlightStepExecution(execution)) {
          count += 1;
        }
      }
    }
  }
  return count;
}

export function countInFlightFactoryExecutions(workOrders: FactoriesWorkOrder[]): number {
  let count = 0;
  for (const order of workOrders) {
    for (const dispatch of order.lineDispatches ?? []) {
      for (const execution of dispatch.stepExecutions ?? []) {
        if (isInFlightStepExecution(execution)) {
          count += 1;
        }
      }
    }
  }
  return count;
}

function orderHasInFlightExecution(order: FactoriesWorkOrder): boolean {
  for (const dispatch of order.lineDispatches ?? []) {
    for (const execution of dispatch.stepExecutions ?? []) {
      if (isInFlightStepExecution(execution)) {
        return true;
      }
    }
  }
  return false;
}

/** Starts whose cards are busy and do not already occupy an execution slot. */
export function reservedStartCount(
  workOrders: FactoriesWorkOrder[],
  dispatchingOrderIds: ReadonlySet<string>,
): number {
  if (dispatchingOrderIds.size === 0) {
    return 0;
  }
  const occupying = new Set<string>();
  for (const order of workOrders) {
    if (order.id && orderHasInFlightExecution(order)) {
      occupying.add(order.id);
    }
  }
  let reserved = 0;
  for (const orderId of dispatchingOrderIds) {
    if (!occupying.has(orderId)) {
      reserved += 1;
    }
  }
  return reserved;
}

/**
 * True when a first-step Start on this line would join the queue instead of
 * starting a run: waiters already sit at step 0 (FIFO), in-flight work
 * already fills the step's parallel cap, or the factory is at its parallel
 * task cap.
 */
export function willQueueFirstStepStart(admission: FirstStepAdmission): boolean {
  const lineId = admission.line.id;
  if (!lineId) {
    return false;
  }
  if (countQueuedItemsAtStep(admission.workOrders, lineId, FIRST_STEP_INDEX) > 0) {
    return true;
  }
  const reserved = admission.reservedStarts ?? 0;
  const stepCap = admission.firstStepMaxParallelism ?? lineStepParallelism(admission.line.steps?.[FIRST_STEP_INDEX]);
  if (countInFlightExecutionsAtStep(admission.workOrders, lineId, FIRST_STEP_INDEX) + reserved >= stepCap) {
    return true;
  }
  const factoryCap = admission.factoryMaxParallelTasks;
  if (factoryCap == null || factoryCap < 1) {
    return false;
  }
  return countInFlightFactoryExecutions(admission.workOrders) + reserved >= factoryCap;
}
