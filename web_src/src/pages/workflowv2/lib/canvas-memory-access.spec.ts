import { describe, expect, it } from "vitest";

import { canEditCanvasMemory } from "./canvas-memory-access";

describe("canEditCanvasMemory", () => {
  const defaultState = {
    canUpdateCanvas: true,
    isTemplate: false,
    canvasDeletedRemotely: false,
    isViewingLiveVersion: true,
    isViewingDraftVersion: false,
  };

  it("allows memory edits on the current live non-draft version", () => {
    expect(canEditCanvasMemory(defaultState)).toBe(true);
  });

  it.each([
    { canUpdateCanvas: false },
    { isTemplate: true },
    { canvasDeletedRemotely: true },
    { isViewingLiveVersion: false },
    { isViewingDraftVersion: true },
  ])("blocks memory edits when %o", (override) => {
    expect(canEditCanvasMemory({ ...defaultState, ...override })).toBe(false);
  });
});
