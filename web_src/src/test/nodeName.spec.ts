import { describe, expect, it } from "vitest";

describe("Node.prototype.nodeName", () => {
  it("returns the element tag when read through the base getter", () => {
    const getter = Object.getOwnPropertyDescriptor(Node.prototype, "nodeName")?.get;
    const element = document.createElement("div");

    expect(getter?.call(element)).toBe("DIV");
  });
});
