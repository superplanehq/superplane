import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { SplitRunReview } from "./SplitRunReview";
import { START_CONFIRM_COPY, START_CONFIRM_STORAGE_KEY } from "./startConfirm";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";

function renderConfirmReview(onStart: () => void, footer = splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer) {
  return render(
    <QueryClientProvider client={new QueryClient({ defaultOptions: { queries: { retry: false } } })}>
      <TooltipProvider>
        <SplitRunReview footer={footer} onStart={onStart} confirmUnclearStart />
      </TooltipProvider>
    </QueryClientProvider>,
  );
}

describe("SplitRunReview start confirm", () => {
  beforeEach(() => {
    window.localStorage.clear();
  });

  afterEach(() => {
    window.localStorage.clear();
  });

  it("asks before Start when analysis has no score", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderConfirmReview(onStart);

    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).not.toHaveBeenCalled();
    const dialog = screen.getByTestId("split-run-start-confirm");
    expect(dialog).toHaveAttribute("data-slot", "frame");
    expect(dialog.closest("[role=alertdialog]")).toHaveClass("bg-popover");
    expect(dialog.closest("[role=alertdialog]")).not.toHaveClass("bg-transparent");
    expect(dialog).toHaveClass("bg-popover", "p-0", "border-0");
    expect(
      within(dialog).getByRole("heading", { name: START_CONFIRM_COPY.title }).closest("[data-slot=frame-panel-header]"),
    ).not.toBeNull();
    expect(within(dialog).getByText(START_CONFIRM_COPY.missing)).toBeInTheDocument();
    const panel = within(dialog).getByText(START_CONFIRM_COPY.skip).closest("[data-slot=frame-panel]");
    expect(panel).not.toBeNull();
    expect(within(panel as HTMLElement).getByRole("button", { name: START_CONFIRM_COPY.cancel })).toHaveClass(
      "rounded-md",
    );
    expect(within(panel as HTMLElement).getByRole("button", { name: START_CONFIRM_COPY.confirm })).toHaveClass(
      "rounded-md",
    );
    expect(within(dialog).getByLabelText(START_CONFIRM_COPY.skip)).toBeInTheDocument();

    await user.click(within(dialog).getByRole("button", { name: START_CONFIRM_COPY.confirm }));
    expect(onStart).toHaveBeenCalledTimes(1);
  });

  it("explains a score below 3", async () => {
    const user = userEvent.setup();
    renderConfirmReview(vi.fn(), { ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer, confidenceScore: 2 });
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(screen.getByText(START_CONFIRM_COPY.low)).toBeInTheDocument();
  });

  it("explains a score of 3 or 4", async () => {
    const user = userEvent.setup();
    renderConfirmReview(vi.fn(), { ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer, confidenceScore: 4 });
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(screen.getByText(START_CONFIRM_COPY.mid)).toBeInTheDocument();
  });

  it("starts immediately at score 5", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderConfirmReview(onStart, splitRunFixtureForWorkOrder(REVIEW_CANDIDATE_WORK_ORDERS[0]).footer);

    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("split-run-start-confirm")).not.toBeInTheDocument();
  });

  it("remembers Do not ask again", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    const { unmount } = renderConfirmReview(onStart);

    await user.click(screen.getByRole("button", { name: "Start" }));
    await user.click(screen.getByTestId("split-run-start-dont-ask"));
    await user.click(screen.getByRole("button", { name: START_CONFIRM_COPY.confirm }));
    expect(window.localStorage.getItem(START_CONFIRM_STORAGE_KEY)).toBe("1");
    expect(onStart).toHaveBeenCalledTimes(1);
    unmount();

    renderConfirmReview(onStart);
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(2);
    expect(screen.queryByRole("heading", { name: START_CONFIRM_COPY.title })).not.toBeInTheDocument();
  });
});
