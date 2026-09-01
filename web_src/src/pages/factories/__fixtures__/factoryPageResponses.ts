import type {
  FactoriesDescribeFactoryVelocityResponse,
  FactoriesFactory,
  FactoriesFactoryIntake,
  FactoriesFactoryIntakeRun,
  FactoriesFactoryLine,
  FactoriesFactoryPullRequest,
  MeNotificationSettings,
  FactoriesWorkOrder,
  FactoriesWorkOrderArtifact,
  FactoriesWorkOrderEvent,
  FactoryApp,
  FactoryLineStep,
  SuperplaneUsersUser,
} from "@/api-client";

import type { FactoriesWorkOrderCheck } from "@/api-client";
import type { BacklogIntakeItemCatalog } from "../pages/backlogIntakeItems";
import { DEFAULT_FACTORY_USAGE, EMPTY_USAGE_REPORT, type StorybookUsageReport } from "./usageReportFixtures";
import { DEFAULT_FACTORY_VELOCITY } from "./velocityReportFixtures";
import {
  ACME_ONBOARDING_FACTORY_ID,
  ACME_ONBOARDING_LINE_ID,
  GITHUB_ISSUES_INTAKE_APP_ID,
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

export const GITHUB_ISSUES_INTAKE_APP: FactoryApp = {
  id: GITHUB_ISSUES_INTAKE_APP_ID,
  name: "GitHub issue intake",
  description: "Listens for GitHub issues and creates backlog tasks.",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

export const GITHUB_ISSUES_INTAKE_ID = "intake-github-issues";

export const GITHUB_ISSUES_INTAKE: FactoriesFactoryIntake = {
  id: GITHUB_ISSUES_INTAKE_ID,
  canvasId: GITHUB_ISSUES_INTAKE_APP_ID,
  name: "GitHub issues",
  description: "Listens for GitHub issues and creates backlog tasks.",
  source: "SOURCE_GITHUB_ISSUES",
  settings: {
    confidencePct: 65,
    labels: [],
    labelFilterMode: "LABEL_FILTER_MODE_INCLUDE",
    assignment: "ASSIGNMENT_ANY",
  },
  healthy: true,
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
};

function minutesBefore(minutes: number): string {
  return new Date(Date.now() - minutes * 60_000).toISOString();
}

/** Two tickets still in analysis, plus scored runs for the Runs tab. */
export const GITHUB_ISSUES_INTAKE_RUNS: FactoriesFactoryIntakeRun[] = [
  {
    id: "intake-run-analyzing-1",
    title: "Handle duplicate refunds on retry",
    placement: "PLACEMENT_ANALYZING",
    createdAt: minutesBefore(4),
  },
  {
    id: "intake-run-analyzing-2",
    title: "Return 409 when the invoice is already paid",
    placement: "PLACEMENT_ANALYZING",
    createdAt: minutesBefore(2),
  },
  {
    id: "intake-run-progressed",
    title: "Show a clearer empty state on the billing page",
    confidencePct: 94,
    placement: "PLACEMENT_PROGRESSED",
    stage: "implement",
    createdAt: minutesBefore(180),
    analyzedAt: minutesBefore(170),
  },
  {
    id: "intake-run-backlog",
    title: "Upgrade the Node 20 base image",
    confidencePct: 81,
    placement: "PLACEMENT_BACKLOG",
    createdAt: minutesBefore(90),
    analyzedAt: minutesBefore(80),
  },
  {
    id: "intake-run-below-threshold",
    title: "Document the refund webhook contract",
    confidencePct: 52,
    placement: "PLACEMENT_BELOW_THRESHOLD",
    createdAt: minutesBefore(8),
    analyzedAt: minutesBefore(5),
  },
];

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
    description: "Closes the task when the pull request merges or is closed.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  GITHUB_ISSUES_INTAKE_APP,
];

export const REFUND_FACTORY_LINES: FactoriesFactoryLine[] = [
  {
    id: REFUND_LINE_PLAN_ID,
    name: "plan-and-implement",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
    steps: [
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
    "Handles reconciliation work: implement a change across affected services, and verify with regression suites.",
  lines: REFUND_FACTORY_LINES,
  onboarding: { completedAt: LAST_WEEK },
};

export const EMPTY_FACTORY: FactoriesFactory = {
  id: EMPTY_FACTORY_ID,
  name: "SuperPlane",
  key: "PF",
  description: "New factory. No lines or tasks configured yet.",
  lines: [],
};

const ACME_ONBOARDING_DONE_APP_ID = "app-acme-done";
const ACME_ONBOARDING_BACKLOG_APP_ID = "app-acme-backlog";

export const ACME_ONBOARDING_APPS: FactoryApp[] = [
  {
    id: ACME_ONBOARDING_BACKLOG_APP_ID,
    name: "Backlog",
    description: "Scopes tasks before they enter a line.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: "app-acme-implementer",
    name: "Implement",
    description: "Implements the plan.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  {
    id: ACME_ONBOARDING_DONE_APP_ID,
    name: "Done",
    description: "Completes the task.",
    createdAt: LAST_WEEK,
    updatedAt: YESTERDAY,
  },
  GITHUB_ISSUES_INTAKE_APP,
];

export const ACME_ONBOARDING_LINE: FactoriesFactoryLine = {
  id: ACME_ONBOARDING_LINE_ID,
  name: "Plan and Implement",
  createdAt: LAST_WEEK,
  updatedAt: YESTERDAY,
  steps: [
    runAppStep("app-acme-implementer", "start-implementation"),
    runAppStep(ACME_ONBOARDING_DONE_APP_ID, "start-done"),
  ],
};

export const ACME_ONBOARDING_FACTORY: FactoriesFactory = {
  id: ACME_ONBOARDING_FACTORY_ID,
  name: "Acme onboarding",
  key: "AO",
  description: "Empty first-run workspace. The board has no tickets yet.",
  lines: [ACME_ONBOARDING_LINE],
  onboarding: { completedAt: LAST_WEEK },
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
  /** Intakes the workspace declared. Created intakes are appended here. */
  intakesByFactoryId?: Record<string, FactoriesFactoryIntake[]>;
  /** Runs the intake produced, keyed by intake id. */
  intakeRunsByIntakeId?: Record<string, FactoriesFactoryIntakeRun[]>;
  usageByFactoryId?: Record<string, StorybookUsageReport>;
  /**
   * Velocity reports keyed by factory, then by requested period in days. A
   * period without an entry falls back to the empty report.
   */
  velocityByFactoryId?: Record<string, Record<number, FactoriesDescribeFactoryVelocityResponse>>;
  organizationWorkspaceUsage?: StorybookUsageReport;
  hostedCreditProducts?: Array<{ id: string; name: string; amountCents: string }>;
  /** Per-user notification settings backing `/api/v1/me/notification-settings`. */
  notificationSettings?: MeNotificationSettings;
  /**
   * Per-order activity timelines. When an order id is absent, the handlers
   * fall back to `DEFAULT_EVENTS_BY_ORDER_ID` from `factoryPageEventFixtures`.
   */
  eventsByOrderId?: Record<string, FactoriesWorkOrderEvent[]>;
  /** Per-order artifacts; same fallback pattern as `eventsByOrderId`. */
  artifactsByOrderId?: Record<string, FactoriesWorkOrderArtifact[]>;
  /** Per-order pull requests; same fallback pattern as `eventsByOrderId`. */
  pullRequestsByOrderId?: Record<string, FactoriesFactoryPullRequest[]>;
  /** Per-order checks (automation-reported scores); same fallback pattern as `eventsByOrderId`. */
  checksByOrderId?: Record<string, FactoriesWorkOrderCheck[]>;
  /** Storybook-only intake items for the Backlog create search. */
  intakeItemCatalog?: BacklogIntakeItemCatalog;
}

export const defaultFactoriesFixture: FactoriesFixture = {
  organizationId: FACTORIES_ORGANIZATION_ID,
  factories: [REFUND_FACTORY, EMPTY_FACTORY, ACME_ONBOARDING_FACTORY],
  workOrdersByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_WORK_ORDERS,
    [EMPTY_FACTORY_ID]: [],
    [ACME_ONBOARDING_FACTORY_ID]: [],
  },
  appsByFactoryId: {
    [PRIMARY_FACTORY_ID]: REFUND_FACTORY_APPS,
    [EMPTY_FACTORY_ID]: [],
    [ACME_ONBOARDING_FACTORY_ID]: ACME_ONBOARDING_APPS,
  },
  intakesByFactoryId: {
    [PRIMARY_FACTORY_ID]: [GITHUB_ISSUES_INTAKE],
    [EMPTY_FACTORY_ID]: [],
    [ACME_ONBOARDING_FACTORY_ID]: [GITHUB_ISSUES_INTAKE],
  },
  intakeRunsByIntakeId: {
    [GITHUB_ISSUES_INTAKE_ID]: GITHUB_ISSUES_INTAKE_RUNS,
  },
  usageByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_USAGE,
    [EMPTY_FACTORY_ID]: EMPTY_USAGE_REPORT,
    [ACME_ONBOARDING_FACTORY_ID]: EMPTY_USAGE_REPORT,
  },
  velocityByFactoryId: {
    [PRIMARY_FACTORY_ID]: DEFAULT_FACTORY_VELOCITY,
  },
  organizationWorkspaceUsage: DEFAULT_FACTORY_USAGE,
};
