import type { FactoriesFactoryPullRequest, FactoriesWorkOrderArtifact } from "@/api-client";

import {
  CLOSED_WORK_ORDER,
  FACTORIES_ORGANIZATION_ID,
  HOUR_AGO,
  OPEN_WORK_ORDER,
  PRIMARY_FACTORY_ID,
  REFUND_FACTORY_LINES,
  REVIEWER_USER,
  defaultFactoriesFixture,
  type FactoriesFixture,
} from "./factoryPageResponses";

export const OPEN_WORK_ORDER_PULL_REQUESTS: FactoriesFactoryPullRequest[] = [
  {
    id: "art-pr-1",
    workOrderId: OPEN_WORK_ORDER.id,
    number: "482",
    url: "https://github.com/example/ledger/pull/482",
    title: "Fix duplicate refund on retry",
    state: "STATE_OPEN",
  },
];

export const OPEN_WORK_ORDER_ARTIFACTS: FactoriesWorkOrderArtifact[] = [
  {
    id: "art-md-1",
    type: "TYPE_MARKDOWN",
    data: {
      title: "Investigation notes",
      body: "Retry policy exceeded idempotency window when the ledger writer was under load; details captured in the design doc.",
    },
    createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
    createdAt: HOUR_AGO,
  },
  {
    id: "art-branch-1",
    type: "TYPE_BRANCH",
    data: {
      name: "feature/refund-retry",
      url: "https://github.com/example/ledger/tree/feature/refund-retry",
    },
    createdBy: { id: REVIEWER_USER.id, name: REVIEWER_USER.name },
    createdAt: HOUR_AGO,
  },
];

export const emptyFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [],
  workOrdersByFactoryId: {},
  appsByFactoryId: {},
};

/** Same shape as the default fixture but with only closed orders in the primary factory. */
export const emptyWorkOrdersFactoriesFixture: FactoriesFixture = {
  ...defaultFactoriesFixture,
  workOrdersByFactoryId: {
    ...defaultFactoriesFixture.workOrdersByFactoryId,
    [PRIMARY_FACTORY_ID]: [CLOSED_WORK_ORDER],
  },
};

const PLAN_LINE_ID = REFUND_FACTORY_LINES[0]?.id;

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
        if (line.id !== PLAN_LINE_ID) {
          return line;
        }
        return {
          ...line,
          steps: [
            ...(line.steps ?? []),
            { name: "release", type: "runApp", app: { app: "app-refund-verifier", entrypoint: "start-verification" } },
            { name: "observe", type: "runApp", app: { app: "app-refund-implementer", entrypoint: "start-observe" } },
          ],
        };
      }),
    };
  }),
};
