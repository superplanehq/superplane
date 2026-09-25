import { describe, expect, it } from "bun:test";

import { preferredFactoryScope } from "./useCanvasFactoryScope";

const pendingCanvas = { factoryId: undefined, waitingForCanvas: true };
const canvasFactory = { factoryId: "factory-from-canvas", waitingForCanvas: false };

describe("preferredFactoryScope", () => {
  it("uses the workspace id and does not wait for the canvas route", () => {
    expect(preferredFactoryScope("factory-9", pendingCanvas)).toEqual({
      factoryId: "factory-9",
      waitingForCanvas: false,
    });
  });

  it("keeps the canvas scope when no workspace id is set", () => {
    expect(preferredFactoryScope(undefined, canvasFactory)).toEqual(canvasFactory);
    expect(preferredFactoryScope("  ", pendingCanvas)).toEqual(pendingCanvas);
  });
});
