import { describe, expect, it } from "vitest";
import { computeFitScale } from "./mermaidSvgSize";

describe("computeFitScale", () => {
  it("fits content into the viewport and respects the max scale", () => {
    expect(computeFitScale(400, 300, 200, 100)).toBe(2);
    expect(computeFitScale(400, 300, 50, 50, 5)).toBe(5);
    expect(computeFitScale(400, 300, 0, 100)).toBe(1);
  });
});
