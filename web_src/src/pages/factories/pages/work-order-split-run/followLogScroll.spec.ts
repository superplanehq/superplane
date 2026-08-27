import { describe, expect, it } from "vitest";

import { followAfterRunningPhaseChange, isNearLogBottom } from "./followLogScroll";

describe("followAfterRunningPhaseChange", () => {
  it("turns Follow on when a phase starts running", () => {
    expect(followAfterRunningPhaseChange(false, null, "implement")).toBe(true);
  });

  it("turns Follow on when a later phase starts running", () => {
    expect(followAfterRunningPhaseChange(false, "implement", "verify")).toBe(true);
  });

  it("keeps Follow as the user left it when the run finishes", () => {
    expect(followAfterRunningPhaseChange(true, "implement", null)).toBe(true);
    expect(followAfterRunningPhaseChange(false, "implement", null)).toBe(false);
  });

  it("does not turn Follow back on while the same phase stays running", () => {
    expect(followAfterRunningPhaseChange(false, "implement", "implement")).toBe(false);
  });
});

describe("isNearLogBottom", () => {
  it("is true within 16 pixels of the bottom", () => {
    expect(isNearLogBottom(84, 200, 100)).toBe(true);
  });

  it("is false when the user scrolls more than 16 pixels up", () => {
    expect(isNearLogBottom(80, 200, 100)).toBe(false);
  });
});
