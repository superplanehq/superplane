import { describe, it, expect } from "vitest";

import { buildEnv, compileMaybeExpr, compileTemplate, evalExpr, evalRowField, evalTemplate } from "./celExpr";
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
});
