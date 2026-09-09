import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";

describe("FirstRunAnalysisScreen", () => {
  it("shows live scoring progress and opens the board on click", async () => {
    const user = userEvent.setup();
    const onGoToBoard = vi.fn();
    render(<FirstRunAnalysisScreen progress={{ total: 12, scored: 4, stageIndex: 1 }} onGoToBoard={onGoToBoard} />);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImported(12))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageScoring(4, 12))).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.note)).toBeInTheDocument();

    await user.click(screen.getByTestId("first-run-go-to-board"));
    expect(onGoToBoard).toHaveBeenCalled();
  });

  it("shows the import stage before any runs arrive", () => {
    render(<FirstRunAnalysisScreen progress={{ total: 0, scored: 0, stageIndex: 0 }} onGoToBoard={vi.fn()} />);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.stageImporting)).toBeInTheDocument();
  });
});
