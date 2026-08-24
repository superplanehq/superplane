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
  name: "Backlog",
  description: "Scopes work orders before they enter a line.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

export const PLAN_LINE_APPS: FactoryApp[] = [
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

export function withPlanLinePhases(line: FactoriesFactoryLine): FactoriesFactoryLine {
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

export function planLineActiveDispatch(
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
