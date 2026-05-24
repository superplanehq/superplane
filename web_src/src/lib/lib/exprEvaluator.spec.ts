import { describe, expect, it } from "vitest";
import { evaluateExpr, formatExprResult } from "@/lib/exprEvaluator";

describe("exprEvaluator", () => {
  it("evaluates expression functions and field access", () => {
    expect(evaluateExpr('upper("hello")', {})).toBe("HELLO");
    expect(evaluateExpr('$["node"].data.name', { node: { data: { name: "test" } } })).toBe("test");
  });

  it("formats primitive, array, and object results for display", () => {
    expect(formatExprResult(null)).toBe("null");
    expect(formatExprResult(["a", "b", "c"])).toBe("[a, b, c]");
    expect(formatExprResult(["a", "b", "c", "d"])).toBe("[4 items]");
    expect(formatExprResult({ one: 1, two: 2 })).toBe("{one, two}");
    expect(formatExprResult({ one: 1, two: 2, three: 3, four: 4 })).toBe("{4 keys}");
  });
});
