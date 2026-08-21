import { describe, expect, it } from "vitest";
import {
  FACTORY_NODE_CARD_HEIGHT,
  FACTORY_NODE_EDIT_VERTICAL_GAP,
  FACTORY_NODE_VERTICAL_GAP,
} from "./factoryCanvasChrome";
import {
  expandFactoryEditVerticalPositions,
  FACTORY_COMPACT_LAYER_STRIDE,
  FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER,
} from "./factoryEditVerticalSpacing";

function node(id: string, x: number, y: number) {
  return { id, position: { x, y } };
}

describe("expandFactoryEditVerticalPositions", () => {
  it("leaves a single node unchanged", () => {
    const nodes = [node("only", 10, 20)];
    expect(expandFactoryEditVerticalPositions(nodes)).toBe(nodes);
  });

  it("leaves same-layer nodes on the same Y", () => {
    const nodes = [node("left", 0, 0), node("right", 400, 0)];
    expect(expandFactoryEditVerticalPositions(nodes)).toEqual(nodes);
  });

  it("adds extra Y per stacked rank so edit gap fits node actions and edge delete", () => {
    const stride = FACTORY_COMPACT_LAYER_STRIDE;
    const nodes = [node("top", 0, 0), node("mid", 0, stride), node("bottom", 0, stride * 2)];

    const expanded = expandFactoryEditVerticalPositions(nodes);
    const extra = FACTORY_EDIT_VERTICAL_EXTRA_PER_LAYER;

    expect(expanded[0].position).toEqual({ x: 0, y: 0 });
    expect(expanded[1].position).toEqual({ x: 0, y: stride + extra });
    expect(expanded[2].position).toEqual({ x: 0, y: stride * 2 + extra * 2 });

    const gapBetweenCards = expanded[1].position.y - FACTORY_NODE_CARD_HEIGHT;
    expect(gapBetweenCards).toBe(FACTORY_NODE_EDIT_VERTICAL_GAP);
    expect(FACTORY_NODE_EDIT_VERTICAL_GAP).toBeGreaterThan(FACTORY_NODE_VERTICAL_GAP);
  });

  it("keeps side-by-side ranks aligned after stretching", () => {
    const stride = FACTORY_COMPACT_LAYER_STRIDE;
    const nodes = [node("a1", 0, 0), node("a2", 0, stride), node("b1", 400, 0), node("b2", 400, stride)];

    const expanded = expandFactoryEditVerticalPositions(nodes);
    expect(expanded.find((item) => item.id === "a1")?.position.y).toBe(0);
    expect(expanded.find((item) => item.id === "b1")?.position.y).toBe(0);
    expect(expanded.find((item) => item.id === "a2")?.position.y).toBe(
      expanded.find((item) => item.id === "b2")?.position.y,
    );
  });

  it("does not change X", () => {
    const stride = FACTORY_COMPACT_LAYER_STRIDE;
    const nodes = [node("top", 42, 0), node("bottom", 42, stride)];
    const expanded = expandFactoryEditVerticalPositions(nodes);
    expect(expanded.map((item) => item.position.x)).toEqual([42, 42]);
  });
});
