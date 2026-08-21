import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderLineDispatchResult,
  FactoriesWorkOrderLineDispatchState,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";

import {
  CLOSED_WORK_ORDER,
  FAILED_WORK_ORDER,
  HOUR_AGO,
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  OPERATOR_USER,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_APPS,
  REFUND_LINE_FEATURE_ID,
  REFUND_LINE_ONBOARDING_ID,
  REFUND_LINE_PLAN_ID,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
  YESTERDAY,
  defaultFactoriesFixture,
  type FactoriesFixture,
} from "./factoryPageResponses";

const RUN_APP_TYPE = "runApp";

function runAppStep(appId: string, entrypoint: string): FactoryLineStep {
  return {
    type: RUN_APP_TYPE,
    app: { app: appId, entrypoint },
  };
}

const PLAN_LINE_DONE_APP_ID = "app-refund-done";
const PLAN_LINE_BACKLOG_APP_ID = "app-refund-backlog";

const BACKLOG_APP: FactoryApp = {
  id: PLAN_LINE_BACKLOG_APP_ID,
  name: "Backlog",
  description: "Scopes work orders before they enter a line.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

const PLAN_LINE_APPS: FactoryApp[] = [
  BACKLOG_APP,
  ...REFUND_FACTORY_APPS.map((app) => {
    if (app.id === "app-refund-planner") {
      return { ...app, name: "Plan" };
    }
    if (app.id === "app-refund-implementer") {
      return { ...app, name: "Implement" };
    }
    if (app.id === "app-refund-verifier") {
      return { ...app, name: "Verify" };
    }
    return app;
  }),
  {
    id: PLAN_LINE_DONE_APP_ID,
    name: "Done",
    description: "Completes or rejects the work order when a pull request merges or closes.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
];

function withPlanLinePhases(line: FactoriesFactoryLine): FactoriesFactoryLine {
  if (line.id !== REFUND_LINE_PLAN_ID) {
    return line;
  }
  return {
    ...line,
    steps: [
      runAppStep("app-refund-planner", "start-plan"),
      runAppStep("app-refund-implementer", "start-implementation"),
      runAppStep("app-refund-verifier", "start-verification"),
      runAppStep(PLAN_LINE_DONE_APP_ID, "start-done"),
    ],
  };
}

function planLineActiveDispatch(
  orderId: string,
  stepExecutions: FactoriesWorkOrderExecution[],
): FactoriesWorkOrderLineDispatch {
  return {
    id: `dispatch-${REFUND_LINE_PLAN_ID}-${orderId}`,
    line: { id: REFUND_LINE_PLAN_ID, name: "plan-and-implement" },
    steps: [
      { name: "Plan", stepIndex: 0 },
      { name: "Implement", stepIndex: 1 },
      { name: "Verify", stepIndex: 2 },
      { name: "Done", stepIndex: 3 },
    ],
    state: "STATE_ACTIVE",
    result: "RESULT_UNKNOWN",
    createdAt: TWO_HOURS_AGO,
    stepExecutions,
  };
}

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
        stepExecutions: (dispatch.stepExecutions ?? []).map((execution) =>
          execution.stepIndex === 1 ? { ...execution, result: "RESULT_PASSED" } : execution,
        ),
      },
    ],
  };
}

export const BOARD_IMPLEMENT_FAILED_ORDER: FactoriesWorkOrder = {
  id: "wo-board-implement-failed",
  number: "106",
  key: "RF-106",
  title: "Fix refund dispatcher timeout loop",
  description:
    "The implement step stopped when backend tests failed on the reconciliation worker. Diagnose the run, then dispatch the line again.",
  state: "STATE_CLOSED",
  result: "RESULT_FAILED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  totalTokens: "6400",
  totalCostCents: "210",
  lineDispatches: [
    {
      ...planLineActiveDispatch("wo-board-implement-failed", [
        {
          id: "exec-failed-plan",
          step: "Plan",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_PASSED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: TWO_HOURS_AGO,
          run: { id: "run-failed-plan", appId: "app-refund-planner", appName: "Plan" },
        },
        {
          id: "exec-failed-implement",
          step: "Implement",
          stepIndex: 1,
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: HOUR_AGO,
          run: { id: LINE_RUN_IMPLEMENT_FAILED_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
        },
      ]),
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
    },
  ],
};

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

const ONBOARDING_FACTORY_LINE: FactoriesFactoryLine = {
  id: REFUND_LINE_ONBOARDING_ID,
  name: "onboarding",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  steps: [runAppStep("app-refund-planner", "start-plan"), runAppStep("app-refund-implementer", "start-implementation")],
};

const FEATURE_PLAN_STEP = "plan";
const FEATURE_IMPLEMENT_STEP = "implement";
const FEATURE_PR_STEP = "pr";
const FEATURE_CI_STEP = "ci";

const FEATURE_STEP_INDEX: Record<string, number> = {
  [FEATURE_PLAN_STEP]: 0,
  [FEATURE_IMPLEMENT_STEP]: 1,
  [FEATURE_PR_STEP]: 2,
  [FEATURE_CI_STEP]: 3,
};

const FEATURE_STEP_LABEL: Record<string, string> = {
  [FEATURE_PLAN_STEP]: "Refund Planner",
  [FEATURE_IMPLEMENT_STEP]: "Refund Implementer",
  [FEATURE_PR_STEP]: "Refund Implementer",
  [FEATURE_CI_STEP]: "Refund Verifier",
};

const FEATURE_DELIVERY_STEPS: NonNullable<FactoriesWorkOrderLineDispatch["steps"]> = [
  { name: FEATURE_STEP_LABEL[FEATURE_PLAN_STEP], stepIndex: 0 },
  { name: FEATURE_STEP_LABEL[FEATURE_IMPLEMENT_STEP], stepIndex: 1 },
  { name: FEATURE_STEP_LABEL[FEATURE_PR_STEP], stepIndex: 2 },
  { name: FEATURE_STEP_LABEL[FEATURE_CI_STEP], stepIndex: 3 },
];

const FEATURE_DELIVERY_LINE: FactoriesFactoryLine = {
  id: REFUND_LINE_FEATURE_ID,
  name: "feature-delivery",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  steps: [
    runAppStep("app-refund-planner", "start-plan"),
    runAppStep("app-refund-implementer", "start-implementation"),
    runAppStep("app-refund-implementer", "start-pull-request"),
    runAppStep("app-refund-verifier", "start-ci-loop"),
  ],
};

function featureLineExecution(
  step: string,
  overrides: Partial<FactoriesWorkOrderExecution> = {},
): FactoriesWorkOrderExecution {
  return {
    id: `exec-feature-${overrides.id ?? step}`,
    step: FEATURE_STEP_LABEL[step] ?? step,
    stepIndex: FEATURE_STEP_INDEX[step] ?? 0,
    state: "STATE_FINISHED",
    result: "RESULT_PASSED",
    createdAt: TWO_HOURS_AGO,
    updatedAt: HOUR_AGO,
    run: {
      id: `run-feature-${step}`,
      appId: "app-refund-implementer",
      appName: "Refund Implementer",
    },
    ...overrides,
  };
}

function featureLineDispatch(stepExecutions: FactoriesWorkOrderExecution[]): FactoriesWorkOrderLineDispatch {
  const state: FactoriesWorkOrderLineDispatchState = stepExecutions.some(
    (execution) => execution.state !== "STATE_FINISHED",
  )
    ? "STATE_ACTIVE"
    : "STATE_FINISHED";

  const lastExecution = stepExecutions[stepExecutions.length - 1];
  const result: FactoriesWorkOrderLineDispatchResult =
    state === "STATE_FINISHED" ? (lastExecution?.result ?? "RESULT_UNKNOWN") : "RESULT_UNKNOWN";

  return {
    id: `dispatch-${REFUND_LINE_FEATURE_ID}-${stepExecutions[0]?.id ?? "empty"}`,
    line: { id: REFUND_LINE_FEATURE_ID, name: "feature-delivery" },
    steps: FEATURE_DELIVERY_STEPS,
    state,
    result,
    createdAt: stepExecutions[0]?.createdAt ?? TWO_HOURS_AGO,
    finishedAt: state === "STATE_FINISHED" ? (lastExecution?.updatedAt ?? HOUR_AGO) : undefined,
    stepExecutions,
  };
}

const FEATURE_RUNNING_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-implement",
  number: "201",
  key: "RF-201",
  title: "Ship ledger retry window",
  description: "Implement the retry window from the plan and open a pull request.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_PLAN_STEP, { id: "plan-1" }),
      featureLineExecution(FEATURE_IMPLEMENT_STEP, {
        id: "impl-1",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Refund Implementer" },
      }),
    ]),
  ],
};

const FEATURE_PR_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-pr",
  number: "202",
  key: "RF-202",
  title: "Open refund schema pull request",
  description: "Plan and implementation passed. Pull request is waiting for review.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: REVIEWER_USER.id, name: REVIEWER_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_PLAN_STEP, { id: "plan-2" }),
      featureLineExecution(FEATURE_IMPLEMENT_STEP, { id: "impl-2" }),
      featureLineExecution(FEATURE_PR_STEP, {
        id: "pr-2",
        state: "STATE_PENDING",
        result: "RESULT_UNKNOWN",
      }),
    ]),
  ],
};

const FEATURE_CI_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-ci",
  number: "203",
  key: "RF-203",
  title: "Run CI on refund enum pull request",
  description: "Pull request is open. CI loop is running.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: OPERATOR_USER.id, name: OPERATOR_USER.name } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_PLAN_STEP, { id: "plan-3" }),
      featureLineExecution(FEATURE_IMPLEMENT_STEP, { id: "impl-3" }),
      featureLineExecution(FEATURE_PR_STEP, { id: "pr-3" }),
      featureLineExecution(FEATURE_CI_STEP, {
        id: "ci-3",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_VERIFY_PASSED_ID, appId: "app-refund-verifier", appName: "Refund Verifier" },
      }),
    ]),
  ],
};

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
      ...(defaultFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? [])
        .map(withPlanPhase)
        .map(withVerifyPhase)
        .map(withWaitingPrReview)
        .map(withDonePhase),
      BOARD_IMPLEMENT_FAILED_ORDER,
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
