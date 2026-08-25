import type {
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderLineDispatchResult,
  FactoriesWorkOrderLineDispatchState,
} from "@/api-client";

import { HOUR_AGO, REFUND_LINE_PLAN_ID, TWO_HOURS_AGO } from "./factoryPageIds";

const PLAN_STEP_INDEX: Record<string, number> = { implement: 0, verify: 1 };
const PLAN_STEP_LABEL: Record<string, string> = {
  implement: "Implement",
  verify: "Verify",
};
const PLAN_STEP_RUN: Record<string, { appId: string; appName: string }> = {
  implement: { appId: "app-refund-implementer", appName: "Implementation" },
  verify: { appId: "app-refund-verifier", appName: "Risk Assessment" },
};

export function planLineExecution(
  step: string,
  overrides: Partial<FactoriesWorkOrderExecution> = {},
): FactoriesWorkOrderExecution {
  const run = PLAN_STEP_RUN[step] ?? PLAN_STEP_RUN.implement;
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
      appId: run.appId,
      appName: run.appName,
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
      { name: "Implement", stepIndex: 0 },
      { name: "Verify", stepIndex: 1 },
    ],
    state,
    result,
    createdAt: stepExecutions[0]?.createdAt ?? TWO_HOURS_AGO,
    finishedAt: state === "STATE_FINISHED" ? (lastExecution?.updatedAt ?? HOUR_AGO) : undefined,
    stepExecutions,
    ...overrides,
  };
}
