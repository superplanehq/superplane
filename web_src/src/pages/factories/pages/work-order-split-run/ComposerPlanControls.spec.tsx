import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { TooltipProvider } from "@/ui/tooltip";

import { DRAFT_READINESS_NOTES } from "../../lib/draftReadiness";
import { ComposerPlanStack } from "./ComposerPlanControls";

function renderStrip(confidence: { score: number; maxScore: number }) {
  return render(
    <TooltipProvider>
      <ComposerPlanStack open={false} confidence={confidence} canTogglePlan={false} />
    </TooltipProvider>,
  );
}

describe("ComposerPlanStack review verdict", () => {
  it("says a 3/3 review is ready and a 2/3 review still warns", () => {
    const { unmount } = renderStrip({ score: 3, maxScore: 3 });
    const ready = screen.getByTestId("split-run-intent-verdict");
    expect(ready).toHaveAttribute("data-tone", "ready");
    expect(ready).toHaveTextContent(DRAFT_READINESS_NOTES.ready.headline);
    unmount();

    renderStrip({ score: 2, maxScore: 3 });
    const caution = screen.getByTestId("split-run-intent-verdict");
    expect(caution).toHaveAttribute("data-tone", "caution");
    expect(caution).toHaveTextContent(DRAFT_READINESS_NOTES.uncertain.headline);
  });

  it("keeps an old score of 3 as a warning", () => {
    renderStrip({ score: 3, maxScore: 5 });
    expect(screen.getByTestId("split-run-intent-verdict")).toHaveAttribute("data-tone", "caution");
  });
});
