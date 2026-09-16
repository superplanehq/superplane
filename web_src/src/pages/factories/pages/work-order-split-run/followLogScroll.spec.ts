import { describe, expect, it } from "bun:test";

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
  it("is true within 144 pixels of the bottom", () => {
    expect(isNearLogBottom(56, 300, 100)).toBe(true);
  });

  it("is false when the user scrolls more than 144 pixels up", () => {
    expect(isNearLogBottom(55, 300, 100)).toBe(false);
  });
});
