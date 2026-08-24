import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter } from "react-router";
import { describe, expect, it, vi } from "vitest";

import { ThemeProvider } from "@/contexts/ThemeProvider";
import { TooltipProvider } from "@/ui/tooltip";

import { formatRelative } from "@/lib/datetime";

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
  it("shows the ticket title, score, reasons, then the plan", async () => {
    const user = userEvent.setup();
    renderModal();

    const dialog = screen.getByTestId("review-candidate-modal");
    expect(within(dialog).queryByRole("heading", { name: "Review candidate" })).not.toBeInTheDocument();
    expect(within(dialog).getByRole("heading", { name: candidate.title })).toBeInTheDocument();
    expect(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.planTab })).toHaveAttribute(
      "aria-selected",
      "true",
    );

    const score = within(dialog).getByTestId("review-candidate-score");
    expect(score).toHaveTextContent("5 / 5");
    expect(score).toHaveTextContent("High");
    expect(within(score).getByText("High").className).toMatch(/emerald/);
    expect(within(dialog).queryByText(candidate.summary)).not.toBeInTheDocument();

    const reasons = within(dialog).getByTestId("review-candidate-reasons");
    expect(within(reasons).getAllByRole("listitem")).toHaveLength(3);
    expect(reasons).toHaveTextContent(candidate.reasons[0]!);
    expect(within(dialog).getByRole("heading", { name: REVIEW_CANDIDATE_COPY.planHeading })).toBeInTheDocument();
    expect(within(dialog).getByTestId("review-candidate-plan-divider")).toBeInTheDocument();
    expect(within(dialog).getByTestId("review-candidate-plan-file")).toHaveTextContent(REVIEW_CANDIDATE_COPY.planFile);
    const plan = within(dialog).getByTestId("review-candidate-plan");
    expect(plan).toHaveTextContent("Goal");
    expect(plan).toHaveTextContent("Add a webhook-specific retry policy using the shared backoff utility.");
    expect(within(dialog).getByRole("button", { name: REVIEW_CANDIDATE_COPY.editPlan })).toBeInTheDocument();
    expect(within(dialog).queryByText(candidate.ticketBody)).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.ticketTab }));
    const ticket = within(dialog).getByTestId("review-candidate-ticket");
    expect(within(ticket).getByRole("heading", { name: candidate.title })).toBeInTheDocument();
    expect(within(ticket).getByText(candidate.ticketBody)).toBeInTheDocument();
    expect(within(ticket).getByText(/GitHub Issues/)).toBeInTheDocument();
    expect(within(ticket).getByText(/acme\/payments-service/)).toBeInTheDocument();
    expect(within(dialog).queryByTestId("review-candidate-plan")).not.toBeInTheDocument();

    await user.click(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.analysisTab }));
    expect(within(dialog).getByText(LINE_INTAKE_COPY.analysisCompleteHeadline)).toBeInTheDocument();
    expect(within(dialog).getByLabelText("Run")).toBeInTheDocument();
    expect(screen.queryByTestId("work-order-split-run")).not.toBeInTheDocument();

    const ingest = within(dialog).getByTestId("split-run-phase-ingest");
    expect(within(ingest).getByRole("button", { name: "details.md" })).toBeInTheDocument();
    expect(within(ingest).getByRole("link", { name: candidate.ticketKey })).toHaveAttribute(
      "href",
      candidate.issue.url,
    );
    expect(ingest).toHaveTextContent("Completed");
    expect(ingest).toHaveTextContent("2s");
    expect(within(dialog).getByTestId("split-run-phase-analyze")).toHaveTextContent("3m 45s");
    expect(within(dialog).getByTestId("split-run-phase-plan")).toHaveTextContent("18s");
    expect(within(dialog).getByTestId("split-run-phase-score")).toHaveTextContent("7s");

    const scoreStream = within(dialog).getByTestId("split-run-stream-score");
    expect(within(scoreStream).getByText("triggered")).toBeInTheDocument();
    expect(within(scoreStream).queryByText("—")).not.toBeInTheDocument();

    const planPhase = within(dialog).getByTestId("split-run-phase-plan");
    expect(within(planPhase).getByRole("button", { name: "plan.md" })).toBeInTheDocument();
    await user.click(within(planPhase).getByRole("button", { name: "plan.md" }));
    const planDialog = await screen.findByRole("dialog", { name: "plan.md" });
    expect(planDialog).toHaveTextContent("Add a webhook-specific retry policy using the shared backoff utility.");
    await user.click(within(planDialog).getByRole("button", { name: "Close" }));

    const scorePhase = within(dialog).getByTestId("split-run-phase-score");
    expect(within(scorePhase).getByTestId("split-run-check-wo-review-pay-842-confidence")).toHaveTextContent("5/5");
    await user.click(within(scorePhase).getByTestId("split-run-check-wo-review-pay-842-confidence"));
    expect(screen.getByRole("heading", { name: "Confidence score" })).toBeInTheDocument();
    expect(screen.getByText(candidate.summary)).toBeInTheDocument();
  });

  it("shows GitHub issue fields and opens the issue in a new tab", async () => {
    const user = userEvent.setup();
    renderModal();

    const dialog = screen.getByTestId("review-candidate-modal");
    await user.click(within(dialog).getByRole("tab", { name: REVIEW_CANDIDATE_COPY.ticketTab }));

    const ticket = within(dialog).getByTestId("review-candidate-ticket");
    const openIssue = within(ticket).getByRole("link", { name: REVIEW_CANDIDATE_COPY.openIssue });
    expect(openIssue).toHaveAttribute("href", candidate.issue.url);
    expect(openIssue).toHaveAttribute("target", "_blank");
    expect(openIssue).toHaveAttribute("rel", "noopener noreferrer");

    expect(
      within(ticket).getByText(new RegExp(`Opened ${formatRelative(candidate.issue.createdAt)}`)),
    ).toBeInTheDocument();
    expect(
      within(ticket).getByText(new RegExp(`Updated ${formatRelative(candidate.issue.updatedAt)}`)),
    ).toBeInTheDocument();
    expect(within(ticket).getByTestId("review-candidate-issue-labels")).toHaveTextContent(
      candidate.issue.labels[0]!.name,
    );
    expect(within(ticket).getByTestId("review-candidate-issue-author")).toHaveTextContent(candidate.issue.author.name);
    expect(within(ticket).getByTestId("review-candidate-issue-author")).toHaveTextContent(
      `@${candidate.issue.author.login}`,
    );
    expect(within(ticket).getByTestId("review-candidate-issue-assignees")).toHaveTextContent(
      candidate.issue.assignees[0]!.name,
    );
    expect(
      within(ticket).getByText("Record the last response code and attempt count on final failure."),
    ).toBeInTheDocument();
  });

  it("closes from Back to results", async () => {
    const onClose = vi.fn();
    const user = userEvent.setup();
    renderModal(onClose);

    await user.click(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.back }));
    expect(onClose).toHaveBeenCalledTimes(1);
  });

  it("edits the implementation plan from the side button", async () => {
    const user = userEvent.setup();
    renderModal();

    const dialog = screen.getByTestId("review-candidate-modal");
    await user.click(within(dialog).getByRole("button", { name: REVIEW_CANDIDATE_COPY.editPlan }));

    const editor = within(dialog).getByLabelText(REVIEW_CANDIDATE_COPY.planEditorLabel);
    expect(editor).toHaveValue(candidate.planMarkdown);
    await user.clear(editor);
    await user.type(editor, "Retry webhooks only.");
    await user.click(within(dialog).getByRole("button", { name: REVIEW_CANDIDATE_COPY.donePlan }));

    expect(within(dialog).getByTestId("review-candidate-plan")).toHaveTextContent("Retry webhooks only.");
    expect(within(dialog).queryByLabelText(REVIEW_CANDIDATE_COPY.planEditorLabel)).not.toBeInTheDocument();
  });

  it("approves the plan without starting implementation twice", async () => {
    const user = userEvent.setup();
    renderModal();

    await user.click(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.approve }));
    expect(screen.getByRole("button", { name: REVIEW_CANDIDATE_COPY.approved })).toBeDisabled();
  });
});
