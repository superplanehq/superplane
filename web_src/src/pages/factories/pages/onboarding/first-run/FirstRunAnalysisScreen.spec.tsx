import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { describe, expect, it, vi } from "vitest";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunAnalysisScreen } from "./FirstRunAnalysisScreen";

describe("FirstRunAnalysisScreen", () => {
  it("shows the overrun notice while stages keep running", () => {
    render(<FirstRunAnalysisScreen status="overrun" currentStageIndex={2} onRetry={vi.fn()} />);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.overrun)).toBeInTheDocument();
    expect(screen.getByText(FIRST_RUN_COPY.analysis.leaveHint)).toBeInTheDocument();
  });

  it("offers a retry when analysis fails", async () => {
    const user = userEvent.setup();
    const onRetry = vi.fn();
    render(<FirstRunAnalysisScreen status="failed" currentStageIndex={0} onRetry={onRetry} />);

    expect(screen.getByText(FIRST_RUN_COPY.analysis.failure)).toBeInTheDocument();
    await user.click(screen.getByTestId("first-run-run-again"));
    expect(onRetry).toHaveBeenCalled();
  });
});
