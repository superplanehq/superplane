import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderLineDispatchResult,
  FactoriesWorkOrderLineDispatchState,
  FactoryLineStep,
} from "@/api-client";

import {
  HOUR_AGO,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  OPERATOR_USER,
  PRIMARY_FACTORY_ID,
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
      lines: [...(factory.lines ?? []), ONBOARDING_FACTORY_LINE, FEATURE_DELIVERY_LINE],
    };
  }),
  workOrdersByFactoryId: {
    ...defaultFactoriesFixture.workOrdersByFactoryId,
    [PRIMARY_FACTORY_ID]: [
      ...(defaultFactoriesFixture.workOrdersByFactoryId[PRIMARY_FACTORY_ID] ?? []),
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
};
