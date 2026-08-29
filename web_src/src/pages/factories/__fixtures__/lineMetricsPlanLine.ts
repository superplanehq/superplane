import type {
  FactoriesFactoryLine,
  FactoriesWorkOrderExecution,
  FactoriesWorkOrderLineDispatch,
  FactoryApp,
  FactoryLineStep,
} from "@/api-client";

import { LAST_WEEK, REFUND_FACTORY_APPS, REFUND_LINE_PLAN_ID, TWO_HOURS_AGO, YESTERDAY } from "./factoryPageResponses";

const RUN_APP_TYPE = "runApp";

export function runAppStep(appId: string, entrypoint: string): FactoryLineStep {
  return {
    type: RUN_APP_TYPE,
    app: { app: appId, entrypoint },
  };
}

export const PLAN_LINE_DONE_APP_ID = "app-refund-done";
const PLAN_LINE_BACKLOG_APP_ID = "app-refund-backlog";

export const BACKLOG_APP: FactoryApp = {
  id: PLAN_LINE_BACKLOG_APP_ID,
  name: "Ingest",
  description: "Create a task when a GitHub issue gets the factory label or is assigned to the SuperPlane agent.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

export const SENTRY_INTAKE_APP: FactoryApp = {
  id: "app-refund-sentry",
  name: "Sentry",
  description: "Create a task when Sentry opens an issue.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

export const SLACK_INTAKE_APP: FactoryApp = {
  id: "app-refund-slack",
  name: "Slack",
  description: "Create a task when someone mentions the SuperPlane agent in Slack.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

export const PLAN_LINE_APPS: FactoryApp[] = [
  BACKLOG_APP,
  SENTRY_INTAKE_APP,
  SLACK_INTAKE_APP,
  ...REFUND_FACTORY_APPS.map((app) => {
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
    description: "Completes or rejects the task when a pull request merges or closes.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
];

export function withPlanLinePhases(line: FactoriesFactoryLine): FactoriesFactoryLine {
  if (line.id !== REFUND_LINE_PLAN_ID) {
    return line;
  }
  return {
    ...line,
    steps: [
      runAppStep("app-refund-implementer", "start-implementation"),
      runAppStep("app-refund-verifier", "start-verification"),
      runAppStep(PLAN_LINE_DONE_APP_ID, "start-done"),
    ],
  };
}

export function planLineActiveDispatch(
  orderId: string,
  stepExecutions: FactoriesWorkOrderExecution[],
): FactoriesWorkOrderLineDispatch {
  return {
    id: `dispatch-${REFUND_LINE_PLAN_ID}-${orderId}`,
    line: { id: REFUND_LINE_PLAN_ID, name: "plan-and-implement" },
    steps: [
      { name: "Implement", stepIndex: 0 },
      { name: "Verify", stepIndex: 1 },
      { name: "Done", stepIndex: 2 },
    ],
    state: "STATE_ACTIVE",
    result: "RESULT_UNKNOWN",
    createdAt: TWO_HOURS_AGO,
    stepExecutions,
  };
}
