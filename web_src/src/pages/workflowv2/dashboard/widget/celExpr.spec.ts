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
