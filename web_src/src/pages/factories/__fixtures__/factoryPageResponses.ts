import type {
  FactoriesFactory,
  FactoriesFactoryLine,
  MeNotificationSettings,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoryApp,
  FactoryLineStep,
  SuperplaneUsersUser,
} from "@/api-client";

import type { FactoriesWorkOrderCheck } from "@/api-client";
import { DEFAULT_FACTORY_USAGE, EMPTY_USAGE_REPORT, type StorybookUsageReport } from "./usageReportFixtures";
import {
  EMPTY_FACTORY_ID,
  FACTORIES_ORGANIZATION_ID,
  LAST_WEEK,
  PRIMARY_FACTORY_ID,
  REFUND_LINE_HOTFIX_ID,
  REFUND_LINE_PLAN_ID,
  YESTERDAY,
  type ORGANIZATION_USERS,
} from "./factoryPageIds";
import {
  APPROVAL_WORK_ORDER,
  CLOSED_FAILED_WORK_ORDER,
  CLOSED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  FAILED_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER,
  SENTRY_DRAFT_WORK_ORDER,
  SLACK_DRAFT_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  QUESTION_WORK_ORDER,
  RUNNING_WORK_ORDER,
} from "./factoryPageWorkOrders";

export * from "./factoryPageIds";
export * from "./factoryPageWorkOrders";

export function toStorybookOrganizationUser(user: (typeof ORGANIZATION_USERS)[number]): SuperplaneUsersUser {
  const avatarUrl = "avatarUrl" in user ? user.avatarUrl : undefined;
  return {
    metadata: { id: user.id, email: user.email },
    spec: { displayName: user.name },
    ...(avatarUrl ? { status: { accountProviders: [{ avatarUrl }] } } : {}),
  };
}

const RUN_APP_TYPE = "runApp";

function runAppStep(appId: string, entrypoint: string): FactoryLineStep {
  return {
    type: RUN_APP_TYPE,
    app: { app: appId, entrypoint },
  };
}

export const REFUND_FACTORY_APPS: FactoryApp[] = [
  {
    id: "app-refund-planner",
    name: "Refund Planner",
    description: "Plans reconciliation work across ledger + payment services.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-refund-implementer",
    name: "Refund Implementer",
    description: "Applies the plan across affected repos and opens PRs.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-refund-verifier",
    name: "Refund Verifier",
    description: "Runs verification suites and gates merge.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-pr-closure",
    name: "PR Closure",
    description: "Closes the work order when the pull request merges or is closed.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
];

export const REFUND_FACTORY_LINES: FactoriesFactoryLine[] = [
  {
    id: REFUND_LINE_PLAN_ID,
    name: "plan-and-implement",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [
      runAppStep("app-refund-planner", "start-plan"),
      runAppStep("app-refund-implementer", "start-implementation"),
      runAppStep("app-refund-verifier", "start-verification"),
    ],
  },
  {
    id: REFUND_LINE_HOTFIX_ID,
    name: "hotfix",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [runAppStep("app-refund-verifier", "start-verification")],
  },
];

export const REFUND_FACTORY: FactoriesFactory = {
  id: PRIMARY_FACTORY_ID,
  name: "Semaphore",
  key: "RF",
  description:
    "Handles reconciliation work: plan a change, implement across affected services, and verify with regression suites.",
  lines: REFUND_FACTORY_LINES,
};

export const EMPTY_FACTORY: FactoriesFactory = {
  id: EMPTY_FACTORY_ID,
  name: "SuperPlane",
  key: "PF",
  description: "New factory. No lines or work orders configured yet.",
  lines: [],
};

export const DEFAULT_WORK_ORDERS: FactoriesWorkOrder[] = [
  OPEN_WORK_ORDER,
  OPEN_WORK_ORDER_SECONDARY,
  QUESTION_WORK_ORDER,
  APPROVAL_WORK_ORDER,
  RUNNING_WORK_ORDER,
  FAILED_WORK_ORDER,
  DRAFT_WORK_ORDER,
  INGEST_DRAFT_WORK_ORDER,
  SENTRY_DRAFT_WORK_ORDER,
  SLACK_DRAFT_WORK_ORDER,
  CLOSED_WORK_ORDER,
  PR_CLOSURE_COMPLETED_WORK_ORDER,
  CLOSED_FAILED_WORK_ORDER,
];

export interface FactoriesFixture {
  organizationId: string;
  factories: FactoriesFactory[];
  workOrdersByFactoryId: Record<string, FactoriesWorkOrder[]>;
  appsByFactoryId: Record<string, FactoryApp[]>;
  usageByFactoryId?: Record<string, StorybookUsageReport>;
  organizationLlmSpend?: StorybookUsageReport;
  /** Per-user notification settings backing `/api/v1/me/notification-settings`. */
  notificationSettings?: MeNotificationSettings;
  /**
   * Per-order activity timelines. When an order id is absent, the handlers
   * fall back to `DEFAULT_EVENTS_BY_ORDER_ID` from `factoryPageEventFixtures`.
   */
  eventsByOrderId?: Record<string, FactoriesWorkOrderEvent[]>;
  /** Per-order artifacts; same fallback pattern as `eventsByOrderId`. */
  artifactsByOrderId?: Record<string, FactoriesWorkOrderArtifact[]>;
  /** Per-order checks (automation-reported scores); same fallback pattern as `eventsByOrderId`. */
  checksByOrderId?: Record<string, FactoriesWorkOrderCheck[]>;
}

export const defaultFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [REFUND_FACTORY, EMPTY_FACTORY],
  workOrdersByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_WORK_ORDERS,
    [EMPTY_FACTORY_ID]: [],
  },
  appsByFactoryId: {
    [PRIMARY_FACTORY_ID]: REFUND_FACTORY_APPS,
    [EMPTY_FACTORY_ID]: [],
  },
  usageByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_USAGE,
    [EMPTY_FACTORY_ID]: EMPTY_USAGE_REPORT,
  },
  organizationLlmSpend: DEFAULT_FACTORY_USAGE,
};
