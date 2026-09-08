import type { FirstRunChrome, FirstRunScoredTicket } from "./firstRunTypes";

export const FIRST_RUN_STORY_EMAIL = "ada@superplane.dev";

/** Isolated stories. The app shows Back on connect, repository, and tickets. */
export function firstRunStoryChrome(stepIndex: number): FirstRunChrome {
  const showsBack = stepIndex === 1 || stepIndex === 2 || stepIndex === 3;
  return {
    displayName: "Ada",
    email: FIRST_RUN_STORY_EMAIL,
    stepIndex,
    onBack: showsBack ? () => undefined : undefined,
  };
}

export const FIRST_RUN_SCORED_TICKETS: FirstRunScoredTicket[] = [
  {
    id: "ticket-1",
    title: "Handle duplicate refunds on retry",
    source: "acme/payments-service",
    confidenceScore: 5,
  },
  {
    id: "ticket-2",
    title: "Return 409 when the invoice is already paid",
    source: "acme/api",
    confidenceScore: 4,
  },
  {
    id: "ticket-3",
    title: "Show a clearer empty state on the billing page",
    source: "acme/web",
    confidenceScore: 3,
  },
  {
    id: "ticket-4",
    title: "Upgrade the Node 20 base image",
    source: "acme/infra",
    confidenceScore: 3,
  },
  {
    id: "ticket-5",
    title: "Add a flake retry to the checkout e2e suite",
    source: "acme/web",
    confidenceScore: 2,
  },
];

export const CLOUD_GITHUB_APP_SLUG = "superplane";
export const CLOUD_GITHUB_STATE = "csrf";
export const CLOUD_GITHUB_LOGIN = "ada";

export const CLOUD_GITHUB_ACME = {
  id: "11",
  accountLogin: "acme",
  accountType: "Organization",
} as const;

export const CLOUD_GITHUB_OCTO = {
  id: "22",
  accountLogin: "octo",
  accountType: "User",
} as const;

/** Requested organization that is not ready to use yet. */
export const CLOUD_GITHUB_GLOBEX = "globex";

export const FIRST_RUN_REPOSITORIES = [
  "acme/api",
  "acme/web",
  "acme/payments-service",
  "acme/billing",
  "acme/docs",
  "acme/infra",
];
