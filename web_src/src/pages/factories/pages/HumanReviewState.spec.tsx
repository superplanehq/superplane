import type { FactoriesFactoryPullRequest } from "@/api-client";
import { render, screen } from "@testing-library/react";
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
    expect(screen.getByRole("status")).toHaveTextContent("Waiting for an assessment");
    expect(screen.queryByText("Approval is not required", { exact: false })).not.toBeInTheDocument();
  });

  it("shows decision evidence, selected reviewers, and assessment and check links", () => {
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
    expect(screen.getByText("A security finding requires review.")).toBeInTheDocument();
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
  });

  it("does not show a previous revision approval as approval for a new commit", () => {
    renderReview(
      { status: "decided", reviewRequired: true, approvalIds: ["11"], assessedCommit: "previous" },
      "current",
    );

    expect(screen.getByRole("heading")).toHaveTextContent("Waiting for merge confidence");
    expect(screen.queryByText("Approved", { exact: false })).not.toBeInTheDocument();
  });

  it("keeps a change request visible after a selected reviewer approves", () => {
    renderReview({
      status: "decided",
      reviewRequired: true,
      approvalIds: ["11"],
      blockingReason: "A reviewer requested changes.",
    });

    expect(screen.getByRole("heading")).toHaveTextContent("Blocked");
    expect(screen.getByRole("status")).toHaveTextContent("A reviewer requested changes");
  });
});
