import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";

describe("FirstRunAnalysisScreen", () => {
  it("names the imported scope and source, and opens the board on click", async () => {
    const user = userEvent.setup();
    const onGoToBoard = vi.fn();
    render(
      <FirstRunAnalysisScreen
        progress={{ total: 12, scored: 4, ready: 2, stageIndex: 1 }}
        sourceName="GitHub issues"
        onGoToBoard={onGoToBoard}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImported(12, "GitHub issues"))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoring)).toBeInTheDocument();
    expect(screen.getByTestId("first-run-ready-count")).toHaveTextContent(FIRST_RUN_COPY.analysis.readyCount(2));
    // The hand-off must not read as work in progress.
    expect(document.querySelector(".animate-spin")).not.toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-go-to-board"));
    expect(onGoToBoard).toHaveBeenCalled();
  });

  it("hides the ready counter until a ticket scores", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 12, scored: 0, ready: 0, stageIndex: 1 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoring)).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-ready-count")).not.toBeInTheDocument();
  });

  it("shows the import stage before any runs arrive", () => {
    render(<FirstRunAnalysisScreen progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0 }} onGoToBoard={vi.fn()} />);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImporting)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoringPending)).toBeInTheDocument();
  });

  // When scoring finishes, the last row is the result, not another wait.
  it("shows the scored result with the ready counter when scoring is done", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 10, scored: 10, ready: 6, stageIndex: 2 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScored(10))).toBeInTheDocument();
    expect(screen.getByTestId("first-run-ready-count")).toHaveTextContent(FIRST_RUN_COPY.analysis.readyCount(6));
    expect(document.querySelector(".animate-spin")).not.toBeInTheDocument();
  });

  it("says the source was empty and points to manual tasks", () => {
    render(
      <FirstRunAnalysisScreen
        progress={{ total: 0, scored: 0, ready: 0, stageIndex: 0, empty: true }}
        sourceName="GitHub issues"
        onGoToBoard={vi.fn()}
      />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.emptyImport("GitHub issues"))).toBeInTheDocument();
    expect(screen.getByTestId("first-run-analysis-empty")).toHaveTextContent(FIRST_RUN_COPY.analysis.emptyNext);
    expect(screen.getByText("🤔")).toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.analysis.stageImporting)).not.toBeInTheDocument();
    expect(screen.queryByText(FIRST_RUN_COPY.analysis.stageScoringPending)).not.toBeInTheDocument();
    expect(document.querySelector(".text-emerald-600")).not.toBeInTheDocument();
    expect(document.querySelector(".animate-spin")).not.toBeInTheDocument();
  });

  it("omits the ready counter when no ticket scored above the threshold", () => {
    render(
      <FirstRunAnalysisScreen progress={{ total: 3, scored: 3, ready: 0, stageIndex: 2 }} onGoToBoard={vi.fn()} />,
    );

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScored(3))).toBeInTheDocument();
    expect(screen.queryByTestId("first-run-ready-count")).not.toBeInTheDocument();
  });
});
