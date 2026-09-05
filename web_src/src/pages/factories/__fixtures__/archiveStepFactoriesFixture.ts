import type { FactoriesFactoryLine, FactoriesWorkOrder, FactoryApp } from "@/api-client";

import {
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_PLAN_ID,
  RUNNING_WORK_ORDER,
  defaultFactoriesFixture,
  type FactoriesFixture,
} from "./factoryPageResponses";
import { PLAN_LINE_APPS, runAppStep } from "./lineMetricsPlanLine";

/** Plan then Implement. Archive Step shows on both while two steps remain. */
function withArchiveReviewSteps(line: FactoriesFactoryLine): FactoriesFactoryLine {
  if (line.id !== REFUND_LINE_PLAN_ID) {
    return line;
  }
  return {
    ...line,
    steps: [
      runAppStep("app-refund-planner", "start-plan"),
      runAppStep("app-refund-implementer", "start-implementation"),
    ],
    columnColors: { backlog: "lime", "phase-0": "sky", "phase-1": "teal" },
  };
}

function withSingleArchiveReviewStep(line: FactoriesFactoryLine): FactoriesFactoryLine {
  if (line.id !== REFUND_LINE_PLAN_ID) {
    return line;
  }
  return {
    ...line,
    steps: [runAppStep("app-refund-implementer", "start-implementation")],
    columnColors: { backlog: "lime", "phase-0": "teal" },
  };
}

const ARCHIVE_REVIEW_APPS: FactoryApp[] = PLAN_LINE_APPS.map((app) =>
  app.id === "app-refund-planner" ? { ...app, name: "Plan" } : app,
);

function archiveReviewFixture(
  mapLine: (line: FactoriesFactoryLine) => FactoriesFactoryLine,
  workOrders: FactoriesWorkOrder[],
): FactoriesFixture {
  return {
    ...defaultFactoriesFixture,
    factories: defaultFactoriesFixture.factories.map((factory) => {
      if (factory.id !== PRIMARY_FACTORY_ID) {
        return factory;
      }
      return {
        ...factory,
        lines: (factory.lines ?? []).map(mapLine),
      };
    }),
    appsByFactoryId: {
      ...defaultFactoriesFixture.appsByFactoryId,
      [PRIMARY_FACTORY_ID]: ARCHIVE_REVIEW_APPS,
    },
    workOrdersByFactoryId: {
      ...defaultFactoriesFixture.workOrdersByFactoryId,
      [PRIMARY_FACTORY_ID]: workOrders,
    },
  };
}

const BACKLOG_ONLY_ORDERS = [DRAFT_WORK_ORDER, INGEST_DRAFT_WORK_ORDER];

/** Empty Plan and Implement. Open a phase menu and archive the step. */
export const archiveStepEmptyFactoriesFixture = archiveReviewFixture(withArchiveReviewSteps, BACKLOG_ONLY_ORDERS);

/**
 * Implement has a running task. Plan is empty. Archive on Implement is
 * blocked. Archive on Plan asks for confirmation.
 */
export const archiveStepBlockedFactoriesFixture = archiveReviewFixture(withArchiveReviewSteps, [
  ...BACKLOG_ONLY_ORDERS,
  RUNNING_WORK_ORDER,
]);

/** One Implement step. Archive Step explains that the last step stays. */
export const archiveStepSingleStepFactoriesFixture = archiveReviewFixture(
  withSingleArchiveReviewStep,
  BACKLOG_ONLY_ORDERS,
);
