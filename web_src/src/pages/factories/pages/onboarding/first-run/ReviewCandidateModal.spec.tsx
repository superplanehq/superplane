import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ReviewCandidateModal } from "./ReviewCandidateModal";
import { REVIEW_CANDIDATES } from "./reviewCandidates";

const candidate = REVIEW_CANDIDATES[0]!;

describe("ReviewCandidateModal", () => {
  it("shows the plan review for a scored ticket", () => {
    render(
      <MemoryRouter>
        <ReviewCandidateModal candidate={candidate} onClose={vi.fn()} />
      </MemoryRouter>,
    );

    expect(screen.getByTestId("review-candidate-modal")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Review candidate" })).toBeInTheDocument();
    expect(screen.getByText("PAY-842")).toBeInTheDocument();
    expect(screen.getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    expect(screen.getByText("95%")).toBeInTheDocument();
    expect(screen.getByText("High")).toBeInTheDocument();
    expect(screen.getByTestId("review-candidate-section-01")).toHaveTextContent("Requirements understood");
    expect(screen.getByTestId("review-candidate-section-04")).toHaveTextContent("Implementation plan");
    expect(screen.getByText("No blocking questions")).toBeInTheDocument();
  });

  it("closes from Back to results", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <ReviewCandidateModal candidate={candidate} onClose={onClose} />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Back to results" }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("approves the plan without starting implementation twice", async () => {
    const user = userEvent.setup();
    render(
      <MemoryRouter>
        <ReviewCandidateModal candidate={candidate} onClose={vi.fn()} />
      </MemoryRouter>,
    );

    await user.click(screen.getByRole("button", { name: "Approve plan and start" }));
    expect(screen.getByRole("button", { name: "Plan approved" })).toBeDisabled();
  });
});
