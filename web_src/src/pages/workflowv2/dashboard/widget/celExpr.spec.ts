import { describe, it, expect } from "vitest";

import {
  buildEnv,
  compileExpr,
  compileMaybeExpr,
  compileTemplate,
  evalExpr,
  evalExprDetailed,
  evalRowField,
  evalTemplate,
  evalTemplateDetailed,
} from "./celExpr";
import { getValueAtPath } from "./fieldPath";

describe("celExpr", () => {
  it("evaluates a simple CEL expression against row fields", () => {
    const maybe = compileMaybeExpr('{{ status == "running" }}');
    expect(maybe.kind).toBe("expr");
    if (maybe.kind !== "expr") return;
    const env = buildEnv();
    const row = { status: "running" };
    expect(evalExpr(maybe.expr, row, env)).toBe(true);
  });

  it("resolves literal dot paths", () => {
    const maybe = compileMaybeExpr("pr_number");
    const env = buildEnv();
    expect(evalRowField(maybe, { pr_number: "42" }, env, getValueAtPath)).toBe("42");
  });

  it("interpolates templates", () => {
    const template = compileTemplate("Destroy PR #{{ pr_number }}?");
    const env = buildEnv();
    expect(evalTemplate(template, { pr_number: "69" }, env, String)).toBe("Destroy PR #69?");
  });

  describe("numeric-string coercion", () => {
    it("evaluates `value / 2` when value is a numeric string", () => {
      const compiled = compileExpr("value / 2");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "10" }, env)).toBe(5);
    });

    it("evaluates mixed arithmetic with stringified numbers", () => {
      const compiled = compileExpr("value * factor + offset");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "4", factor: "3", offset: "2" }, env)).toBe(14);
    });

    it("interpolates `{{ value / 2 }}` against stringified row values", () => {
      const template = compileTemplate("Half = {{ value / 2 }}");
      const env = buildEnv();
      expect(evalTemplate(template, { value: "10" }, env, String)).toBe("Half = 5");
    });

    it("returns undefined when the operand cannot be coerced to a number", () => {
      const compiled = compileExpr("value / 2");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "not a number" }, env)).toBeUndefined();
    });

    it("preserves string equality (does not coerce all strings unconditionally)", () => {
      const compiled = compileExpr('name == "42"');
      const env = buildEnv();
      expect(evalExpr(compiled, { name: "42" }, env)).toBe(true);
      expect(evalExpr(compiled, { name: "43" }, env)).toBe(false);
    });
  });

  describe("formatDate builtin", () => {
    it("formats an ISO timestamp using MM/dd in local time", () => {
      const local = new Date(2026, 2, 15, 14, 30); // March 15, 2026 (local TZ)
      const compiled = compileExpr('formatDate(createdAt, "MM/dd")');
      const env = buildEnv();
      expect(evalExpr(compiled, { createdAt: local.toISOString() }, env)).toBe("03/15");
    });

    it("supports yyyy-MM-dd HH:mm patterns", () => {
      const local = new Date(2026, 0, 5, 9, 7); // Jan 5, 2026 09:07
      const compiled = compileExpr('formatDate(ts, "yyyy-MM-dd HH:mm")');
      expect(evalExpr(compiled, { ts: local.toISOString() }, buildEnv())).toBe("2026-01-05 09:07");
    });

    it("accepts a Date instance and renders single-digit tokens unpadded", () => {
      const local = new Date(2026, 4, 3, 7, 4, 9); // May 3, 2026 07:04:09
      const compiled = compileExpr('formatDate(value, "M/d H:m:s")');
      expect(evalExpr(compiled, { value: local }, buildEnv())).toBe("5/3 7:4:9");
    });

    it("treats large numbers as epoch milliseconds", () => {
      const local = new Date(2026, 5, 1, 12, 0); // Jun 1, 2026 12:00
      const compiled = compileExpr('formatDate(ms, "yyyy")');
      expect(evalExpr(compiled, { ms: local.getTime() }, buildEnv())).toBe("2026");
    });

    it("treats small numbers as epoch seconds", () => {
      const local = new Date(2026, 6, 4, 12, 0); // Jul 4, 2026 12:00
      const seconds = Math.trunc(local.getTime() / 1000);
      const compiled = compileExpr('formatDate(sec, "MM/dd")');
      expect(evalExpr(compiled, { sec: seconds }, buildEnv())).toBe("07/04");
    });

    it("returns empty string for unparseable values and empty patterns", () => {
      const env = buildEnv();
      expect(evalExpr(compileExpr('formatDate(bad, "MM/dd")'), { bad: "not a date" }, env)).toBe("");
      expect(evalExpr(compileExpr('formatDate(value, "")'), { value: "2026-03-15T00:00:00Z" }, env)).toBe("");
      expect(evalExpr(compileExpr('formatDate(value, "MM/dd")'), { value: null }, env)).toBe("");
    });

    it("preserves non-token characters in the pattern", () => {
      const local = new Date(2026, 2, 15);
      const compiled = compileExpr('formatDate(value, "[yyyy]/[MM]")');
      expect(evalExpr(compiled, { value: local.toISOString() }, buildEnv())).toBe("[2026]/[03]");
    });
  });

  describe("error reporting", () => {
    it("reports a compile error for invalid CEL", () => {
      const compiled = compileExpr("value /");
      const result = evalExprDetailed(compiled, {}, buildEnv());
      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.error).toBeTruthy();
    });

    it("reports a type error when arithmetic cannot be coerced", () => {
      const compiled = compileExpr("value / 2");
      const result = evalExprDetailed(compiled, { value: "abc" }, buildEnv());
      expect(result.ok).toBe(false);
      if (!result.ok) expect(result.error).toMatch(/division|type/i);
    });

    it("propagates the first segment error from evalTemplateDetailed", () => {
      const template = compileTemplate("ok={{ value }} bad={{ value / }}");
      const result = evalTemplateDetailed(template, { value: 1 }, buildEnv(), String);
      expect(result.ok).toBe(false);
    });

    it("returns the rendered string when all segments succeed", () => {
      const template = compileTemplate("x={{ value / 2 }}");
      const result = evalTemplateDetailed(template, { value: "10" }, buildEnv(), String);
      expect(result.ok).toBe(true);
      if (result.ok) expect(result.value).toBe("x=5");
    });
  });
});
