import { canvasAppIds } from "@/pages/app/__fixtures__/handlers";

/** Shared with the home fixture so routes stay in sync across HomePage → Factories navigation. */
export const FACTORIES_ORGANIZATION_ID = "3ee1aa47-3a60-4c1f-b645-0b9859ab91f8";

export const PRIMARY_FACTORY_ID = "factory-refunds";
export const EMPTY_FACTORY_ID = "factory-payments";
export const ACME_ONBOARDING_FACTORY_ID = "factory-acme-onboarding";

/** Workspace key for `PRIMARY_FACTORY_ID` — routes use this, not the raw id. */
export const PRIMARY_FACTORY_KEY = "RF";
/** Workspace key for `EMPTY_FACTORY_ID` — routes use this, not the raw id. */
export const EMPTY_FACTORY_KEY = "PF";
/** Workspace key for `ACME_ONBOARDING_FACTORY_ID` — routes use this, not the raw id. */
export const ACME_ONBOARDING_FACTORY_KEY = "AO";

export const STORYBOOK_ME_USER_ID = "storybook-user";
export const STORYBOOK_ME_USER_NAME = "Leonardo DiCaprio";
export const STORYBOOK_ME_USER_EMAIL = "john.doe@superplane.dev";
export const STORYBOOK_ME_USER_AVATAR_URL = "/storybook/leonardo-dicaprio.jpg";

// Relative timestamps so `formatTimeAgo` stays stable across story loads.
const NOW_MS = Date.now();
const MINUTE_MS = 60_000;
const HOUR_MS = 60 * MINUTE_MS;
const DAY_MS = 24 * HOUR_MS;
const relativeIso = (offsetMs: number) => new Date(NOW_MS - offsetMs).toISOString();

export const HOUR_AGO = relativeIso(HOUR_MS);
export const TWO_HOURS_AGO = relativeIso(2 * HOUR_MS);
export const YESTERDAY = relativeIso(DAY_MS);
export const LAST_WEEK = relativeIso(7 * DAY_MS);

export function minutesAgo(minutes: number) {
  return relativeIso(minutes * MINUTE_MS);
}

/**
 * Canvas run ids on Line phase cards. Must be UUIDs: AppPage strips any
 * `?run=` value that fails `isValidRunId`, which drops factory run autolayout.
 *
 * `LINE_RUN_IMPLEMENT_FAILED_ID` matches the captured Software Factory
 * published run so Storybook reuses that run's executions and root event.
 */
export const LINE_RUN_IMPLEMENT_ID = "8f3a1c2e-4b5d-46f0-a789-0b1c2d3e4f50";
export const LINE_RUN_IMPLEMENT_FAILED_ID = canvasAppIds.publishedRunId ?? "fef4cee8-fdd7-47af-b5da-e739664cd31d";
export const LINE_RUN_IMPLEMENT_PASSED_ID = "9a4b2d3f-5c6e-47f0-b890-1c2d3e4f5061";
export const LINE_RUN_VERIFY_PASSED_ID = "0b5c3e4a-6d7f-4081-8901-2d3e4f506172";
export const LINE_RUN_IMPLEMENT_FAILED_ROOT_EVENT_ID =
  canvasAppIds.rootEventId ?? "755a4430-2481-43f6-94cb-089c331a5d2f";

export const REVIEWER_USER = {
  id: "user-reviewer-alex",
  name: "Alex Reviewer",
  email: "alex@superplane.dev",
} as const;

export const OPERATOR_USER = {
  id: "user-operator-jamie",
  name: "Jamie Operator",
  email: "jamie@superplane.dev",
} as const;

export const ARNOLD_USER = {
  id: "user-arnold",
  name: "Arnold Schwarzenegger",
  email: "arnold@superplane.dev",
  avatarUrl: "/storybook/arnold-schwarzenegger.jpg",
} as const;

export const ORGANIZATION_USERS = [
  {
    id: STORYBOOK_ME_USER_ID,
    name: STORYBOOK_ME_USER_NAME,
    email: STORYBOOK_ME_USER_EMAIL,
    avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
  },
  ARNOLD_USER,
  REVIEWER_USER,
  OPERATOR_USER,
];

export const REFUND_LINE_PLAN_ID = "line-plan-and-implement";
export const REFUND_LINE_HOTFIX_ID = "line-hotfix";
export const REFUND_LINE_ONBOARDING_ID = "line-onboarding";
export const REFUND_LINE_FEATURE_ID = "line-feature-delivery";
export const ACME_ONBOARDING_LINE_ID = "line-acme-onboarding";
export const GITHUB_ISSUES_INTAKE_APP_ID = "app-github-issues-intake";
