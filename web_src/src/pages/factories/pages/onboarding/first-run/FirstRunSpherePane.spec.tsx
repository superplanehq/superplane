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
  });
});
