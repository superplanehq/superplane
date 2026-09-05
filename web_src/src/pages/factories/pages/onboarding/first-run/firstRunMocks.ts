import type { FirstRunChrome, FirstRunScoredTicket } from "./firstRunTypes";

export const FIRST_RUN_STORY_EMAIL = "ada@superplane.dev";
export const FIRST_RUN_STORY_ORGANIZATION_ID = "org-storybook-acme";
export const FIRST_RUN_STORY_ORGANIZATION_NAME = "Acme";

/**
 * Chrome for an isolated screen story. Defaults to a user who can quit
 * onboarding; pass `{ onQuitOnboarding: undefined }` to show the sign-out-only
 * fallback instead.
 */
export function firstRunStoryChrome(stepIndex: number, overrides: Partial<FirstRunChrome> = {}): FirstRunChrome {
  return {
    displayName: "Ada",
    email: FIRST_RUN_STORY_EMAIL,
    organizationId: FIRST_RUN_STORY_ORGANIZATION_ID,
    organizationName: FIRST_RUN_STORY_ORGANIZATION_NAME,
    onQuitOnboarding: () => undefined,
    stepIndex,
    ...overrides,
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

export const FIRST_RUN_REPOSITORIES = [
  "acme/api",
  "acme/web",
  "acme/payments-service",
  "acme/billing",
  "acme/docs",
  "acme/infra",
];
