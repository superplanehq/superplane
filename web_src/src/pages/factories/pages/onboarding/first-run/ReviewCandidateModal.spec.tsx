import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { LINE_INTAKE_COPY } from "../../lineIntakeModel";
import { ReviewCandidateModal } from "./ReviewCandidateModal";
import { REVIEW_CANDIDATE_COPY, REVIEW_CANDIDATES } from "./reviewCandidates";

const candidate = REVIEW_CANDIDATES[0]!;

function renderModal(onClose = vi.fn()) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <MemoryRouter>
        <ThemeProvider>
          <TooltipProvider>
            <ReviewCandidateModal candidate={candidate} onClose={onClose} />
          </TooltipProvider>
        </ThemeProvider>
      </MemoryRouter>
    </QueryClientProvider>,
  );
}

describe("ReviewCandidateModal", () => {
  it("shows the plan first, then ticket and analysis run tabs", async () => {
    const user = userEvent.setup();
    renderModal();

    const dialog = screen.getByTestId("review-candidate-modal");
    expect(within(dialog).getByRole("heading", { name: REVIEW_CANDIDATE_COPY.kicker })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.planTab })).toHaveAttribute(
      "aria-selected",
      "true",
    );
    expect(within(dialog).getByText("PAY-842")).toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: "Add retry handling to webhook delivery" })).toBeInTheDocument();
    expect(within(dialog).getByTestId("review-candidate-section-04")).toHaveTextContent("Implementation plan");
    expect(within(dialog).queryByText(candidate.ticketBody)).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.ticketTab }));
    expect(within(dialog).getByText(candidate.ticketBody)).toBeInTheDocument();
    expect(within(dialog).getByText(/GitHub Issues/)).toBeInTheDocument();
    expect(within(dialog).getByText(/acme\/payments-service/)).toBeInTheDocument();
    expect(within(dialog).queryByTestId("review-candidate-section-04")).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.analysisTab }));
    expect(within(dialog).getByText(LINE_INTAKE_COPY.analysisCompleteHeadline)).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Run")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();
  });

  it("closes from Back to results", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderModal(onClose);

    await user.click(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.back }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("approves the plan without starting implementation twice", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.approve }));
    expect(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.approved })).toBeDisabled();
  });
});
