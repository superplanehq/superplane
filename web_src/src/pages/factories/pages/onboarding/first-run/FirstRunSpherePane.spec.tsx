import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { FirstRunSpherePane } from "./FirstRunSpherePane";

describe("FirstRunSpherePane", () => {
  it("renders the caption, chips, and phase labels", () => {
    render(
      <FirstRunSpherePane
        level={1}
        caption="puppies-inc/app"
        captionHighlight="Scoring:"
        leftChip={{ label: "Discover", value: "12 tickets found", tone: "amber" }}
        rightChip={{ label: "Verify", value: "Review-ready PR", tone: "amber" }}
        phasesLit
      />,
    );
    expect(screen.getByText("Scoring:")).toBeInTheDocument();
    expect(screen.getByText("puppies-inc/app")).toBeInTheDocument();
    expect(screen.getByText("12 tickets found")).toBeInTheDocument();
    expect(screen.getByText("Review-ready PR")).toBeInTheDocument();
    expect(screen.getByText("01 Plan")).toBeInTheDocument();
    expect(screen.getByText("04 Review")).toBeInTheDocument();
    expect(document.querySelector("script[data-onboarding-src]")).toBeNull();
  });

  it("exposes globe attributes and stays empty when the art script is missing", () => {
    render(
      <FirstRunSpherePane
        level={0.24}
        caption="Awaiting GitHub connection"
        art={{ mode: "globe", background: "#b7b174", arrowColor: "#eeede9", count: 100 }}
      />,
    );

    const pane = screen.getByTestId("first-run-art-pane");
    expect(pane).toHaveAttribute("data-art-mode", "globe");
    expect(pane).toHaveAttribute("data-art-background", "#b7b174");
    expect(pane).toHaveAttribute("data-art-arrow-color", "#eeede9");
    expect(pane).toHaveAttribute("data-art-count", "100");
    expect(screen.queryByText("01 Plan")).not.toBeInTheDocument();
    expect(pane.querySelector("canvas")).toBeNull();
  });
});
