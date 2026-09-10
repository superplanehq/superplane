import type { FactoriesFactoryPullRequest, FactoriesWorkOrder } from "@/api-client";

import { BOARD_REVIEW_CANDIDATE_WORK_ORDERS } from "../pages/onboarding/first-run/reviewCandidates";
import { INGEST_CREATED_BY } from "./factoryPageWorkOrders";
import {
  APPROVAL_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FAILED_WORK_ORDER,
  HOUR_AGO,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_PLAN_ID,
  RUNNING_WORK_ORDER,
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
  BOARD_DONE_REJECTED_ORDER,
  BOARD_IMPLEMENT_FAILED_ORDER,
  BOARD_IMPLEMENT_NOTIFY_ORDER,
  FEATURE_CI_WORK_ORDER,
  FEATURE_DELIVERY_LINE,
  FEATURE_PR_WORK_ORDER,
  FEATURE_RUNNING_WORK_ORDER,
  ONBOARDING_FACTORY_LINE,
} from "./lineMetricsBoardOrders";

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
          id: "exec-verify-implement",
          step: "Implement",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: HOUR_AGO,
          run: { id: "run-verify-implement", appId: "app-refund-implementer", appName: "Implement" },
        },
        {
          id: "exec-verify-open",
          step: "Verify",
          stepIndex: 1,
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
            execution.step === "Implement" ? { ...execution, result: "RESULT_PASSED" as const } : execution,
          ),
          {
            id: "exec-verify-pr",
            step: "Verify",
            stepIndex: 1,
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
    order.id === PR_CLOSURE_COMPLETED_WORK_ORDER.id
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
        steps: [...(dispatch.steps ?? []), { name: "Done", stepIndex: 2 }],
        stepExecutions: [
          ...(dispatch.stepExecutions ?? []),
          {
            id: doneRun.executionId,
            step: "Done",
            stepIndex: 2,
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
export const LINE_BOARD_VERIFY_ENUM_ORDER = {
  ...withVerifyPhase(OPEN_WORK_ORDER_SECONDARY),
  createdBy: INGEST_CREATED_BY,
};
export const LINE_BOARD_VERIFY_PR_REVIEW_ORDER = withWaitingPrReview(FAILED_WORK_ORDER);
export const LINE_BOARD_DONE_RECEIPTS_ORDER = withDonePhase(PR_CLOSURE_COMPLETED_WORK_ORDER);

function backlogReviewOrder(orderId: string): FactoriesWorkOrder {
  const order = BOARD_REVIEW_CANDIDATE_WORK_ORDERS.find((entry) => entry.id === orderId);
  if (!order) {
    throw new Error(`Missing backlog review ticket ${orderId}`);
  }
  return order;
}

const BACKLOG_WEBHOOK_ORDER_ID = "wo-review-pay-842";
const BACKLOG_FACTORY_500_ORDER_ID = "wo-review-pay-844";
const BACKLOG_WEBHOOK_ORDER = backlogReviewOrder(BACKLOG_WEBHOOK_ORDER_ID);
const BACKLOG_FACTORY_500_ORDER = backlogReviewOrder(BACKLOG_FACTORY_500_ORDER_ID);

function lineBoardPullRequest(
  order: FactoriesWorkOrder,
  number: string,
  state: NonNullable<FactoriesFactoryPullRequest["state"]>,
  url?: string,
): FactoriesFactoryPullRequest {
  return {
    id: `pr-${order.id}`,
    workOrderId: order.id,
    number,
    url: url ?? `https://github.com/example/ledger/pull/${number}`,
    title: order.title,
    state,
  };
}

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
      // Board inventory: 4 backlog, 3 implement, 2 verify, 4 done.
      // Two cards in each column carry a pull request.
      ...BOARD_REVIEW_CANDIDATE_WORK_ORDERS,
      DRAFT_WORK_ORDER,
      RUNNING_WORK_ORDER,
      APPROVAL_WORK_ORDER,
      BOARD_IMPLEMENT_NOTIFY_ORDER,
      BOARD_IMPLEMENT_FAILED_ORDER,
      LINE_BOARD_VERIFY_ENUM_ORDER,
      LINE_BOARD_VERIFY_PR_REVIEW_ORDER,
      LINE_BOARD_DONE_RECEIPTS_ORDER,
      BOARD_DONE_REJECTED_ORDER,
      BOARD_DONE_CANCELED_ORDER,
      FEATURE_RUNNING_WORK_ORDER,
      FEATURE_PR_WORK_ORDER,
      FEATURE_CI_WORK_ORDER,
    ],
  },
  pullRequestsByOrderId: {
    [BACKLOG_WEBHOOK_ORDER_ID]: [lineBoardPullRequest(BACKLOG_WEBHOOK_ORDER, "842", "STATE_DRAFT")],
    [BACKLOG_FACTORY_500_ORDER_ID]: [lineBoardPullRequest(BACKLOG_FACTORY_500_ORDER, "844", "STATE_DRAFT")],
    [APPROVAL_WORK_ORDER.id!]: [lineBoardPullRequest(APPROVAL_WORK_ORDER, "109", "STATE_OPEN")],
    [BOARD_IMPLEMENT_NOTIFY_ORDER.id!]: [lineBoardPullRequest(BOARD_IMPLEMENT_NOTIFY_ORDER, "114", "STATE_DRAFT")],
    [LINE_BOARD_VERIFY_PR_REVIEW_ORDER.id!]: [
      lineBoardPullRequest(
        LINE_BOARD_VERIFY_PR_REVIEW_ORDER,
        "6812",
        "STATE_OPEN",
        "https://github.com/superplanehq/superplane/pull/6812",
      ),
    ],
    [LINE_BOARD_VERIFY_ENUM_ORDER.id!]: [lineBoardPullRequest(LINE_BOARD_VERIFY_ENUM_ORDER, "102", "STATE_DRAFT")],
    [LINE_BOARD_DONE_RECEIPTS_ORDER.id!]: [lineBoardPullRequest(LINE_BOARD_DONE_RECEIPTS_ORDER, "510", "STATE_MERGED")],
    [BOARD_DONE_REJECTED_ORDER.id!]: [lineBoardPullRequest(BOARD_DONE_REJECTED_ORDER, "112", "STATE_CLOSED")],
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
            runAppStep("app-refund-implementer", "start-observe"),
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
