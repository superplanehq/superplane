import { describe, expect, it } from "vitest";

describe("jsdom DOMMatrixReadOnly", () => {
  it("exposes a constructor so React Flow can read viewport zoom", () => {
    const matrix = new window.DOMMatrixReadOnly("matrix(0.8, 0, 0, 0.8, 10, 20)");

    expect(matrix.m22).toBe(0.8);
  });

  it("treats none as identity zoom", () => {
    expect(new window.DOMMatrixReadOnly("none").m22).toBe(1);
  });
});
