import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";
import { automationNameForLineStep } from "./factoryLineFormShared";

/**
 * Patches a draft work order so it looks like it just landed on the first
 * step of `line` — `STATE_OPEN`, with a new active line dispatch that
 * carries a pending step-0 execution. `buildLinePhaseBoard`
 * (linePhaseRuns.ts) then places its card in the first phase column right
 * away, instead of waiting for the dispatch response and the follow-up
 * `ListWorkOrders` refetch.
 *
 * The synthetic execution's `run` is left unset on purpose:
 * `liveColumnIndexForExecution` places a run-less execution on its raw
 * `stepIndex` (0 here) regardless of the first step's app id, so placement
 * doesn't depend on data this optimistic patch doesn't have yet. Its
 * `STATE_PENDING` state makes `isActiveWorkOrderExecution` true, so it is
 * picked as the dispatch's current execution.
 */
export function buildOptimisticDispatchedOrder(
  order: FactoriesWorkOrder,
  line: Pick<FactoriesFactoryLine, "id" | "name" | "steps">,
  now: string,
): FactoriesWorkOrder {
  if (!line.id) {
    return order;
  }

  const execution: FactoriesWorkOrderExecution = {
    id: `optimistic-execution-${order.id ?? "unknown"}`,
    step: automationNameForLineStep(line.steps?.[0], undefined, 0),
    state: "STATE_PENDING",
    stepIndex: 0,
    createdAt: now,
    updatedAt: now,
  };

  const dispatch: FactoriesWorkOrderLineDispatch = {
    id: `optimistic-dispatch-${order.id ?? "unknown"}`,
    line: { id: line.id, name: line.name },
    state: "STATE_ACTIVE",
    createdAt: now,
    stepExecutions: [execution],
  };

  return {
    ...order,
    state: "STATE_OPEN",
    updatedAt: now,
    lineDispatches: [...(order.lineDispatches ?? []), dispatch],
  };
}
