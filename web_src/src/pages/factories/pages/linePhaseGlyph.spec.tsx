import { render } from "@testing-library/react";
import { describe, expect, it } from "vitest";

import { PhaseGlyph } from "./linePhaseGlyph";

describe("PhaseGlyph", () => {
  it("fills passed and failed so status reads at a glance", () => {
    const { rerender, container } = render(<PhaseGlyph kind="passed" />);
    expect(container.firstElementChild?.className).toMatch(/status-completed-dot/);
    expect(container.querySelector(".lucide-check")).not.toBeNull();

    rerender(<PhaseGlyph kind="failed" />);
    expect(container.firstElementChild?.className).toMatch(/status-failed-dot/);
    expect(container.querySelector(".lucide-x")).not.toBeNull();
  });

  it("fills running and waiting disks", () => {
    const { rerender, container } = render(<PhaseGlyph kind="running" />);
    expect(container.firstElementChild?.className).toMatch(/status-running-dot/);
    expect(container.querySelector(".lucide-loader-circle")).not.toBeNull();

    rerender(<PhaseGlyph kind="waiting" />);
    expect(container.firstElementChild?.className).toMatch(/status-waiting-dot/);
    expect(container.querySelector(".lucide-clock")).not.toBeNull();
  });
});
