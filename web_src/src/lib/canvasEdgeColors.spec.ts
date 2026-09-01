import { describe, expect, it } from "vitest";

import { CANVAS_CONNECTOR_COLOR } from "./canvasEdgeColors";

describe("CANVAS_CONNECTOR_COLOR", () => {
  it("follows the canvas edge stroke so factory append stems stay visible", () => {
    expect(CANVAS_CONNECTOR_COLOR).toBe("var(--xy-edge-stroke, var(--sp-handle-border, #C9D5E1))");
  });
});
