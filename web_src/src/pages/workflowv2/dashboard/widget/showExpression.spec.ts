import { describe, expect, it } from "vitest";

import { evaluateShow } from "./showExpression";

describe("evaluateShow", () => {
  it("evaluates string equality", () => {
    expect(evaluateShow('row.status == "failed"', { status: "failed" })).toBe(true);
    expect(evaluateShow('row.status == "failed"', { status: "passed" })).toBe(false);
  });

  it("evaluates numeric comparisons", () => {
    expect(evaluateShow("row.count > 5", { count: 10 })).toBe(true);
    expect(evaluateShow("row.count > 5", { count: 1 })).toBe(false);
  });

  it("supports logical operators and parentheses", () => {
    const row = { status: "failed", retried: true };
    expect(evaluateShow('(row.status == "failed") && !row.retried', row)).toBe(false);
    expect(evaluateShow('row.status == "failed" || row.status == "running"', row)).toBe(true);
  });

  it("returns the default value on parse error", () => {
    expect(evaluateShow("row.status ===", {}, false)).toBe(false);
    expect(evaluateShow("totally bogus expression !!!!", {})).toBe(true);
  });

  it("supports bare field references without `row.` prefix", () => {
    expect(evaluateShow('status == "ok"', { status: "ok" })).toBe(true);
  });
});
