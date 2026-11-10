import type { FactoriesFactoryPullRequest } from "@/api-client";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it } from "bun:test";
import { MemoryRouter } from "react-router";

import { FactoriesLayoutContext } from "../layout/factoriesLayoutContext";
import { HumanReviewState } from "./HumanReviewState";

function renderReview(humanReview: NonNullable<FactoriesFactoryPullRequest["humanReview"]>, currentCommit?: string) {
  return render(
    <MemoryRouter>
      <FactoriesLayoutContext.Provider
        value={{
          organizationId: "org",
          factoryId: "factory",
          factoryKey: "SP",
          routeSegment: "sp",
          factory: null,
          factories: [],
          openCreateWorkOrder: () => {},
        }}
      >
        <HumanReviewState
          pullRequest={{
            id: "pr",
            humanReview,
            currentRevision: currentCommit ? { sha: currentCommit } : undefined,
            activities: [{ run: { id: "assessment", canvasId: "merge-confidence" } }],
          }}
        />
      </FactoriesLayoutContext.Provider>
    </MemoryRouter>,
  );
}

describe("HumanReviewState", () => {
  it("keeps an undecided review blocked even when the review-required field defaults to false", () => {
    renderReview({ status: "waiting_assessment", reviewRequired: false, blockingReason: "Waiting for an assessment." });

    expect(screen.getByRole("heading")).toHaveTextContent("Waiting for merge confidence");
    expect(screen.queryByText(/Not required/)).not.toBeInTheDocument();
  });

  it("keeps reviewers visible and reveals decision evidence when expanded", async () => {
    const user = userEvent.setup();
    renderReview({
      status: "decided",
      reviewRequired: true,
      rationale: "A security finding requires review.",
      selectedReviewers: [{ id: "7", login: "reviewer", reason: "Owns the affected files." }],
      blockingReason: "Waiting for an approval.",
      sourceAssessmentRunId: "assessment",
      checkUrl: "https://github.com/acme/app/runs/7",
    });

    expect(screen.getByRole("heading")).toHaveTextContent("Blocked");
    expect(screen.getByRole("heading")).toHaveTextContent("@reviewer");
    expect(screen.queryByText("A security finding requires review.")).not.toBeInTheDocument();
    expect(screen.queryByText("Owns the affected files.", { exact: false })).not.toBeInTheDocument();

    const toggle = screen.getByRole("button", { name: /Request review/ });
    expect(toggle).toHaveAttribute("aria-expanded", "false");
    await user.click(toggle);

    expect(toggle).toHaveAttribute("aria-expanded", "true");
    expect(screen.getByText("A security finding requires review.")).toBeInTheDocument();
    expect(screen.getByRole("status")).toHaveTextContent("Waiting for an approval");
    expect(screen.getByRole("listitem")).toHaveTextContent("@reviewer");
    expect(screen.getByRole("listitem")).toHaveTextContent("Owns the affected files.");
    expect(screen.getByRole("link", { name: "View assessment" })).toHaveAttribute(
      "href",
      expect.stringContaining("assessment"),
    );
    expect(screen.getByRole("link", { name: "View GitHub check" })).toHaveAttribute(
      "href",
      "https://github.com/acme/app/runs/7",
    );

    await user.click(toggle);
    expect(screen.queryByRole("link", { name: "View assessment" })).not.toBeInTheDocument();
  });

  it("does not show a previous revision approval as approval for a new commit", () => {
    renderReview(
      { status: "decided", reviewRequired: true, approvalIds: ["11"], assessedCommit: "previous" },
      "current",
    );

    expect(screen.getByRole("status")).toHaveTextContent("Waiting for merge confidence");
    expect(screen.queryByText("Approved", { exact: false })).not.toBeInTheDocument();
  });

  it("keeps a change request blocked after a selected reviewer approves", async () => {
    const user = userEvent.setup();
    renderReview({
      status: "decided",
      reviewRequired: true,
      approvalIds: ["11"],
      blockingReason: "A reviewer requested changes.",
    });

    expect(screen.getByRole("heading")).toHaveTextContent("Blocked");
    await user.click(screen.getByRole("button", { name: /Request review/ }));
    expect(screen.getByRole("status")).toHaveTextContent("A reviewer requested changes");
  });

  it("keeps publication errors visible without expanding details", () => {
    renderReview({ status: "decided", reviewRequired: true, publicationError: "GitHub is unavailable" });

    expect(screen.getByRole("alert")).toHaveTextContent("GitHub");
  });
});
