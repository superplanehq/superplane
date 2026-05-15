import { describe, expect, it } from "vitest";
import { getRunsFitAllDecision, prepareRunsViewportOnModeEntry } from "./runs-viewport";

describe("prepareRunsViewportOnModeEntry", () => {
  it("seeds runs viewport from current canvas viewport when available", () => {
    const canvasViewport = { x: 120, y: 240, zoom: 0.75 };
    const existingRunsViewport = { x: 10, y: 20, zoom: 1.3 };

    const result = prepareRunsViewportOnModeEntry({
      currentCanvasViewport: canvasViewport,
      existingRunsViewport,
      hasFitToView: false,
    });

    expect(result).toEqual({
      runsViewport: canvasViewport,
      hasFitToView: true,
      seededFromCanvasViewport: true,
    });
  });

  it("falls back to existing runs viewport when current canvas viewport is unavailable", () => {
    const existingRunsViewport = { x: 30, y: 40, zoom: 0.9 };

    const result = prepareRunsViewportOnModeEntry({
      currentCanvasViewport: undefined,
      existingRunsViewport,
      hasFitToView: false,
    });

    expect(result).toEqual({
      runsViewport: existingRunsViewport,
      hasFitToView: true,
      seededFromCanvasViewport: false,
    });
  });

  it("preserves fit state when no viewport exists", () => {
    const result = prepareRunsViewportOnModeEntry({
      currentCanvasViewport: undefined,
      existingRunsViewport: undefined,
      hasFitToView: false,
    });

    expect(result).toEqual({
      runsViewport: undefined,
      hasFitToView: false,
      seededFromCanvasViewport: false,
    });
  });
});

describe("getRunsFitAllDecision", () => {
  it("skips fit-all once after entering runs mode from canvas viewport", () => {
    const result = getRunsFitAllDecision({
      isRunsMode: true,
      runCanvasNodeIdsKey: "node-a|node-b",
      skipInitialRunsFitAll: true,
    });

    expect(result).toEqual({
      shouldFitAll: false,
      skipInitialRunsFitAll: false,
    });
  });

  it("fits all when runs mode is active and skip flag is cleared", () => {
    const result = getRunsFitAllDecision({
      isRunsMode: true,
      runCanvasNodeIdsKey: "node-a|node-b",
      skipInitialRunsFitAll: false,
    });

    expect(result).toEqual({
      shouldFitAll: true,
      skipInitialRunsFitAll: false,
    });
  });

  it("does nothing when runs mode is inactive", () => {
    const result = getRunsFitAllDecision({
      isRunsMode: false,
      runCanvasNodeIdsKey: "node-a|node-b",
      skipInitialRunsFitAll: true,
    });

    expect(result).toEqual({
      shouldFitAll: false,
      skipInitialRunsFitAll: true,
    });
  });
});
