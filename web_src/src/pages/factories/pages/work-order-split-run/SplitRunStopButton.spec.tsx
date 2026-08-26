import { render, screen } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { SplitRunStopButton } from "./SplitRunStopButton";

describe("SplitRunStopButton", () => {
  it("updates the default action when the footer kind changes", () => {
    const { rerender } = render(<SplitRunStopButton kind="running" />);

    expect(screen.getByRole("button", { name: "Stop and Close" })).toBeInTheDocument();

    rerender(<SplitRunStopButton kind="failed" />);

    expect(screen.getByRole("button", { name: "Rerun step" })).toBeInTheDocument();
  });
});
