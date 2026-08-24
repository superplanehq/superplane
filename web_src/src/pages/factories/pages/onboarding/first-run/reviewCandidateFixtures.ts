import type { FactoriesWorkOrder } from "@/api-client";

import {
  HOUR_AGO,
  LAST_WEEK,
  OPERATOR_USER,
  REVIEWER_USER,
  STORYBOOK_ME_USER_AVATAR_URL,
  STORYBOOK_ME_USER_ID,
  STORYBOOK_ME_USER_NAME,
  TWO_HOURS_AGO,
  YESTERDAY,
} from "../../../__fixtures__/factoryPageResponses";
import {
  REVIEW_CANDIDATE_COPY,
  githubIssueUrl,
  implementationPlanMarkdown,
  type ReviewCandidate,
  type ReviewCandidateSection,
  type ReviewIssue,
  type ReviewIssuePerson,
} from "./reviewCandidateModel";

function planMarkdownFromSections(sections: ReviewCandidateSection[]): string {
  const requirements = sectionByNumber(sections, "01");
  const criteria = sectionByNumber(sections, "02");
  const files = sectionByNumber(sections, "03");
  const plan = sectionByNumber(sections, "04");

  return implementationPlanMarkdown({
    goal: requirements?.intro ?? "",
    files: files?.items ?? [],
    steps: plan?.items ?? [],
    verify: criteria?.items ?? [],
  });
}

function sectionByNumber(sections: ReviewCandidateSection[], number: string) {
  return sections.find((section) => section.number === number);
}

function withPlanMarkdown(candidate: Omit<ReviewCandidate, "planMarkdown">): ReviewCandidate {
  return { ...candidate, planMarkdown: planMarkdownFromSections(candidate.sections) };
}

const ISSUE_AUTHOR: ReviewIssuePerson = {
  name: STORYBOOK_ME_USER_NAME,
  login: "ldicaprio",
  avatarUrl: STORYBOOK_ME_USER_AVATAR_URL,
};

const ISSUE_ASSIGNEE_ALEX: ReviewIssuePerson = {
  name: REVIEWER_USER.name,
  login: "alex",
};

const ISSUE_ASSIGNEE_JAMIE: ReviewIssuePerson = {
  name: OPERATOR_USER.name,
  login: "jamie",
};

function reviewIssue(ticketKey: string, issue: Omit<ReviewIssue, "url">): ReviewIssue {
  return { ...issue, url: githubIssueUrl(REVIEW_CANDIDATE_COPY.ticketRepository, ticketKey) };
}

function issueMarkdown(summary: string, extra: string): string {
  return `${summary}\n\n${extra}`;
}

export const REVIEW_CANDIDATES: ReviewCandidate[] = [
  withPlanMarkdown({
    workOrderId: "wo-review-pay-842",
    ticketKey: "PAY-842",
    title: "Add retry handling to webhook delivery",
    ticketBody:
      "Webhook delivery stops after a transient provider error. Add bounded retries that stay idempotent and keep the delivery audit trail.",
    issue: reviewIssue("PAY-842", {
      createdAt: LAST_WEEK,
      updatedAt: TWO_HOURS_AGO,
      author: ISSUE_AUTHOR,
      assignees: [ISSUE_ASSIGNEE_ALEX],
      labels: [{ name: "reliability" }, { name: "webhooks" }, { name: "ready" }],
      bodyMarkdown: issueMarkdown(
        "Webhook delivery stops after a transient provider error. Add bounded retries that stay idempotent and keep the delivery audit trail.",
        `## Acceptance criteria

- Retry 5xx, 408, and 429 responses up to four attempts.
- Reuse the delivery ID so retries never create a duplicate event.
- Record the last response code and attempt count on final failure.

## Notes

Do not retry permanent 4xx responses, except rate limits and timeouts.`,
      ),
    }),
    confidencePct: 95,
    confidenceBand: "High",
    reasons: [
      "Acceptance criteria name the retryable status codes and the attempt limit.",
      "The dispatcher and the shared retry utility are mapped, with neighboring tests.",
      "No product decision is open. The change stays in one delivery path.",
    ],
    summary:
      "Analysis complete. Requirements, acceptance criteria, relevant code, tests, and the implementation plan have been reviewed.",
    readyNote:
      "No implementation has started. Review the completed analysis and approve the plan when you are ready for SuperPlane to begin.",
    sections: [
      {
        number: "01",
        title: "Requirements understood",
        intro:
          "The retry behavior is already established in two adjacent services. SuperPlane found the relevant module, matching test patterns, and no unresolved product decisions.",
        items: [
          "Retry transient webhook failures with bounded exponential backoff.",
          "Keep every retry idempotent and preserve the existing delivery audit trail.",
          "Do not retry permanent 4xx responses, except rate limits and timeouts.",
        ],
      },
      {
        number: "02",
        title: "Acceptance criteria",
        intro: "Mapped from the ticket and repository conventions.",
        items: [
          "5xx, 408, and 429 responses are retried up to four attempts.",
          "Retries use the existing delivery ID and never produce duplicate events.",
          "The final failure records the last response code and attempt count.",
          "Existing successful-delivery behavior and latency remain unchanged.",
        ],
      },
      {
        number: "03",
        title: "Relevant code and tests",
        intro: "Evidence used to assess feasibility and confidence.",
        items: [
          "src/webhooks/dispatcher.ts — Delivery loop and provider error handling",
          "src/shared/retry-policy.ts — Existing backoff pattern used by two services",
          "tests/webhooks/dispatcher.test.ts — 14 neighboring tests cover delivery outcomes",
        ],
      },
      {
        number: "04",
        title: "Implementation plan",
        intro: "The planned path to a review-ready pull request.",
        items: [
          "Add a webhook-specific retry policy using the shared backoff utility.",
          "Classify provider responses as retryable or terminal in the dispatcher.",
          "Reuse the delivery ID across attempts and record attempt metadata.",
          "Add tests for success-after-retry, exhaustion, 429, and permanent 4xx responses.",
        ],
      },
    ],
    noBlockingQuestions: "The analysis found enough context to begin implementation with the plan above.",
  }),
  withPlanMarkdown({
    workOrderId: "wo-review-pay-843",
    ticketKey: "PAY-843",
    title: "Handle duplicate refunds on retry",
    ticketBody:
      "A retry of a successful refund can post a second ledger entry. Treat the same provider key as the original refund.",
    issue: reviewIssue("PAY-843", {
      createdAt: LAST_WEEK,
      updatedAt: YESTERDAY,
      author: ISSUE_AUTHOR,
      assignees: [ISSUE_ASSIGNEE_JAMIE],
      labels: [{ name: "refunds" }, { name: "idempotency" }, { name: "ready" }],
      bodyMarkdown: issueMarkdown(
        "A retry of a successful refund can post a second ledger entry. Treat the same provider key as the original refund.",
        `## Acceptance criteria

- A retry with the same provider key returns the original refund result.
- The ledger gains at most one posted refund for that key.
- A failed first attempt can retry after success without a duplicate post.

## Notes

Reuse the charge idempotency store. Keep the existing refund audit trail.`,
      ),
    }),
    confidencePct: 94,
    confidenceBand: "High",
    reasons: [
      "Acceptance criteria require one ledger post per provider key.",
      "The refund post path and the charge idempotency store are mapped.",
      "Neighboring tests already cover success and conflict.",
    ],
    summary:
      "Analysis complete. Requirements, acceptance criteria, relevant code, tests, and the implementation plan have been reviewed.",
    readyNote:
      "No implementation has started. Review the completed analysis and approve the plan when you are ready for SuperPlane to begin.",
    sections: [
      {
        number: "01",
        title: "Requirements understood",
        intro:
          "The ledger already records refund attempts. SuperPlane found the retry path and the duplicate-guard tests.",
        items: [
          "Treat a repeated refund request with the same provider key as the original refund.",
          "Do not create a second ledger entry when the first attempt succeeded.",
          "Keep the existing refund audit trail.",
        ],
      },
      {
        number: "02",
        title: "Acceptance criteria",
        intro: "Mapped from the ticket and repository conventions.",
        items: [
          "A retry with the same idempotency key returns the original refund result.",
          "The ledger gains at most one posted refund for that key.",
          "Failed first attempts can retry without a duplicate post after success.",
        ],
      },
      {
        number: "03",
        title: "Relevant code and tests",
        intro: "Evidence used to assess feasibility and confidence.",
        items: [
          "src/refunds/posting.ts — Refund post and provider key lookup",
          "src/refunds/idempotency.ts — Existing key store used by charges",
          "tests/refunds/posting.test.ts — Neighboring tests cover success and conflict",
        ],
      },
      {
        number: "04",
        title: "Implementation plan",
        intro: "The planned path to a review-ready pull request.",
        items: [
          "Reuse the charge idempotency store for refund retries.",
          "Return the stored result when the key already posted.",
          "Add tests for success-after-retry and duplicate-after-success.",
        ],
      },
    ],
    noBlockingQuestions: "The analysis found enough context to begin implementation with the plan above.",
  }),
  withPlanMarkdown({
    workOrderId: "wo-review-pay-844",
    ticketKey: "PAY-844",
    title: "Return 409 when the invoice is already paid",
    ticketBody:
      "A second pay on a paid invoice returns HTTP 500. Map the already-paid state to HTTP 409 and leave the invoice unchanged.",
    issue: reviewIssue("PAY-844", {
      createdAt: YESTERDAY,
      updatedAt: TWO_HOURS_AGO,
      author: ISSUE_AUTHOR,
      assignees: [ISSUE_ASSIGNEE_ALEX],
      labels: [{ name: "bug" }, { name: "invoices" }, { name: "api" }],
      bodyMarkdown: issueMarkdown(
        "A second pay on a paid invoice returns HTTP 500. Map the already-paid state to HTTP 409 and leave the invoice unchanged.",
        `## Acceptance criteria

- A second pay on a paid invoice returns HTTP 409.
- The response names the invoice as already paid.
- A first successful pay still returns HTTP 200.

## Notes

The domain already rejects a second capture. Only the HTTP mapping is wrong.`,
      ),
    }),
    confidencePct: 88,
    confidenceBand: "High",
    reasons: [
      "Acceptance criteria name HTTP 409 for a second pay on a paid invoice.",
      "The pay handler and the existing conflict error mapping are known.",
      "The domain already rejects a second capture. Only the HTTP layer is wrong.",
    ],
    summary:
      "Analysis complete. Requirements, acceptance criteria, relevant code, tests, and the implementation plan have been reviewed.",
    readyNote:
      "No implementation has started. Review the completed analysis and approve the plan when you are ready for SuperPlane to begin.",
    sections: [
      {
        number: "01",
        title: "Requirements understood",
        intro: "Invoice pay already rejects a second capture in the model. The HTTP layer still returns 500.",
        items: [
          "Map an already-paid invoice to HTTP 409.",
          "Keep the invoice state unchanged on the second pay request.",
          "Do not change the successful first-pay response.",
        ],
      },
      {
        number: "02",
        title: "Acceptance criteria",
        intro: "Mapped from the ticket and repository conventions.",
        items: [
          "A second pay on a paid invoice returns 409.",
          "The response names the invoice as already paid.",
          "A first successful pay still returns 200.",
        ],
      },
      {
        number: "03",
        title: "Relevant code and tests",
        intro: "Evidence used to assess feasibility and confidence.",
        items: [
          "src/invoices/pay.ts — Pay handler and state transition",
          "src/http/errors.ts — Conflict mapping used by subscriptions",
          "tests/invoices/pay.test.ts — Existing paid-state coverage",
        ],
      },
      {
        number: "04",
        title: "Implementation plan",
        intro: "The planned path to a review-ready pull request.",
        items: [
          "Translate the already-paid domain error to 409 in the pay handler.",
          "Reuse the conflict error body used by subscriptions.",
          "Add a test for a second pay on a paid invoice.",
        ],
      },
    ],
    noBlockingQuestions: "The analysis found enough context to begin implementation with the plan above.",
  }),
  withPlanMarkdown({
    workOrderId: "wo-review-pay-845",
    ticketKey: "PAY-845",
    title: "Show a clearer empty state on the billing page",
    ticketBody:
      "The billing page empty branch does not tell the user what to do next. Show a short title and one create-invoice action.",
    issue: reviewIssue("PAY-845", {
      createdAt: YESTERDAY,
      updatedAt: HOUR_AGO,
      author: ISSUE_AUTHOR,
      assignees: [],
      labels: [{ name: "ui" }, { name: "billing" }],
      bodyMarkdown: issueMarkdown(
        "The billing page empty branch does not tell the user what to do next. Show a short title and one create-invoice action.",
        `## Acceptance criteria

- A user with no invoices sees the empty state and a create-invoice action.
- A user with invoices still sees the invoice list.
- The empty state uses the existing page header.

## Notes

This change is copy and layout only. Do not add a new billing API.`,
      ),
    }),
    confidencePct: 81,
    confidenceBand: "Medium",
    reasons: [
      "Acceptance criteria name the empty-state title and the create-invoice action.",
      "The billing page already has an empty branch and a shared empty-state component.",
      "The change is copy and layout only. No new billing API.",
    ],
    summary:
      "Analysis complete. Requirements, acceptance criteria, relevant code, tests, and the implementation plan have been reviewed.",
    readyNote:
      "No implementation has started. Review the completed analysis and approve the plan when you are ready for SuperPlane to begin.",
    sections: [
      {
        number: "01",
        title: "Requirements understood",
        intro: "The billing page already has an empty branch. The copy does not tell the user what to do next.",
        items: [
          "Show a short empty-state title and one next action.",
          "Keep the page layout when invoices exist.",
          "Do not add a new billing API.",
        ],
      },
      {
        number: "02",
        title: "Acceptance criteria",
        intro: "Mapped from the ticket and repository conventions.",
        items: [
          "A user with no invoices sees the empty state and a create-invoice action.",
          "A user with invoices still sees the invoice list.",
          "The empty state uses the existing page header.",
        ],
      },
      {
        number: "03",
        title: "Relevant code and tests",
        intro: "Evidence used to assess feasibility and confidence.",
        items: [
          "web/src/pages/Billing.tsx — Invoice list and empty branch",
          "web/src/ui/EmptyState.tsx — Shared empty-state pattern",
          "web/src/pages/Billing.spec.tsx — List and empty coverage",
        ],
      },
      {
        number: "04",
        title: "Implementation plan",
        intro: "The planned path to a review-ready pull request.",
        items: [
          "Replace the empty paragraph with the shared empty-state component.",
          "Add a create-invoice action that opens the existing dialog.",
          "Add a test for the empty-state title and action.",
        ],
      },
    ],
    noBlockingQuestions: "The analysis found enough context to begin implementation with the plan above.",
  }),
  withPlanMarkdown({
    workOrderId: "wo-review-pay-846",
    ticketKey: "PAY-846",
    title: "Upgrade the Node 20 base image",
    ticketBody:
      "The app and worker images still pin Node 18. Move both images to Node 20 and keep the same entrypoint.",
    issue: reviewIssue("PAY-846", {
      createdAt: YESTERDAY,
      updatedAt: HOUR_AGO,
      author: ISSUE_AUTHOR,
      assignees: [ISSUE_ASSIGNEE_ALEX, ISSUE_ASSIGNEE_JAMIE],
      labels: [{ name: "infrastructure" }, { name: "node" }, { name: "ci" }],
      bodyMarkdown: issueMarkdown(
        "The app and worker images still pin Node 18. Move both images to Node 20 and keep the same entrypoint.",
        `## Acceptance criteria

- The app and worker images build on Node 20.
- CI builds both images and runs the existing image smoke tests.
- Runtime environment variables stay the same.

## Notes

Keep the same entrypoint and process user. Do not change application dependencies in this ticket.`,
      ),
    }),
    confidencePct: 76,
    confidenceBand: "Medium",
    reasons: [
      "Acceptance criteria name the Node 20 images and the existing smoke tests.",
      "The change is limited to two Dockerfiles and the image CI job.",
      "CI already runs a Node 20 job, so the target image is known.",
    ],
    summary:
      "Analysis complete. Requirements, acceptance criteria, relevant code, tests, and the implementation plan have been reviewed.",
    readyNote:
      "No implementation has started. Review the completed analysis and approve the plan when you are ready for SuperPlane to begin.",
    sections: [
      {
        number: "01",
        title: "Requirements understood",
        intro: "Two Dockerfiles pin Node 18. CI already has a Node 20 job for the web package.",
        items: [
          "Move the application and worker images to Node 20.",
          "Keep the same entrypoint and process user.",
          "Do not change application dependencies in this ticket.",
        ],
      },
      {
        number: "02",
        title: "Acceptance criteria",
        intro: "Mapped from the ticket and repository conventions.",
        items: [
          "The app and worker images build on Node 20.",
          "CI builds both images and runs the existing image smoke tests.",
          "Runtime environment variables stay the same.",
        ],
      },
      {
        number: "03",
        title: "Relevant code and tests",
        intro: "Evidence used to assess feasibility and confidence.",
        items: [
          "docker/app.Dockerfile — Application base image",
          "docker/worker.Dockerfile — Worker base image",
          ".github/workflows/images.yml — Image build and smoke tests",
        ],
      },
      {
        number: "04",
        title: "Implementation plan",
        intro: "The planned path to a review-ready pull request.",
        items: [
          "Update both Dockerfiles to the Node 20 base image.",
          "Align the CI image job with the new tag.",
          "Run the existing image smoke tests.",
        ],
      },
    ],
    noBlockingQuestions: "The analysis found enough context to begin implementation with the plan above.",
  }),
];

const REVIEW_CANDIDATES_BY_ID = new Map(REVIEW_CANDIDATES.map((candidate) => [candidate.workOrderId, candidate]));

export function reviewCandidateForWorkOrderId(id: string | undefined): ReviewCandidate | undefined {
  if (!id) {
    return undefined;
  }
  return REVIEW_CANDIDATES_BY_ID.get(id);
}

export const REVIEW_CANDIDATE_WORK_ORDERS: FactoriesWorkOrder[] = REVIEW_CANDIDATES.map((candidate, index) => ({
  id: candidate.workOrderId,
  number: String(842 + index),
  key: candidate.ticketKey,
  title: candidate.title,
  description: candidate.summary,
  state: "STATE_DRAFT",
  result: "RESULT_UNSPECIFIED",
  createdAt: HOUR_AGO,
  updatedAt: HOUR_AGO,
  createdBy: { user: { id: STORYBOOK_ME_USER_ID, name: STORYBOOK_ME_USER_NAME } },
  assignees: [],
  lineDispatches: [],
}));
