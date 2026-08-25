import type { FirstRunChrome, FirstRunScoredTicket } from "./firstRunTypes";

export const FIRST_RUN_STORY_EMAIL = "ada@superplane.dev";

export function firstRunStoryChrome(stepIndex: number): FirstRunChrome {
  return {
    displayName: "Ada",
    email: FIRST_RUN_STORY_EMAIL,
    stepIndex,
  };
}

export const FIRST_RUN_SCORED_TICKETS: FirstRunScoredTicket[] = [
  {
    id: "ticket-1",
    title: "Handle duplicate refunds on retry",
    source: "acme/payments-service",
    confidencePct: 94,
  },
  {
    id: "ticket-2",
    title: "Return 409 when the invoice is already paid",
    source: "acme/api",
    confidencePct: 88,
  },
  {
    id: "ticket-3",
    title: "Show a clearer empty state on the billing page",
    source: "acme/web",
    confidencePct: 81,
  },
  {
    id: "ticket-4",
    title: "Upgrade the Node 20 base image",
    source: "acme/infra",
    confidencePct: 76,
  },
  {
    id: "ticket-5",
    title: "Add a flake retry to the checkout e2e suite",
    source: "acme/web",
    confidencePct: 68,
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
