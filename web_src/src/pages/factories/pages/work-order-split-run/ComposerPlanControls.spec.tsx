import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { ComposerPlanStack } from "./ComposerPlanControls";

describe("ComposerPlanStack", () => {
  it("lets the settings row wrap inside the available width", () => {
    render(
      <ComposerPlanStack
        open={false}
        clarity={{ score: 4, summary: "The prompt is specific." }}
        confidence={{ score: 4, summary: "The change fits this line." }}
        modelSelect={<button type="button">Model</button>}
        actions={<button type="button">Start</button>}
      />,
    );

    const chips = screen.getByTestId("split-run-intent-composer-chips");
    const settings = screen.getByTestId("split-run-intent-settings");
    const scores = screen.getByTestId("score-evidence-row");

    expect(chips).toHaveClass("flex-wrap", "min-w-0");
    expect(settings).toHaveClass("flex-wrap", "max-w-full", "shrink-0");
    expect(settings.className).toContain("min(100%,max-content)");
    expect(scores).toHaveClass("max-w-full", "shrink-0");
    expect(scores.className).toContain("min(100%,max-content)");
  });
});
