import { describe, expect, it } from "vitest";

import {
  confidenceBandClassName,
  confidenceBandForScore,
  implementationPlanMarkdown,
  githubIssueUrl,
  reviewCandidateForWorkOrderId,
  REVIEW_CANDIDATE_COPY,
  BOARD_REVIEW_CANDIDATES,
  REVIEW_CANDIDATE_WORK_ORDERS,
  REVIEW_CANDIDATES,
} from "./reviewCandidates";

describe("reviewCandidates", () => {
  it("maps a work order id to a review candidate", () => {
    const candidate = reviewCandidateForWorkOrderId("wo-review-pay-842");
    expect(candidate?.ticketKey).toBe("PAY-842");
    expect(candidate?.ticketBody).toContain("Webhook delivery");
    expect(candidate?.reasons).toHaveLength(3);
    expect(candidate?.confidenceScore).toBe(5);
    expect(candidate?.sections.map((section) => section.title)).toEqual([
      "Requirements understood",
      "Acceptance criteria",
      "Relevant code and tests",
      "Implementation plan",
    ]);
    expect(candidate?.planMarkdown).toContain("## Goal");
    expect(candidate?.planMarkdown).toContain("Add a webhook-specific retry policy using the shared backoff utility.");
    expect(candidate?.issue.url).toBe(githubIssueUrl(REVIEW_CANDIDATE_COPY.ticketRepository, "PAY-842"));
    expect(candidate?.issue.author.login).toBe("ldicaprio");
    expect(candidate?.issue.labels.map((label) => label.name)).toEqual(["reliability", "webhooks", "ready"]);
    expect(candidate?.issue.assignees.map((person) => person.login)).toEqual(["alex"]);
    expect(candidate?.issue.bodyMarkdown).toContain(candidate?.ticketBody);
  });

  it("builds a GitHub issue URL from the repository and ticket key", () => {
    expect(githubIssueUrl("acme/payments-service", "PAY-846")).toBe(
      "https://github.com/acme/payments-service/issues/846",
    );
  });

  it("builds a markdown implementation plan from analysis parts", () => {
    expect(
      implementationPlanMarkdown({
        goal: "Move both images to Node 20.",
        files: ["docker/app.Dockerfile — Application base image"],
        steps: ["Update both Dockerfiles to the Node 20 base image."],
        verify: ["CI builds both images and runs the existing image smoke tests."],
      }),
    ).toBe(`## Goal

Move both images to Node 20.

## Files to change

- docker/app.Dockerfile — Application base image

## Steps

1. Update both Dockerfiles to the Node 20 base image.

## Verify

- CI builds both images and runs the existing image smoke tests.`);
  });

  it("puts mixed confidence scores on the three board review tickets", () => {
    expect(BOARD_REVIEW_CANDIDATES.map((candidate) => [candidate.ticketKey, candidate.confidenceScore])).toEqual([
      ["PAY-842", 5],
      ["PAY-844", 4],
      ["PAY-845", 3],
    ]);
  });

  it("builds draft backlog work orders for each candidate", () => {
    expect(REVIEW_CANDIDATE_WORK_ORDERS).toHaveLength(REVIEW_CANDIDATES.length);
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.state).toBe("STATE_DRAFT");
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.key).toBe("PAY-842");
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.lineDispatches).toEqual([]);
  });

  it("bands confidence scores", () => {
    expect(confidenceBandForScore(5)).toBe("High");
    expect(confidenceBandForScore(4)).toBe("High");
    expect(confidenceBandForScore(3)).toBe("Medium");
    expect(confidenceBandForScore(2)).toBe("Low");
    expect(confidenceBandClassName("High")).toMatch(/emerald/);
    expect(confidenceBandClassName("Medium")).toMatch(/orange/);
    expect(confidenceBandClassName("Low")).toMatch(/red/);
  });
});
