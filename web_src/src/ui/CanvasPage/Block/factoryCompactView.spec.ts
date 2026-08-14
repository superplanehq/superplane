import { describe, expect, it } from "vitest";
import { isVerticalCanvasFlow } from "@/lib/canvasFlowDirection";

/**
 * Mirrors Block/content.tsx factory compact rules — factory canvases never collapse.
 * Kept as a focused unit so we do not mount the full Block tree.
 */
function resolveFactoryCompactView(flowDirection: "horizontal" | "vertical" | undefined, collapsed: boolean) {
  if (isVerticalCanvasFlow(flowDirection)) {
    return false;
  }
  return collapsed;
}

function resolveFactoryToggleView(
  flowDirection: "horizontal" | "vertical" | undefined,
  onToggleView: (() => void) | undefined,
) {
  if (isVerticalCanvasFlow(flowDirection)) {
    return undefined;
  }
  return onToggleView;
}

describe("factory compact view", () => {
  it("forces expanded on vertical factory flow even when collapsed is true", () => {
    expect(resolveFactoryCompactView("vertical", true)).toBe(false);
    expect(resolveFactoryCompactView("vertical", false)).toBe(false);
  });

  it("honors collapse on horizontal canvases", () => {
    expect(resolveFactoryCompactView("horizontal", true)).toBe(true);
    expect(resolveFactoryCompactView(undefined, true)).toBe(true);
  });

  it("omits toggle view on factory flow", () => {
    const toggle = () => undefined;
    expect(resolveFactoryToggleView("vertical", toggle)).toBeUndefined();
    expect(resolveFactoryToggleView("horizontal", toggle)).toBe(toggle);
  });
});
