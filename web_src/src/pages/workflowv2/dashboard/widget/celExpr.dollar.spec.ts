import { describe, it, expect } from "vitest";

import {
  buildEnv,
  compileExpr,
  compileMaybeExpr,
  compileTemplate,
  DOLLAR_REWRITE_IDENTIFIER,
  evalExpr,
  evalRowField,
  evalTemplate,
  rewriteDollarRefs,
} from "./celExpr";
import { getValueAtPath } from "./fieldPath";

describe("rewriteDollarRefs", () => {
  it("rewrites bare `$` to the safe identifier", () => {
    expect(rewriteDollarRefs('$["x"].y')).toBe(`${DOLLAR_REWRITE_IDENTIFIER}["x"].y`);
  });

  it("rewrites multiple `$` occurrences in the same expression", () => {
    const out = rewriteDollarRefs('$["a"].x + $["b"].y');
    expect(out).toBe(`${DOLLAR_REWRITE_IDENTIFIER}["a"].x + ${DOLLAR_REWRITE_IDENTIFIER}["b"].y`);
  });

  it("leaves `$` inside double-quoted strings alone", () => {
    expect(rewriteDollarRefs('"price: $5"')).toBe('"price: $5"');
  });

  it("leaves `$` inside single-quoted strings alone", () => {
    expect(rewriteDollarRefs("'price: $5'")).toBe("'price: $5'");
  });

  it("respects backslash escapes inside string literals", () => {
    // The escaped quote shouldn't end the string, so the trailing `$` stays inside.
    expect(rewriteDollarRefs('"foo\\" $bar"')).toBe('"foo\\" $bar"');
  });

  it("rewrites a `$` that appears between two string literals", () => {
    const out = rewriteDollarRefs('"a" + $["x"] + "b"');
    expect(out).toBe(`"a" + ${DOLLAR_REWRITE_IDENTIFIER}["x"] + "b"`);
  });

  it("is a no-op for expressions without `$`", () => {
    expect(rewriteDollarRefs("pr_number / 2")).toBe("pr_number / 2");
  });
});

describe("CEL with `$` node refs", () => {
  it("compiles and evaluates `$['name'].outputs.<key>` against the row", () => {
    const compiled = compileExpr('$["deploy"].outputs.url');
    expect(compiled.ok).toBe(true);
    const env = buildEnv();
    const row = {
      [DOLLAR_REWRITE_IDENTIFIER]: {
        deploy: { outputs: { url: "https://example.com" } },
      },
    } as Record<string, unknown>;
    expect(evalExpr(compiled, row, env)).toBe("https://example.com");
  });

  it("evaluates `$['name'].data.<key>` (the canvas-style shortcut)", () => {
    const compiled = compileExpr('$["build"].data.commit');
    const env = buildEnv();
    const row = {
      [DOLLAR_REWRITE_IDENTIFIER]: {
        build: { data: { commit: "abc123" } },
      },
    } as Record<string, unknown>;
    expect(evalExpr(compiled, row, env)).toBe("abc123");
  });

  it("works inside a `{{ }}` full expression via compileMaybeExpr", () => {
    const maybe = compileMaybeExpr('{{ $["a"].outputs.score > 50 }}');
    expect(maybe.kind).toBe("expr");
    if (maybe.kind !== "expr") return;
    const env = buildEnv();
    const row = {
      [DOLLAR_REWRITE_IDENTIFIER]: { a: { outputs: { score: 80 } } },
    } as Record<string, unknown>;
    expect(evalExpr(maybe.expr, row, env)).toBe(true);
  });

  it("works inside a partial `{{ }}` template", () => {
    const template = compileTemplate('Deploy URL: {{ $["deploy"].outputs.url }}');
    const env = buildEnv();
    const row = {
      [DOLLAR_REWRITE_IDENTIFIER]: { deploy: { outputs: { url: "https://x.test" } } },
    } as Record<string, unknown>;
    expect(evalTemplate(template, row, env, String)).toBe("Deploy URL: https://x.test");
  });

  it("resolves literal dot paths via the row's `$` key", () => {
    // Literal mode (no `{{ }}`) goes through getValueAtPath/parsePath, which
    // already handles `$["name"]` because `$` is a normal identifier char
    // there. The row carries `$` directly (alongside the rewrite alias).
    const maybe = compileMaybeExpr('$["deploy"].outputs.url');
    const env = buildEnv();
    const row = {
      $: { deploy: { outputs: { url: "https://literal.test" } } },
      [DOLLAR_REWRITE_IDENTIFIER]: { deploy: { outputs: { url: "https://literal.test" } } },
    } as Record<string, unknown>;
    expect(evalRowField(maybe, row, env, getValueAtPath)).toBe("https://literal.test");
  });
});
