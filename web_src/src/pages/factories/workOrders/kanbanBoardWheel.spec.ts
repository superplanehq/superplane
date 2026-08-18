import { describe, expect, it } from "vitest";

import { shouldRedirectWheelToHorizontalScroll } from "./kanbanBoardWheel";

function wheel(partial: Partial<Pick<WheelEvent, "deltaX" | "deltaY" | "target">>) {
  return { deltaX: 0, deltaY: 40, target: null, ...partial };
}

describe("shouldRedirectWheelToHorizontalScroll", () => {
  it("returns false when the board does not overflow on x", () => {
    const board = document.createElement("div");
    Object.defineProperty(board, "scrollWidth", { value: 400 });
    Object.defineProperty(board, "clientWidth", { value: 400 });

    expect(shouldRedirectWheelToHorizontalScroll(wheel({}), board)).toBe(false);
  });

  it("returns true for a vertical wheel when the board overflows on x", () => {
    const board = document.createElement("div");
    Object.defineProperty(board, "scrollWidth", { value: 800 });
    Object.defineProperty(board, "clientWidth", { value: 400 });

    expect(shouldRedirectWheelToHorizontalScroll(wheel({ deltaY: 40 }), board)).toBe(true);
  });

  it("leaves a native horizontal wheel to the board", () => {
    const board = document.createElement("div");
    Object.defineProperty(board, "scrollWidth", { value: 800 });
    Object.defineProperty(board, "clientWidth", { value: 400 });

    expect(shouldRedirectWheelToHorizontalScroll(wheel({ deltaX: 40, deltaY: 10 }), board)).toBe(false);
  });
});
