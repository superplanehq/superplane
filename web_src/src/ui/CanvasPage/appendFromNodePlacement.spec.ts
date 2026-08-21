import { describe, expect, it } from "vitest";
import { FACTORY_NODE_CARD_HEIGHT, FACTORY_NODE_EDIT_VERTICAL_GAP } from "@/lib/factoryCanvasChrome";
import { computeAppendFromNodePlacement } from "./appendFromNodePlacement";

describe("computeAppendFromNodePlacement", () => {
  const viewport = { x: 0, y: 0, zoom: 1 };

  it("places the factory edit placeholder using the edit vertical gap", () => {
    const { placeholderPosition } = computeAppendFromNodePlacement({
      sourcePosition: { x: 10, y: 20 },
      sourceWidth: 280,
      sourceHeight: FACTORY_NODE_CARD_HEIGHT,
      isVerticalFlow: true,
      viewport,
      canvasWidth: 2000,
      canvasHeight: 2000,
    });

    expect(placeholderPosition).toEqual({
      x: 10,
      y: 20 + FACTORY_NODE_CARD_HEIGHT + FACTORY_NODE_EDIT_VERTICAL_GAP,
    });
  });
});
