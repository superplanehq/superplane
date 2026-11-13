import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";
import { ReviewRequestDecisionSummary } from "./ReviewRequestDecisionSummary";

describe("Review request run outcomes", () => {
  for (const [nodeId, status, heading] of [
    ["review-unnecessary", "review_unnecessary", "Review unnecessary"],
    ["no-reviewer", "no_eligible_reviewer", "No eligible reviewer"],
    ["stale-report", "stale", "Stale report"],
    ["review-outcome", "already_covered", "Review already covered"],
  ]) {
    it(`shows ${status} from persisted canvas output`, () => {
      render(
        <ReviewRequestDecisionSummary
          run={{ id: "run-1" }}
          executions={[
            {
              runId: "other-run",
              nodeId,
              outputs: { default: [{ data: { reviewRequest: { status: "requested", rationale: "Other run" } } }] },
            },
            {
              runId: "run-1",
              nodeId,
              outputs: {
                default: [
                  {
                    data: {
                      reviewRequest: { status, rationale: "Saved decision", reviewers: [], fallbackUsed: false },
                    },
                  },
                ],
              },
            },
          ]}
        />,
      );
      expect(screen.getByRole("heading", { name: heading })).toBeVisible();
      expect(screen.queryByText("Other run")).not.toBeInTheDocument();
    });
  }
});
