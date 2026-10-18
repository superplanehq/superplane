import type {
  FactoriesFactory,
  FactoriesFactoryPullRequest,
  FactoriesLineRef,
  FactoriesWorkOrder,
  FactoriesWorkOrderCheckScore,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
} from "@/api-client";

import { isActiveWorkOrderExecution } from "./workOrderExecutions";

export const workOrderListFactory: FactoriesFactory = { id: "f", name: "Refunds", key: "RF" };

/** A step execution plus the line it ran on, before `order()` groups dispatches. */
type TestExecution = FactoriesWorkOrderExecution & { line?: FactoriesLineRef };

interface OrderOverrides extends Omit<Partial<FactoriesWorkOrder>, "lineDispatches"> {
  executions?: TestExecution[];
  checkScores?: FactoriesWorkOrderCheckScore[];
}

export function workOrderListOrder(overrides: OrderOverrides = {}) {
  const { executions, ...rest } = overrides;
  return {
    id: overrides.id ?? "wo-1",
    title: "Order title",
    state: "STATE_OPEN" as const,
    result: "RESULT_UNSPECIFIED" as const,
    createdAt: "2024-06-01T00:00:00Z",
    updatedAt: "2024-06-02T00:00:00Z",
    lineDispatches: executions ? dispatchesFromExecutions(executions) : [],
    ...rest,
  };
}

export function workOrderListPullRequest(
  workOrderId: string,
  overrides: Partial<FactoriesFactoryPullRequest> = {},
): FactoriesFactoryPullRequest {
  return {
    id: `pr-${workOrderId}`,
    workOrderId,
    number: "12",
    url: "https://github.com/acme/payments/pull/12",
    title: "Ship refund retries",
    state: "STATE_OPEN",
    ...overrides,
  };
}

function dispatchesFromExecutions(executions: TestExecution[]): FactoriesWorkOrderLineDispatch[] {
  const byLineId = new Map<string, FactoriesWorkOrderLineDispatch>();
  for (const { line, ...execution } of executions) {
    const lineId = line?.id ?? "unknown";
    const dispatch = byLineId.get(lineId);
    if (dispatch) {
      dispatch.stepExecutions = [...(dispatch.stepExecutions ?? []), execution];
      continue;
    }
    byLineId.set(lineId, {
      id: `dispatch-${lineId}`,
      line,
      state: isActiveWorkOrderExecution(execution) ? "STATE_ACTIVE" : "STATE_FINISHED",
      stepExecutions: [execution],
    });
  }
  return [...byLineId.values()];
}
