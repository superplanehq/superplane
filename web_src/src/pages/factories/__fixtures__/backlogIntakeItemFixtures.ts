import type { FactoriesFactoryIntake } from "@/api-client";
import linearIcon from "@/assets/icons/integrations/linear.svg";

import { lineMetricsFactoriesFixture } from "./lineMetricsFactoriesFixture";
import {
  GITHUB_ISSUES_INTAKE,
  GITHUB_ISSUES_INTAKE_ID,
  PRIMARY_FACTORY_ID,
  type FactoriesFixture,
} from "./factoryPageResponses";
import type { BacklogIntakeItem, BacklogIntakeItemCatalog } from "../pages/backlogIntakeItems";

export const SENTRY_EXCEPTIONS_INTAKE_ID = "intake-sentry-exceptions";
export const PAGERDUTY_INCIDENTS_INTAKE_ID = "intake-pagerduty-incidents";
export const LINEAR_ISSUES_INTAKE_ID = "intake-linear-issues";

export const SENTRY_EXCEPTIONS_INTAKE: FactoriesFactoryIntake = {
  id: SENTRY_EXCEPTIONS_INTAKE_ID,
  canvasId: "app-sentry-intake",
  name: "Sentry exceptions",
  description: "Unresolved errors from production.",
  source: "SOURCE_SENTRY_EXCEPTIONS",
  healthy: true,
};

export const PAGERDUTY_INCIDENTS_INTAKE: FactoriesFactoryIntake = {
  id: PAGERDUTY_INCIDENTS_INTAKE_ID,
  canvasId: "app-pagerduty-intake",
  name: "PagerDuty incidents",
  description: "Firing incidents that need a task.",
  source: "SOURCE_PAGERDUTY_INCIDENTS",
  healthy: true,
};

export const LINEAR_ISSUES_INTAKE: FactoriesFactoryIntake = {
  id: LINEAR_ISSUES_INTAKE_ID,
  canvasId: "app-linear-intake",
  name: "Linear issues",
  description: "Issues from the product team.",
  source: "SOURCE_UNSPECIFIED",
  healthy: true,
};

const githubItems: BacklogIntakeItem[] = [
  {
    id: "gh-issue-12",
    intakeId: GITHUB_ISSUES_INTAKE_ID,
    key: "#12",
    title: "Handle duplicate refunds on retry",
    body: "A second refund request posts a second credit.",
  },
  {
    id: "gh-issue-13",
    intakeId: GITHUB_ISSUES_INTAKE_ID,
    key: "#13",
    title: "Return 409 when the invoice is already paid",
    body: "The API returns 500 after a double submit.",
  },
  {
    id: "gh-issue-14",
    intakeId: GITHUB_ISSUES_INTAKE_ID,
    key: "#14",
    title: "Upgrade the Node 20 base image",
    body: "The image is past the support window.",
  },
  {
    id: "gh-issue-15",
    intakeId: GITHUB_ISSUES_INTAKE_ID,
    key: "#15",
    title: "Show a clearer empty state on the billing page",
    body: "The empty list looks like an error.",
  },
  {
    id: "gh-issue-16",
    intakeId: GITHUB_ISSUES_INTAKE_ID,
    key: "#16",
    title: "Document the refund webhook contract",
    body: "Partners ask which fields are required.",
  },
];

const sentryItems: BacklogIntakeItem[] = [
  {
    id: "sentry-1",
    intakeId: SENTRY_EXCEPTIONS_INTAKE_ID,
    key: "PROJ-9",
    title: "TypeError: cart is undefined",
    body: "Checkout crashes when the cart cookie is missing.",
  },
  {
    id: "sentry-2",
    intakeId: SENTRY_EXCEPTIONS_INTAKE_ID,
    key: "PROJ-18",
    title: "RangeError: Maximum call stack",
    body: "The price formatter recurses on bundled SKUs.",
  },
];

const pagerdutyItems: BacklogIntakeItem[] = [
  {
    id: "pd-1",
    intakeId: PAGERDUTY_INCIDENTS_INTAKE_ID,
    key: "PD-441",
    title: "Payments API error budget burned",
    body: "Latency crossed the page threshold for 12 minutes.",
  },
  {
    id: "pd-2",
    intakeId: PAGERDUTY_INCIDENTS_INTAKE_ID,
    key: "PD-442",
    title: "Webhook consumer lag",
    body: "The refund consumer is 20 minutes behind.",
  },
];

const linearItems: BacklogIntakeItem[] = [
  {
    id: "lin-1",
    intakeId: LINEAR_ISSUES_INTAKE_ID,
    key: "LIN-204",
    title: "Triage bugs from the support inbox",
    body: "Move confirmed product bugs onto the engineering board.",
  },
  {
    id: "lin-2",
    intakeId: LINEAR_ISSUES_INTAKE_ID,
    key: "LIN-211",
    title: "Clarify refund copy in the help center",
    body: "Customers still expect an email receipt after a refund.",
  },
];

export const defaultBacklogIntakeItemCatalog: BacklogIntakeItemCatalog = {
  items: githubItems,
};

export const severalIntakeItemCatalog: BacklogIntakeItemCatalog = {
  items: [...githubItems, ...sentryItems, ...pagerdutyItems, ...linearItems],
  iconSrcByIntakeId: {
    [LINEAR_ISSUES_INTAKE_ID]: linearIcon,
  },
};

export const severalIntakeFactoriesFixture: FactoriesFixture = {
  ...lineMetricsFactoriesFixture,
  intakesByFactoryId: {
    ...lineMetricsFactoriesFixture.intakesByFactoryId,
    [PRIMARY_FACTORY_ID]: [
      GITHUB_ISSUES_INTAKE,
      SENTRY_EXCEPTIONS_INTAKE,
      PAGERDUTY_INCIDENTS_INTAKE,
      LINEAR_ISSUES_INTAKE,
    ],
  },
  intakeItemCatalog: severalIntakeItemCatalog,
};

/** Two listeners at the head of Backlog: GitHub issues and Sentry exceptions. */
export const githubAndSentryIntakeFactoriesFixture: FactoriesFixture = {
  ...lineMetricsFactoriesFixture,
  intakesByFactoryId: {
    ...lineMetricsFactoriesFixture.intakesByFactoryId,
    [PRIMARY_FACTORY_ID]: [GITHUB_ISSUES_INTAKE, SENTRY_EXCEPTIONS_INTAKE],
  },
  intakeItemCatalog: { items: [...githubItems, ...sentryItems] },
};

export const noIntakeFactoriesFixture: FactoriesFixture = {
  ...lineMetricsFactoriesFixture,
  intakesByFactoryId: {
    ...lineMetricsFactoriesFixture.intakesByFactoryId,
    [PRIMARY_FACTORY_ID]: [],
  },
  intakeItemCatalog: { items: [] },
};
