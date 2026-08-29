import type {
  FactoriesFactoryLine,
  FactoriesWorkOrder,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoriesWorkOrderLineDispatchResult,
  FactoriesWorkOrderLineDispatchState,
} from "@/api-client";

import {
  ARNOLD_USER,
  HOUR_AGO,
  LAST_WEEK,
  LINE_RUN_IMPLEMENT_FAILED_ID,
  LINE_RUN_IMPLEMENT_ID,
  LINE_RUN_IMPLEMENT_NOTIFY_ID,
  LINE_RUN_VERIFY_PASSED_ID,
  OPERATOR_USER,
  REFUND_LINE_FEATURE_ID,
  REFUND_LINE_ONBOARDING_ID,
  REVIEWER_USER,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
  YESTERDAY,
  minutesAgo,
} from "./factoryPageResponses";
import { INGEST_CREATED_BY, SENTRY_CREATED_BY, SLACK_CREATED_BY } from "./factoryPageWorkOrders";
import { PLAN_LINE_DONE_APP_ID, planLineActiveDispatch, runAppStep } from "./lineMetricsPlanLine";

export const BOARD_IMPLEMENT_NOTIFY_ORDER: FactoriesWorkOrder = {
  id: "wo-board-implement-notify",
  number: "114",
  key: "RF-114",
  title: "Notify on status change after a reopen",
  description: [
    "A user does not get a status-change notification when a task is reopened.",
    "",
    "Send the same notification that a status change already sends, so the assignee sees the reopen.",
  ].join("\n"),
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: minutesAgo(40),
  updatedAt: minutesAgo(12),
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  totalTokens: "4100",
  totalCostCents: "128",
  lineDispatches: [
    planLineActiveDispatch("wo-board-implement-notify", [
      {
        id: "exec-notify-implement",
        step: "Implement",
        stepIndex: 0,
        state: "STATE_FINISHED",
        result: "RESULT_PASSED",
        createdAt: minutesAgo(36),
        updatedAt: minutesAgo(12),
        run: { id: LINE_RUN_IMPLEMENT_NOTIFY_ID, appId: "app-refund-implementer", appName: "Implementation" },
      },
    ]),
  ],
};

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
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  totalTokens: "6400",
  totalCostCents: "210",
  lineDispatches: [
    {
      ...planLineActiveDispatch("wo-board-implement-failed", [
        {
          id: "exec-failed-implement",
          step: "Implement",
          stepIndex: 0,
          state: "STATE_FINISHED",
          result: "RESULT_FAILED",
          createdAt: TWO_HOURS_AGO,
          updatedAt: HOUR_AGO,
          run: { id: LINE_RUN_IMPLEMENT_FAILED_ID, appId: "app-refund-implementer", appName: "Implementation" },
        },
      ]),
      state: "STATE_FINISHED",
      result: "RESULT_FAILED",
    },
  ],
};

function boardDoneOrder({
  id,
  number,
  key,
  title,
  description,
  result,
  dispatchResult,
  doneResult,
  updatedAt,
  assignee,
  createdBy = INGEST_CREATED_BY,
}: {
  id: string;
  number: string;
  key: string;
  title: string;
  description: string;
  result: FactoriesWorkOrder["result"];
  dispatchResult: FactoriesWorkOrderLineDispatchResult;
  doneResult: FactoriesWorkOrderExecution["result"];
  updatedAt: string;
  assignee: { id: string; name: string };
  createdBy?: FactoriesWorkOrder["createdBy"];
}): FactoriesWorkOrder {
  return {
    id,
    number,
    key,
    title,
    description,
    state: "STATE_CLOSED",
    result,
    createdAt: LAST_WEEK,
    updatedAt,
    createdBy,
    assignees: [assignee],
    lineDispatches: [
      {
        ...planLineActiveDispatch(id, [
          {
            id: `exec-done-implement-${id}`,
            step: "Implement",
            stepIndex: 0,
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: LAST_WEEK,
            updatedAt: LAST_WEEK,
            run: { id: `run-done-implement-${id}`, appId: "app-refund-implementer", appName: "Implement" },
          },
          {
            id: `exec-done-verify-${id}`,
            step: "Verify",
            stepIndex: 1,
            state: "STATE_FINISHED",
            result: "RESULT_PASSED",
            createdAt: LAST_WEEK,
            updatedAt: LAST_WEEK,
            run: { id: `run-done-verify-${id}`, appId: "app-refund-verifier", appName: "Verify" },
          },
          {
            id: `exec-done-${id}`,
            step: "Done",
            stepIndex: 2,
            state: "STATE_FINISHED",
            result: doneResult,
            createdAt: updatedAt,
            updatedAt,
            run: { id: `run-done-${id}`, appId: PLAN_LINE_DONE_APP_ID, appName: "Done" },
          },
        ]),
        state: "STATE_FINISHED",
        result: dispatchResult,
      },
    ],
  };
}

export const BOARD_DONE_COMPLETED_SLA: FactoriesWorkOrder = boardDoneOrder({
  id: "wo-board-done-sla",
  number: "110",
  key: "RF-110",
  title: "Publish refund SLA dashboard",
  description: "Publish the refund SLA dashboard so support can see provider latency by day.",
  result: "RESULT_COMPLETED",
  dispatchResult: "RESULT_PASSED",
  doneResult: "RESULT_PASSED",
  updatedAt: TWO_HOURS_AGO,
  assignee: { id: ARNOLD_USER.id, name: ARNOLD_USER.name },
});

export const BOARD_DONE_COMPLETED_PLAYBOOK: FactoriesWorkOrder = boardDoneOrder({
  id: "wo-board-done-playbook",
  number: "111",
  key: "RF-111",
  title: "Document provider timeout playbook",
  description: "Write the playbook for provider timeouts so on-call can retry or fail closed.",
  result: "RESULT_COMPLETED",
  dispatchResult: "RESULT_PASSED",
  doneResult: "RESULT_PASSED",
  updatedAt: YESTERDAY,
  assignee: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME },
  createdBy: SLACK_CREATED_BY,
});

export const BOARD_DONE_REJECTED_ORDER: FactoriesWorkOrder = boardDoneOrder({
  id: "wo-board-done-rejected",
  number: "112",
  key: "RF-112",
  title: "Replace the refund batch exporter",
  description: "Replace the batch exporter. A reviewer rejected the change because it drops the audit columns.",
  result: "RESULT_REJECTED",
  dispatchResult: "RESULT_PASSED",
  doneResult: "RESULT_PASSED",
  updatedAt: HOUR_AGO,
  assignee: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
});

export const BOARD_DONE_CANCELED_ORDER: FactoriesWorkOrder = boardDoneOrder({
  id: "wo-board-done-canceled",
  number: "113",
  key: "RF-113",
  title: "Migrate refunds to the v2 provider API",
  description: "The v2 migration stopped when the provider delayed the cutover. The task was canceled.",
  result: "RESULT_UNSPECIFIED",
  dispatchResult: "RESULT_CANCELLED",
  doneResult: "RESULT_CANCELLED",
  updatedAt: TWO_HOURS_AGO,
  assignee: { id: OPERATOR_USER.id, name: OPERATOR_USER.name },
});

export const ONBOARDING_FACTORY_LINE: FactoriesFactoryLine = {
  id: REFUND_LINE_ONBOARDING_ID,
  name: "onboarding",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  steps: [runAppStep("app-refund-implementer", "start-implementation")],
};

const FEATURE_IMPLEMENT_STEP = "implement";
const FEATURE_PR_STEP = "pr";
const FEATURE_CI_STEP = "ci";

const FEATURE_STEP_INDEX: Record<string, number> = {
  [FEATURE_IMPLEMENT_STEP]: 0,
  [FEATURE_PR_STEP]: 1,
  [FEATURE_CI_STEP]: 2,
};

const FEATURE_STEP_LABEL: Record<string, string> = {
  [FEATURE_IMPLEMENT_STEP]: "Implementation",
  [FEATURE_PR_STEP]: "Implementation",
  [FEATURE_CI_STEP]: "Risk Assessment",
};

const FEATURE_DELIVERY_STEPS: NonNullable<FactoriesWorkOrderLineDispatch["steps"]> = [
  { name: FEATURE_STEP_LABEL[FEATURE_IMPLEMENT_STEP], stepIndex: 0 },
  { name: FEATURE_STEP_LABEL[FEATURE_PR_STEP], stepIndex: 1 },
  { name: FEATURE_STEP_LABEL[FEATURE_CI_STEP], stepIndex: 2 },
];

export const FEATURE_DELIVERY_LINE: FactoriesFactoryLine = {
  id: REFUND_LINE_FEATURE_ID,
  name: "feature-delivery",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  steps: [
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
      appName: "Implementation",
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

export const FEATURE_RUNNING_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-implement",
  number: "201",
  key: "RF-201",
  title: "Ship ledger retry window",
  description: "Implement the retry window and open a pull request.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: SLACK_CREATED_BY,
  origin: {
    url: "https://acme.slack.com/archives/C0REFUNDS/p1710000000000000",
    label: "acme#C0REFUNDS",
  },
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_IMPLEMENT_STEP, {
        id: "impl-1",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_IMPLEMENT_ID, appId: "app-refund-implementer", appName: "Implementation" },
      }),
    ]),
  ],
};

export const FEATURE_PR_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-pr",
  number: "202",
  key: "RF-202",
  title: "Open refund schema pull request",
  description: "Implementation passed. Pull request is waiting for review.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: INGEST_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_IMPLEMENT_STEP, { id: "impl-2" }),
      featureLineExecution(FEATURE_PR_STEP, {
        id: "pr-2",
        state: "STATE_PENDING",
        result: "RESULT_UNKNOWN",
      }),
    ]),
  ],
};

export const FEATURE_CI_WORK_ORDER: FactoriesWorkOrder = {
  id: "wo-feature-ci",
  number: "203",
  key: "RF-203",
  title: "Run CI on refund enum pull request",
  description: "Pull request is open. CI loop is running.",
  state: "STATE_OPEN",
  result: "RESULT_UNSPECIFIED",
  createdAt: YESTERDAY,
  updatedAt: HOUR_AGO,
  createdBy: SENTRY_CREATED_BY,
  assignees: [{ id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME }],
  lineDispatches: [
    featureLineDispatch([
      featureLineExecution(FEATURE_IMPLEMENT_STEP, { id: "impl-3" }),
      featureLineExecution(FEATURE_PR_STEP, { id: "pr-3" }),
      featureLineExecution(FEATURE_CI_STEP, {
        id: "ci-3",
        state: "STATE_STARTED",
        result: "RESULT_UNKNOWN",
        run: { id: LINE_RUN_VERIFY_PASSED_ID, appId: "app-refund-verifier", appName: "Risk Assessment" },
      }),
    ]),
  ],
};
