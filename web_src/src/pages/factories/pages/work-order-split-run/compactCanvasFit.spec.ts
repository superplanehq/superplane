import { describe, expect, it } from "vitest";

import { compactCanvasFitKey, shouldFitCompactCanvas } from "./compactCanvasFit";

describe("compactCanvasFitKey", () => {
  it("uses empty when there are no nodes", () => {
    expect(compactCanvasFitKey([])).toBe("empty");
  });

  it("joins node ids in a stable order", () => {
    expect(compactCanvasFitKey(["step-1", "kickoff"])).toBe("kickoff|step-1");
    expect(compactCanvasFitKey(["kickoff", "step-1"])).toBe("kickoff|step-1");
  });

  it("fits only after live nodes arrive", () => {
    expect(shouldFitCompactCanvas("empty")).toBe(false);
    expect(shouldFitCompactCanvas("kickoff|step-1")).toBe(true);
  });
});
