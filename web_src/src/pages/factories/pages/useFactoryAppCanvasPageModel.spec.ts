import { describe, expect, it } from "vitest";

import { resolveCanUpdateFactoryAutomation } from "./useFactoryAppCanvasPageModel";

describe("resolveCanUpdateFactoryAutomation", () => {
  it("is false while permissions load", () => {
    expect(resolveCanUpdateFactoryAutomation(true, () => true)).toBe(false);
  });

  it("is false without canvases update", () => {
    expect(
      resolveCanUpdateFactoryAutomation(false, (resource, action) => resource === "canvases" && action === "read"),
    ).toBe(false);
  });

  it("is true after permissions load with canvases update", () => {
    expect(
      resolveCanUpdateFactoryAutomation(false, (resource, action) => resource === "canvases" && action === "update"),
    ).toBe(true);
  });
});
