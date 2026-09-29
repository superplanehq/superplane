import { describe, expect, it } from "bun:test";

import { checkTickBar } from "./consoleCheckRows";

describe("checkTickBar", () => {
  it("draws one box per point on a five-point score", () => {
    expect(checkTickBar({ score: 4, maxScore: 5 })).toEqual({ total: 5, filled: 4 });
    expect(checkTickBar({ score: 5, maxScore: 5 })).toEqual({ total: 5, filled: 5 });
  });

  it("uses a dense bar for a percent", () => {
    expect(checkTickBar({ score: 80, maxScore: 100, format: "percent" })).toEqual({ total: 20, filled: 16 });
  });

  it("uses one box for a pass or fail", () => {
    expect(checkTickBar({ score: 1, maxScore: 1, format: "boolean" })).toEqual({ total: 1, filled: 1 });
    expect(checkTickBar({ score: 0, maxScore: 1, format: "boolean" })).toEqual({ total: 1, filled: 0 });
  });
});
