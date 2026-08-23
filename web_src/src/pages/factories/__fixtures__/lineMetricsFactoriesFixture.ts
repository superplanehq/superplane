import type { FactoriesWorkOrder } from "@/api-client";

import { REVIEW_CANDIDATE_WORK_ORDERS } from "../pages/onboarding/first-run/reviewCandidates";
import {
  CLOSED_WORK_ORDER,
  FAILED_WORK_ORDER,
  HOUR_AGO,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_PLAN_ID,
  TWO_HOURS_AGO,
  YESTERDAY,
  defaultFactoriesFixture,
  type FactoriesFixture,
} from "./factoryPageResponses";
import {
  BACKLOG_APP,
  PLAN_LINE_APPS,
  PLAN_LINE_DONE_APP_ID,
  planLineActiveDispatch,
  runAppStep,
  withPlanLinePhases,
} from "./lineMetricsPlanLine";
import {
  BOARD_DONE_CANCELED_ORDER,
  BOARD_DONE_COMPLETED_PLAYBOOK,
  BOARD_DONE_COMPLETED_SLA,
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  FEATURE_CI_WORK_ORDER,
  FEATURE_DELIVERY_LINE,
  FEATURE_PR_WORK_ORDER,
  FEATURE_RUNNING_WORK_ORDER,
  ONBOARDING_FACTORY_LINE,
} from "./lineMetricsBoardOrders";

function withPlanPhase(order: FactoriesWorkOrder): FactoriesWorkOrder {
  const orderId = order.id;
  if (!orderId || orderId !== OPEN_WORK_ORDER.id) {
    return order;
  }
  return {
    ...order,
    lineDispatches: [
      planLineActiveDispatch(orderId, [
        {
          id: "exec-plan-open",
          step: "Plan",
          stepIndex: 0,
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
          createdAt: HOUR_AGO,
          updatedAt: HOUR_AGO,
          run: { id: "run-plan-open", appId: "app-refund-planner", appName: "Plan" },
        },
      ]),
    ],
  };
}

function withVerifyPhase(order: FactoriesWorkOrder): FactoriesWorkOrder {
  const orderId = order.id;
  if (!orderId || orderId !== OPEN_WORK_ORDER_SECONDARY.id) {
    return order;
  }
  return {
    ...order,
    lineDispatches: [
      planLineActiveDispatch(orderId, [
        {
          id: "exec-verify-plan",
          step: "Plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: TWO_HOURS_AGO,
          run: { id: "run-verify-plan", appId: "app-refund-planner", appName: "Plan" },
        },
        {
          id: "exec-verify-implement",
          step: "Implement",
          stepIndex: 1,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: HOUR_AGO,
          run: { id: "run-verify-implement", appId: "app-refund-implementer", appName: "Implement" },
        },
        {
          id: "exec-verify-open",
          step: "Verify",
          stepIndex: 2,
          state: "STATE_STARTED",
          result: "RESULT_UNKNOWN",
          createdAt: HOUR_AGO,
          updatedAt: HOUR_AGO,
          run: { id: "run-verify-open", appId: "app-refund-verifier", appName: "Verify" },
        },
      ]),
    ],
  };
}

function withWaitingPrReview(order: FactoriesWorkOrder): FactoriesWorkOrder {
  if (order.id !== FAILED_WORK_ORDER.id) {
    return order;
  }
  const dispatch = order.lineDispatches?.[0];
  if (!dispatch) {
    return order;
  }
  return {
    ...order,
    statusNotes: OPEN_WORK_ORDER.statusNotes,
    lineDispatches: [
      {
        ...dispatch,
        result: "RESULT_UNKNOWN",
        stepExecutions: [
          ...(dispatch.stepExecutions ?? []).map((execution) =>
            execution.stepIndex === 1 ? { ...execution, result: "RESULT_PASSED" as const } : execution,
          ),
          {
            id: "exec-verify-pr",
            step: "Verify",
            stepIndex: 2,
            state: "STATE_FINISHED",
            result: "RESULT_PASSED" as const,
            createdAt: HOUR_AGO,
            updatedAt: HOUR_AGO,
            run: { id: "run-verify-pr", appId: "app-refund-verifier", appName: "Verify" },
          },
        ],
      },
    ],
  };
}

function withDonePhase(order: FactoriesWorkOrder): FactoriesWorkOrder {
  const doneRun =
    order.id === CLOSED_WORK_ORDER.id
      ? { executionId: "exec-done-closed", runId: "run-done-closed", appName: "Done" }
      : order.id === PR_CLOSURE_COMPLETED_WORK_ORDER.id
        ? { executionId: "exec-done-pr-closure", runId: "run-done-pr-closure", appName: "PR Closure" }
        : null;
  if (!doneRun) {
    return order;
  }
  const dispatch = order.lineDispatches?.[0];
  if (!dispatch) {
    return order;
  }
  return {
    ...order,
    lineDispatches: [
      {
        ...dispatch,
        steps: [...(dispatch.steps ?? []), { name: "Done", stepIndex: 3 }],
        stepExecutions: [
          ...(dispatch.stepExecutions ?? []),
          {
            id: doneRun.executionId,
            step: "Done",
            stepIndex: 3,
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: YESTERDAY,
            updatedAt: YESTERDAY,
            run: { id: doneRun.runId, appId: PLAN_LINE_DONE_APP_ID, appName: doneRun.appName },
          },
        ],
      },
    ],
  };
}

/**
 * Extra lines for the populated Lines list: unused onboarding plus a
 * four-phase feature line.
 */
export const lineMetricsFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  factories: defaultFactoriesFixture.factories.map((factory) => {
    if (factory.id !== PRIMARY_FACTORY_ID) {
      return factory;
    }
    return {
      ...factory,
      lines: [...(factory.lines ?? []).map(withPlanLinePhases), ONBOARDING_FACTORY_LINE, FEATURE_DELIVERY_LINE],
    };
  }),
  appsByFactoryId: {
    ...defaultFactoriesFixture.appsByFactoryId,
    [PRIMARY_FACTORY_ID]: PLAN_LINE_APPS,
  },
  workOrdersByFactoryId: {
    ...defaultFactoriesFixture.workOrdersByFactoryId,
    [PRIMARY_FACTORY_ID]: [
      ...REVIEW_CANDIDATE_WORK_ORDERS,
      ...(defaultFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? [])
        .map(withPlanPhase)
        .map(withVerifyPhase)
        .map(withWaitingPrReview)
        .map(withDonePhase),
      BOARD_IMPLEMENT_FAILED_ORDER,
      BOARD_DONE_COMPLETED_SLA,
      BOARD_DONE_COMPLETED_PLAYBOOK,
      BOARD_DONE_REJECTED_ORDER,
      BOARD_DONE_CANCELED_ORDER,
      FEATURE_RUNNING_WORK_ORDER,
      FEATURE_PR_WORK_ORDER,
      FEATURE_CI_WORK_ORDER,
    ],
  },
};

/** Story-only clone: Plan and Implement has five phases so the board can scroll on x. */
export const fiveStepLineFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  factories: defaultFactoriesFixture.factories.map((factory) => {
    if (factory.id !== PRIMARY_FACTORY_ID) {
      return factory;
    }
    return {
      ...factory,
      lines: (factory.lines ?? []).map((line) => {
        if (line.id !== REFUND_LINE_PLAN_ID) {
          return line;
        }
        return {
          ...line,
          steps: [
            ...(line.steps ?? []),
            runAppStep("app-refund-verifier", "start-verification"),
            runAppStep("app-refund-planner", "start-plan"),
          ],
        };
      }),
    };
  }),
  appsByFactoryId: {
    ...defaultFactoriesFixture.appsByFactoryId,
    [PRIMARY_FACTORY_ID]: [BACKLOG_APP, ...(defaultFactoriesFixture.appsByFactoryId[PRIMARY_FACTORY_ID] ?? [])],
  },
};
