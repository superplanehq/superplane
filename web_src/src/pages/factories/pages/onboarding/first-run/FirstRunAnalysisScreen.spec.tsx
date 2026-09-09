import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";

describe("FirstRunAnalysisScreen", () => {
  it("shows live scoring progress and opens the board on click", async () => {
    const user = userEvent.setup();
    const onGoToBoard = vi.fn();
    render(
      <FirstRunAnalysisScreen progress={{ total: 12, scored: 4, ready: 2, stageIndex: 1 }} onGoToBoard={onGoToBoard} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImported(12))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoring(4, 12))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.note)).toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-go-to-board"));
    expect(onGoToBoard).toHaveBeenCalled();
  });

  it("shows the import stage before any runs arrive", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImporting)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoringPending)).toBeInTheDocument();
  });

  // When scoring finishes, the last row is the result, not another wait.
  it("shows the scored result instead of a fake board-building stage", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 10, scored: 10, ready: 6, stageIndex: 2 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScored(10, 6))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.noteDone)).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.analysis.note)).not.toBeInTheDocument();
    expect(document.querySelector(".animate-spin")).not.toBeInTheDocument();
  });

  it("omits the ready fragment when no ticket scored above the threshold", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 3, scored: 3, ready: 0, stageIndex: 2 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScored(3, 0))).toBeInTheDocument();
  });
});
