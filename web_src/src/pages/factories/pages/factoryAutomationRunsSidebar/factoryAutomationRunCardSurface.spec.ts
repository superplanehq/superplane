import { describe, expect, it } from "bun:test";

import { factoryAutomationRunCardSurfaceClass } from "./factoryAutomationRunCardSurface";

describe("factoryAutomationRunCardSurfaceClass", () => {
  it("keeps hover on the card and does not paint the column", () => {
    const idle = factoryAutomationRunCardSurfaceClass(false);
    expect(idle).toContain("hover:bg-slate-100");
    expect(idle).not.toContain("bg-sky-50");
    expect(idle).not.toContain("bg-slate-200");
  });

  it("marks the selected card without a sky column wash", () => {
    const selected = factoryAutomationRunCardSurfaceClass(true);
    expect(selected).toContain("bg-slate-200/80");
    expect(selected).not.toContain("bg-sky-50");
  });
});
