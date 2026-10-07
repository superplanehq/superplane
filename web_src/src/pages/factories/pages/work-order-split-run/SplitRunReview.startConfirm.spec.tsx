import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { render, screen, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "bun:test";

import { TooltipProvider } from "@/ui/tooltip";

import type { FactoriesWorkOrderCheck } from "@/api-client";

import { DRAFT_WORK_ORDER } from "../../__fixtures__/factoryPageResponses";
import { REVIEW_CANDIDATE_WORK_ORDERS } from "../onboarding/first-run/reviewCandidates";
import { SplitRunReview } from "./SplitRunReview";
import { START_CONFIRM_COPY, START_CONFIRM_STORAGE_KEY } from "./startConfirm";
import { splitRunFixtureForWorkOrder } from "./splitRunMocks";
import type { SplitRunFooter } from "./splitRunFooter";

function reviewCheck(key: "clarity" | "complexity" | "verifiability", score: number): FactoriesWorkOrderCheck {
  return {
    id: `check-${key}`,
    key,
    name: key === "clarity" ? "Clarity score" : key.charAt(0).toUpperCase() + key.slice(1),
    score,
    maxScore: 3,
    format: "FORMAT_FRACTION",
    level: score >= 3 ? "LEVEL_POSITIVE" : score >= 2 ? "LEVEL_CAUTION" : "LEVEL_CRITICAL",
  };
}

function reviewFooter(scores: { clarity: number; complexity: number; verifiability: number }): SplitRunFooter {
  return splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER, {
    demoArtifacts: false,
    checks: [
      reviewCheck("clarity", scores.clarity),
      reviewCheck("complexity", scores.complexity),
      reviewCheck("verifiability", scores.verifiability),
    ],
  }).footer;
}

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

  it("explains a Clarity score of 2 or lower", async () => {
    const user = userEvent.setup();
    renderConfirmReview(vi.fn(), {
      ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer,
      clarityScore: 2,
      confidenceScore: 5,
    });
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(screen.getByText(START_CONFIRM_COPY.low)).toBeInTheDocument();
  });

  it("explains a low Confidence or a score of 3", async () => {
    const user = userEvent.setup();
    renderConfirmReview(vi.fn(), {
      ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer,
      clarityScore: 5,
      confidenceScore: 2,
    });
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(screen.getByText(START_CONFIRM_COPY.mid)).toBeInTheDocument();
  });

  it("starts a review headline of 3 without the steering dialog", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderConfirmReview(onStart, reviewFooter({ clarity: 3, complexity: 3, verifiability: 3 }));

    expect(screen.getByText("This task is ready to start")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("split-run-start-confirm")).not.toBeInTheDocument();
  });

  it("still asks when a review headline is 1 or 2", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderConfirmReview(onStart, reviewFooter({ clarity: 3, complexity: 2, verifiability: 3 }));

    expect(screen.getByText("Review the plan before you start")).toBeInTheDocument();
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).not.toHaveBeenCalled();
    expect(screen.getByText(START_CONFIRM_COPY.mid)).toBeInTheDocument();
  });

  it("still asks for an old score of 3", async () => {
    const user = userEvent.setup();
    renderConfirmReview(vi.fn(), {
      ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer,
      clarityScore: 5,
      confidenceScore: 3,
    });
    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(screen.getByText(START_CONFIRM_COPY.mid)).toBeInTheDocument();
  });

  it("starts immediately when both scores are 4 or higher", async () => {
    const user = userEvent.setup();
    const onStart = vi.fn();
    renderConfirmReview(onStart, {
      ...splitRunFixtureForWorkOrder(DRAFT_WORK_ORDER).footer,
      clarityScore: 4,
      confidenceScore: 5,
    });

    await user.click(screen.getByRole("button", { name: "Start" }));
    expect(onStart).toHaveBeenCalledTimes(1);
    expect(screen.queryByTestId("split-run-start-confirm")).not.toBeInTheDocument();
  });

  it("starts immediately on an intake draft with a high Confidence", async () => {
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
