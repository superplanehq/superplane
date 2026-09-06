import { afterEach, describe, expect, it, vi } from "vitest";

import { evaluateShow } from "./showExpression";

// Security property: `evaluateShow` never routes the expression through
// `eval` / `new Function`. Clauses are interpreted by the CEL engine
// (`@marcbachmann/cel-js` via `celExpr.ts`), which walks its own AST, so
// widgets imported from untrusted YAML cannot execute arbitrary JS — see
// the "sandbox escape" test below.
//
// Note on `defaultValue` in the newer tests: we deliberately pass the
// OPPOSITE of the expected result so a silent fail-soft fallback can never
// masquerade as a correct evaluation.

describe("evaluateShow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

  describe("legacy expression forms", () => {
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

    it("supports single-quoted string literals", () => {
      expect(evaluateShow("status == 'ok'", { status: "ok" }, false)).toBe(true);
      expect(evaluateShow("status == 'ok'", { status: "bad" }, true)).toBe(false);
    });

    it("compares against the null literal", () => {
      expect(evaluateShow("value == null", { value: null }, false)).toBe(true);
      expect(evaluateShow("value == null", { value: "x" }, true)).toBe(false);
    });

    it("supports the <=, >=, and != operators", () => {
      expect(evaluateShow("count <= 5", { count: 5 }, false)).toBe(true);
      expect(evaluateShow("count >= 5", { count: 4 }, true)).toBe(false);
      expect(evaluateShow('status != "failed"', { status: "passed" }, false)).toBe(true);
    });

    it("resolves nested field paths", () => {
      expect(evaluateShow("a.b.c > 1", { a: { b: { c: 5 } } }, false)).toBe(true);
    });

    it("treats root identifiers missing from the row as null, not as errors", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      // Legacy resolved unknown fields to `undefined`, making comparisons
      // false; CEL alone would raise "Unknown variable". The adapter binds
      // unknown roots to `null` so heterogeneous rows keep filtering.
      expect(evaluateShow('status == "deleted"', {}, true)).toBe(false);
      expect(evaluateShow('status != "deleted"', {}, false)).toBe(true);
      expect(warn).not.toHaveBeenCalled();
    });

    it("coerces numeric strings for ordering comparisons (stringified memory rows)", () => {
      expect(evaluateShow("count > 5", { count: "10" }, false)).toBe(true);
      expect(evaluateShow("count > 5", { count: "3" }, true)).toBe(false);
    });

    it("returns the truthiness of a bare field reference", () => {
      expect(evaluateShow("count", { count: 3 }, false)).toBe(true);
      expect(evaluateShow("name", { name: "" }, true)).toBe(false);
    });
  });

  describe("CEL surface (issue #6232)", () => {
    it("evaluates the reported expression: epochMs(createdAt) > (now - 604800) * 1000", () => {
      const expr = "epochMs(createdAt) > (now - 604800) * 1000";
      const recent = new Date().toISOString();
      expect(evaluateShow(expr, { createdAt: recent }, false)).toBe(true);
      expect(evaluateShow(expr, { createdAt: "2020-01-01T00:00:00Z" }, true)).toBe(false);
    });

    it("supports arithmetic with + - * / and parentheses", () => {
      expect(evaluateShow("(a + b) / 2 >= 5", { a: 4, b: 6 }, false)).toBe(true);
      expect(evaluateShow("(a + b) / 2 >= 5", { a: 1, b: 2 }, true)).toBe(false);
      expect(evaluateShow("a - b * 2 == -8", { a: 2, b: 5 }, false)).toBe(true);
    });

    it("supports builtin function calls", () => {
      expect(evaluateShow('lower(status) == "ok"', { status: "OK" }, false)).toBe(true);
      expect(evaluateShow('contains(name, "prod")', { name: "prod-eu-1" }, false)).toBe(true);
      expect(evaluateShow('contains(name, "prod")', { name: "staging" }, true)).toBe(false);
    });

    it("supports the in operator and ternaries", () => {
      expect(evaluateShow('status in ["ok", "good"]', { status: "good" }, false)).toBe(true);
      expect(evaluateShow('status == "ok" ? count > 1 : false', { status: "ok", count: 2 }, false)).toBe(true);
    });
  });

  describe("fail-soft behavior", () => {
    it("returns defaultValue and warns on syntax errors, never throws", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      expect(evaluateShow("epochMs(", {}, true)).toBe(true);
      expect(evaluateShow("epochMs(", {}, false)).toBe(false);
      expect(evaluateShow("a &&& b", { a: true, b: true })).toBe(true);
      expect(warn).toHaveBeenCalled();
    });

    it("returns defaultValue and warns on runtime eval errors", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      // Nested key missing below the root: CEL raises "No such key"; the
      // adapter falls back to defaultValue instead of hiding/throwing.
      expect(evaluateShow('a.b == "x"', { a: {} }, true)).toBe(true);
      expect(evaluateShow('a.b == "x"', { a: {} }, false)).toBe(false);
      expect(warn).toHaveBeenCalled();
    });

    it("returns defaultValue for empty or blank expressions without warning", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      expect(evaluateShow(undefined, {}, true)).toBe(true);
      expect(evaluateShow("", {}, false)).toBe(false);
      expect(evaluateShow("   ", {}, true)).toBe(true);
      expect(warn).not.toHaveBeenCalled();
    });
  });

  describe("sandbox", () => {
    it("cannot escape into JS via constructor chains", () => {
      const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
      const escape = '("").constructor.constructor("globalThis.__pwned = true")()';
      expect(evaluateShow(escape, {}, false)).toBe(false);
      expect((globalThis as Record<string, unknown>).__pwned).toBeUndefined();
      expect(warn).toHaveBeenCalled();
    });
  });

  describe("documented divergences from the legacy evaluator", () => {
    it("cross-type equality follows CEL (typed) semantics", () => {
      // The legacy evaluator loosely normalized scalars, so `10 == "10"`
      // and `true == "true"` were true. CEL equality is typed: values of
      // different types are simply not equal. Same-type comparisons — the
      // documented authoring forms — are unaffected.
      expect(evaluateShow('count == "10"', { count: 10 }, true)).toBe(false);
      expect(evaluateShow("flag == true", { flag: "true" }, true)).toBe(false);
      // Same-type equality on stringified rows still works.
      expect(evaluateShow('count == "10"', { count: "10" }, false)).toBe(true);
    });
  });
});
