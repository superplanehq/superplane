import type {
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderLineDispatchResult,
  FactoriesWorkOrderLineDispatchState,
} from "@/api-client";

import { HOUR_AGO, REFUND_LINE_PLAN_ID, TWO_HOURS_AGO } from "./factoryPageIds";

const PLAN_STEP_INDEX: Record<string, number> = { plan: 0, implement: 1, verify: 2 };
const PLAN_STEP_LABEL: Record<string, string> = {
  plan: "Refund Planner",
  implement: "Refund Implementer",
  verify: "Refund Verifier",
};

export function planLineExecution(
  step: string,
  overrides: Partial<FactoriesWorkOrderExecution> = {},
): FactoriesWorkOrderExecution {
  return {
    id: `exec-${step}-${overrides.id ?? Math.random().toString(36).slice(2, 8)}`,
    step: PLAN_STEP_LABEL[step] ?? step,
    stepIndex: PLAN_STEP_INDEX[step] ?? 0,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: TWO_HOURS_AGO,
    updatedAt: HOUR_AGO,
    run: {
      id: `run-${step}`,
      appId: "app-refund-planner",
      appName: "Refund Planner",
    },
    ...overrides,
  };
}

/**
 * Builds a single line dispatch (traversal) around a set of step
 * executions. Fixtures give every work order at most one dispatch of the
 * plan-and-implement line.
 */
export function planLineDispatch(
  stepExecutions: FactoriesWorkOrderExecution[],
  overrides: Partial<FactoriesWorkOrderLineDispatch> = {},
): FactoriesWorkOrderLineDispatch {
  const state: FactoriesWorkOrderLineDispatchState = stepExecutions.some(
    (execution) => execution.state !== "STATE_FINISHED",
  )
    ? "STATE_ACTIVE"
    : "STATE_FINISHED";

  const lastExecution = stepExecutions[stepExecutions.length - 1];
  const result: FactoriesWorkOrderLineDispatchResult =
    state === "STATE_FINISHED" ? (lastExecution?.result ?? "RESULT_UNKNOWN") : "RESULT_UNKNOWN";

  return {
    id: `dispatch-${REFUND_LINE_PLAN_ID}-${stepExecutions[0]?.id ?? "empty"}`,
    line: { id: REFUND_LINE_PLAN_ID, name: "plan-and-implement" },
    steps: [
      { name: "Refund Planner", stepIndex: 0 },
      { name: "Refund Implementer", stepIndex: 1 },
      { name: "Refund Verifier", stepIndex: 2 },
    ],
    state,
    result,
    createdAt: stepExecutions[0]?.createdAt ?? TWO_HOURS_AGO,
    finishedAt: state === "STATE_FINISHED" ? (lastExecution?.updatedAt ?? HOUR_AGO) : undefined,
    stepExecutions,
    ...overrides,
  };
}
