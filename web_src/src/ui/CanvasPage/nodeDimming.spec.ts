import { describe, expect, it } from "vitest";
import { isCanvasNodeHighlighted, shouldBlankCanvasNodeBody } from "./nodeDimming";

describe("nodeDimming", () => {
  it("does not blank bodies for run participants during edge hover", () => {
    const runParticipantSet = new Set(["run-node-1", "run-node-2", "run-node-3"]);
    const highlightedNodeIds = new Set(["run-node-1", "run-node-2"]);

    expect(
      isCanvasNodeHighlighted({
        nodeId: "run-node-3",
        edgeHoverActive: true,
        highlightedNodeIds,
        runDimActive: true,
        runParticipantSet,
      }),
    ).toBe(false);

    expect(
      shouldBlankCanvasNodeBody({
        nodeId: "run-node-3",
        edgeHoverActive: true,
        runDimActive: true,
        runParticipantSet,
      }),
    ).toBe(false);
  });

  it("blanks bodies only for non-participants when runs dimming is active without edge hover", () => {
    const runParticipantSet = new Set(["run-node-1", "run-node-2"]);

    expect(
      shouldBlankCanvasNodeBody({
        nodeId: "run-node-1",
        edgeHoverActive: false,
        runDimActive: true,
        runParticipantSet,
      }),
    ).toBe(false);

    expect(
      shouldBlankCanvasNodeBody({
        nodeId: "other-node",
        edgeHoverActive: false,
        runDimActive: true,
        runParticipantSet,
      }),
    ).toBe(true);
  });
});
