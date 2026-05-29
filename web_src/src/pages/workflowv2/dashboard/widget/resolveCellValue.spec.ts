import { describe, expect, it } from "vitest";

import { buildEnv } from "./celExpr";
import { compileFieldResolver, resolveCellValue } from "./resolveCellValue";

describe("compileFieldResolver", () => {
  it("resolves a literal dot path against many rows", () => {
    const resolver = compileFieldResolver("nested.value");
    expect(resolver.resolve({ nested: { value: "a" } })).toBe("a");
    expect(resolver.resolve({ nested: { value: 42 } })).toBe(42);
  });

  it("evaluates a CEL expression once per row when the field is `{{ expr }}`", () => {
    const resolver = compileFieldResolver("{{ upper(name) }}");
    expect(resolver.resolve({ name: "passed" })).toBe("PASSED");
    expect(resolver.resolve({ name: "failed" })).toBe("FAILED");
  });

  it("returns undefined for an empty field reference without throwing", () => {
    const resolver = compileFieldResolver("");
    expect(resolver.resolve({ value: 1 })).toBeUndefined();
  });

  it("shares the passed env across many resolvers so they observe the same `now()`", () => {
    const env = buildEnv();
    const a = compileFieldResolver("{{ now }}", env);
    const b = compileFieldResolver("{{ now }}", env);
    expect(a.resolve({})).toBe(b.resolve({}));
  });

  it("treats non-record rows as empty so authors don't crash on stray scalars", () => {
    const resolver = compileFieldResolver("value");
    expect(resolver.resolve(null)).toBeUndefined();
    expect(resolver.resolve(42)).toBeUndefined();
    expect(resolver.resolve(["a"])).toBeUndefined();
  });
});

describe("resolveCellValue", () => {
  it("resolves a single value using the same rules as the compiled resolver", () => {
    expect(resolveCellValue("status", { status: "passed" })).toBe("passed");
    expect(resolveCellValue("{{ lower(status) }}", { status: "PASSED" })).toBe("passed");
  });
});
