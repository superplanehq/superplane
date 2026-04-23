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

  describe("filePathMatches", () => {
    const commits = [
      { added: ["pkg/integrations/github/on_push.go"], modified: [], removed: [] },
      { added: [], modified: ["web_src/src/App.tsx"], removed: ["docs/old.md"] },
    ];

    it("returns true when a modified file matches the pattern", () => {
      expect(evaluateExpr("filePathMatches(commits, \"web_src/**\")", { commits })).toBe(true);
    });

    it("returns true when an added file matches the pattern", () => {
      expect(evaluateExpr("filePathMatches(commits, \"pkg/**\")", { commits })).toBe(true);
    });

    it("returns true when a removed file matches the pattern", () => {
      expect(evaluateExpr("filePathMatches(commits, \"docs/**\")", { commits })).toBe(true);
    });

    it("returns false when no file matches the pattern", () => {
      expect(evaluateExpr("filePathMatches(commits, \"migrations/**\")", { commits })).toBe(false);
    });

    it("supports single-segment wildcard", () => {
      expect(evaluateExpr("filePathMatches(commits, \"pkg/integrations/*\")", { commits })).toBe(false);
      expect(evaluateExpr("filePathMatches(commits, \"pkg/integrations/github/*\")", { commits })).toBe(true);
    });

    it("returns false for empty commits", () => {
      expect(evaluateExpr("filePathMatches(commits, \"pkg/**\")", { commits: [] })).toBe(false);
    });

    it("supports exact match pattern", () => {
      expect(evaluateExpr("filePathMatches(commits, \"docs/old.md\")", { commits })).toBe(true);
      expect(evaluateExpr("filePathMatches(commits, \"docs/new.md\")", { commits })).toBe(false);
    });
  });
});
