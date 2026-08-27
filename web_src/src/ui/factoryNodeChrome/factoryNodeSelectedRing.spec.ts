import { describe, expect, it } from "vitest";

import {
  FACTORY_NODE_SELECTED_RING_CLASSNAME,
  factoryNodeCardFrameClassName,
  factoryNodeCardSelectedAttr,
} from "./factoryNodeSelectedRing";

describe("factoryNodeCardFrameClassName", () => {
  it("adds the shared selection ring when the node is selected", () => {
    const className = factoryNodeCardFrameClassName(true);

    expect(className).toContain("z-10");
    expect(className).toContain(FACTORY_NODE_SELECTED_RING_CLASSNAME);
  });

  it("keeps the card frame without a selection ring when the node is not selected", () => {
    const className = factoryNodeCardFrameClassName(false);

    expect(className).toContain("rounded-2xl");
    expect(className).not.toContain("z-10");
    expect(className).not.toContain("ring-4");
  });
});

describe("factoryNodeCardSelectedAttr", () => {
  it("returns true when the node is selected", () => {
    expect(factoryNodeCardSelectedAttr(true)).toBe("true");
  });

  it("returns undefined when the node is not selected", () => {
    expect(factoryNodeCardSelectedAttr(false)).toBeUndefined();
  });
});
