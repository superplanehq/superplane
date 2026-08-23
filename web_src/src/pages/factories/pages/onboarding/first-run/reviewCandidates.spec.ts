import { describe, expect, it } from "vitest";

import {
  confidenceBandForScore,
  isReviewCandidateTab,
  reviewCandidateForWorkOrderId,
  REVIEW_CANDIDATE_WORK_ORDERS,
  REVIEW_CANDIDATES,
} from "./reviewCandidates";

describe("reviewCandidates", () => {
  it("maps a work order id to a review candidate", () => {
    const candidate = reviewCandidateForWorkOrderId("wo-review-pay-842");
    expect(candidate?.ticketKey).toBe("PAY-842");
    expect(candidate?.ticketBody).toContain("Webhook delivery");
    expect(candidate?.confidencePct).toBe(95);
    expect(candidate?.sections.map((section) => section.title)).toEqual([
      "Requirements understood",
      "Acceptance criteria",
      "Relevant code and tests",
      "Implementation plan",
    ]);
  });

  it("builds draft backlog work orders for each candidate", () => {
    expect(REVIEW_CANDIDATE_WORK_ORDERS).toHaveLength(REVIEW_CANDIDATES.length);
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.state).toBe("STATE_DRAFT");
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.key).toBe("PAY-842");
    expect(REVIEW_CANDIDATE_WORK_ORDERS[0]?.lineDispatches).toEqual([]);
  });

  it("bands confidence scores", () => {
    expect(confidenceBandForScore(95)).toBe("High");
    expect(confidenceBandForScore(81)).toBe("Medium");
    expect(confidenceBandForScore(68)).toBe("Low");
  });

  it("accepts only the three review tabs", () => {
    expect(isReviewCandidateTab("plan")).toBe(true);
    expect(isReviewCandidateTab("ticket")).toBe(true);
    expect(isReviewCandidateTab("analysis")).toBe(true);
    expect(isReviewCandidateTab("runs")).toBe(false);
  });
});
