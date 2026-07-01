import { describe, expect, it } from "vitest";

import { canEditCanvasMemory, shouldLoadCanvasMemoryEntries } from "./canvas-memory-access";

describe("canEditCanvasMemory", () => {
  const editingState = {
    canUpdateCanvas: true,
    canvasDeletedRemotely: false,
    hasEditableVersion: true,
  };

  it("allows memory edits when the app is in edit mode for an authorized canvas", () => {
    expect(canEditCanvasMemory(editingState)).toBe(true);
  });

  it("blocks memory edits when the app is in read mode (no editable version)", () => {
    expect(canEditCanvasMemory({ ...editingState, hasEditableVersion: false })).toBe(false);
  });

  it.each([{ canUpdateCanvas: false }, { canvasDeletedRemotely: true }])("blocks memory edits when %o", (override) => {
    expect(canEditCanvasMemory({ ...editingState, ...override })).toBe(false);
  });
});

describe("shouldLoadCanvasMemoryEntries", () => {
  it.each([
    { isMemoryMode: true, isViewingLiveVersion: false, expected: true },
    { isMemoryMode: false, isViewingLiveVersion: true, expected: true },
    { isMemoryMode: true, isViewingLiveVersion: true, expected: true },
    { isMemoryMode: false, isViewingLiveVersion: false, expected: false },
  ])(
    "returns $expected when isMemoryMode=$isMemoryMode and isViewingLiveVersion=$isViewingLiveVersion",
    ({ isMemoryMode, isViewingLiveVersion, expected }) => {
      expect(shouldLoadCanvasMemoryEntries(isMemoryMode, isViewingLiveVersion)).toBe(expected);
    },
  );
});
