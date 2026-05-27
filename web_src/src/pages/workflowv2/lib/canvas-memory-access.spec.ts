import { describe, expect, it } from "vitest";

import { canEditCanvasMemory } from "./canvas-memory-access";

describe("canEditCanvasMemory", () => {
  const editingState = {
    canUpdateCanvas: true,
    isTemplate: false,
    canvasDeletedRemotely: false,
    hasEditableVersion: true,
  };

  it("allows memory edits when the app is in edit mode for an authorized canvas", () => {
    expect(canEditCanvasMemory(editingState)).toBe(true);
  });

  it("blocks memory edits when the app is in read mode (no editable version)", () => {
    expect(canEditCanvasMemory({ ...editingState, hasEditableVersion: false })).toBe(false);
  });

  it.each([{ canUpdateCanvas: false }, { isTemplate: true }, { canvasDeletedRemotely: true }])(
    "blocks memory edits when %o",
    (override) => {
      expect(canEditCanvasMemory({ ...editingState, ...override })).toBe(false);
    },
  );
});
