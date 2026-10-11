import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "bun:test";

import { OwnerSetupPane } from "./OwnerSetupPane";

describe("OwnerSetupPane", () => {
  it("draws a still pixel mark instead of the factory phases", () => {
    render(<OwnerSetupPane caption="Awaiting owner" step={1} stepCount={2} />);

    const pane = screen.getByTestId("owner-setup-pane");
    expect(pane).toHaveAttribute("data-step", "1");
    expect(pane).toHaveAttribute("data-step-count", "2");
    expect(pane.querySelector("canvas")).not.toBeNull();
    expect(screen.getByText("Awaiting owner")).toBeInTheDocument();
    expect(screen.queryByText("01 Plan")).not.toBeInTheDocument();
  });
});
