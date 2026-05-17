import { describe, expect, it } from "vitest";

import { syncViewportOnModeSwitch } from "./runs-viewport";

describe("syncViewportOnModeSwitch", () => {
  it("copies the current canvas viewport into runs mode and skips the initial fit-all", () => {
    expect(
      syncViewportOnModeSwitch({
        previousMode: null,
        nextMode: "runs",
        canvasViewport: { x: 120, y: -40, zoom: 0.72 },
      }),
    ).toEqual({
      runsViewport: { x: 120, y: -40, zoom: 0.72 },
      runsHasFitToView: true,
      skipNextRunsFitAll: true,
    });
  });

  it("falls back to a fresh runs fit when there is no canvas viewport to preserve", () => {
    expect(
      syncViewportOnModeSwitch({
        previousMode: null,
        nextMode: "runs",
      }),
    ).toEqual({
      runsViewport: undefined,
      runsHasFitToView: false,
      skipNextRunsFitAll: false,
    });
  });

  it("copies the latest runs viewport back to the canvas when leaving runs mode", () => {
    expect(
      syncViewportOnModeSwitch({
        previousMode: "runs",
        nextMode: null,
        runsViewport: { x: -300, y: 90, zoom: 0.55 },
      }),
    ).toEqual({
      canvasViewport: { x: -300, y: 90, zoom: 0.55 },
      canvasHasFitToView: true,
      skipNextRunsFitAll: false,
    });
  });
});
