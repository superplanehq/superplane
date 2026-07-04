import { describe, expect, it } from "vitest";
import { evaluateExpr, formatExprResult } from "@/lib/exprEvaluator";

describe("exprEvaluator", () => {
  it("evaluates expression functions and field access", () => {
    expect(evaluateExpr('upper("hello")', {})).toBe("HELLO");
    expect(evaluateExpr('$["node"].data.name', { node: { data: { name: "test" } } })).toBe("test");
  });

  it("evaluates root and previous context helpers", () => {
    expect(evaluateExpr("root().data.name", { __root: { data: { name: "DCO" } } })).toBe("DCO");
    expect(evaluateExpr("previous().data.status", { __previousByDepth: { "1": { data: { status: "passed" } } } })).toBe(
      "passed",
    );
  });

  it("evaluates expr-compatible string and array slices", () => {
    expect(evaluateExpr("root().data.sha[:7]", { __root: { data: { sha: "d6f3c8a2e8b7" } } })).toBe("d6f3c8a");
    expect(evaluateExpr("root().data.sha[2:7]", { __root: { data: { sha: "d6f3c8a2e8b7" } } })).toBe("f3c8a");
    expect(evaluateExpr("root().data.items[1:]", { __root: { data: { items: ["a", "b", "c"] } } })).toEqual(["b", "c"]);
  });

  it("formats timestamp strings through the date helper", () => {
    expect(
      evaluateExpr('date(root().timestamp).Format("2006-01-02 15:04:05")', {
        __root: { timestamp: "2024-01-01T09:08:07Z" },
      }),
    ).toBe("2024-01-01 09:08:07");
  });

  it("formats timestamp strings with date methods", () => {
    expect(
      evaluateExpr('root().timestamp.Format("2006-01-02 15:04:05")', {
        __root: { timestamp: "2024-01-01T09:08:07Z" },
      }),
    ).toBe("2024-01-01 09:08:07");
  });

  it("formats timestamp strings in UTC", () => {
    expect(evaluateExpr('date("2024-01-01T00:00:00Z").Format("2006-01-02 15:04:05")', {})).toBe("2024-01-01 00:00:00");
    expect(evaluateExpr('date("2024-01-01T01:00:00+01:00").Format("2006-01-02 15:04:05")', {})).toBe(
      "2024-01-01 00:00:00",
    );
  });

  it("formats primitive, array, and object results for display", () => {
    expect(formatExprResult(null)).toBe("null");
    expect(formatExprResult(["a", "b", "c"])).toBe("[a, b, c]");
    expect(formatExprResult(["a", "b", "c", "d"])).toBe("[4 items]");
    expect(formatExprResult({ one: 1, two: 2 })).toBe("{one, two}");
    expect(formatExprResult({ one: 1, two: 2, three: 3, four: 4 })).toBe("{4 keys}");
  });
});
