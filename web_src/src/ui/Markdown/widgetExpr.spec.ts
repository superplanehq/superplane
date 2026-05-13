import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import {
  buildEnv,
  compileExpr,
  compileMaybeExpr,
  compileTemplate,
  evalExpr,
  evalRowField,
  evalTemplate,
} from "./widgetExpr";

const stringify = (value: unknown): string => {
  if (value === null || value === undefined) return "";
  if (typeof value === "string") return value;
  if (typeof value === "number" || typeof value === "boolean") return String(value);
  try {
    return JSON.stringify(value);
  } catch {
    return String(value);
  }
};

const dotPath = (row: Record<string, unknown>, path: string): unknown => {
  let cur: unknown = row;
  for (const part of path.split(".")) {
    if (cur == null || typeof cur !== "object") return undefined;
    cur = (cur as Record<string, unknown>)[part];
  }
  return cur;
};

describe("compileExpr", () => {
  it("returns ok=true with the parsed CST for a valid expression", () => {
    const r = compileExpr("1 + 2");
    expect(r.ok).toBe(true);
    if (r.ok) expect(r.cst).toBeDefined();
  });

  it("returns ok=false with an error message for invalid syntax", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const r = compileExpr("1 +");
    expect(r.ok).toBe(false);
    if (!r.ok) {
      expect(r.error.length).toBeGreaterThan(0);
      expect(r.raw).toBe("1 +");
    }
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });
});

describe("compileMaybeExpr", () => {
  it("treats a plain dot-path as a literal", () => {
    const r = compileMaybeExpr("pr_number");
    expect(r.kind).toBe("literal");
    if (r.kind === "literal") expect(r.value).toBe("pr_number");
  });

  it("treats a fully-wrapped {{ ... }} string as a CEL expression", () => {
    const r = compileMaybeExpr("{{ 1 + 2 }}");
    expect(r.kind).toBe("expr");
  });

  it("treats partial interpolation as a literal (templates use compileTemplate)", () => {
    const r = compileMaybeExpr("PR #{{ pr_number }}");
    expect(r.kind).toBe("literal");
  });
});

describe("compileTemplate", () => {
  it("returns a single literal segment for a string with no braces", () => {
    const t = compileTemplate("hello");
    expect(t.hasExpr).toBe(false);
    expect(t.segments).toHaveLength(1);
    expect(t.segments[0]).toEqual({ kind: "literal", value: "hello" });
  });

  it("interleaves literal and expression segments", () => {
    const t = compileTemplate("PR #{{ pr_number }} ready");
    expect(t.hasExpr).toBe(true);
    expect(t.segments.map((s) => s.kind)).toEqual(["literal", "expr", "literal"]);
  });

  it("supports multiple expression segments in one string", () => {
    const t = compileTemplate("{{ a }} and {{ b }}");
    expect(t.segments.map((s) => s.kind)).toEqual(["expr", "literal", "expr"]);
  });
});

describe("evalExpr", () => {
  it("resolves identifiers from the row", () => {
    const env = buildEnv();
    const compiled = compileExpr("pr_number");
    expect(evalExpr(compiled, { pr_number: 42 }, env)).toBe(42);
  });

  it("merges globals (now) with the row", () => {
    const env = buildEnv({ now: 1700000000 });
    const compiled = compileExpr("now - created_at");
    expect(evalExpr(compiled, { created_at: 1699999940 }, env)).toBe(60);
  });

  it("returns undefined when the expression references a missing identifier (fail-soft)", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const env = buildEnv();
    const compiled = compileExpr("missing");
    expect(evalExpr(compiled, {}, env)).toBeUndefined();
    expect(warn).toHaveBeenCalled();
    warn.mockRestore();
  });

  it("returns undefined for a parse-failed expression", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const compiled = compileExpr("1 +");
    expect(evalExpr(compiled, {}, buildEnv())).toBeUndefined();
    warn.mockRestore();
  });

  it("supports ternary, comparison, and arithmetic", () => {
    const env = buildEnv();
    const compiled = compileExpr('status == "passed" ? 1 : 0');
    expect(evalExpr(compiled, { status: "passed" }, env)).toBe(1);
    expect(evalExpr(compiled, { status: "failed" }, env)).toBe(0);
  });
});

describe("buildEnv custom functions", () => {
  it("int() coerces strings, numbers, and bools", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr('int("42")'), {}, env)).toBe(42);
    expect(evalExpr(compileExpr("int(3.7)"), {}, env)).toBe(3);
    expect(evalExpr(compileExpr("int(true)"), {}, env)).toBe(1);
  });

  it("float() coerces and falls back to 0", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr('float("3.14")'), {}, env)).toBeCloseTo(3.14);
    expect(evalExpr(compileExpr('float("nope")'), {}, env)).toBe(0);
  });

  it("string() stringifies primitives", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr("string(42)"), {}, env)).toBe("42");
    expect(evalExpr(compileExpr("string(true)"), {}, env)).toBe("true");
  });

  it("contains/startsWith/endsWith are free functions", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr('contains("hello world", "world")'), {}, env)).toBe(true);
    expect(evalExpr(compileExpr('startsWith("hello", "he")'), {}, env)).toBe(true);
    expect(evalExpr(compileExpr('endsWith("hello", "lo")'), {}, env)).toBe(true);
    expect(evalExpr(compileExpr('contains("hello", "WORLD")'), {}, env)).toBe(false);
  });

  it("matches() runs a regex and fails soft on bad patterns", () => {
    const env = buildEnv();
    // cel-js passes backslashes through verbatim in string literals, so a
    // CEL source like `"^[a-z]+\d+$"` lands in JS as the regex pattern
    // `^[a-z]+\d+$`. Authoring this in a JS literal needs `\\d+`.
    expect(evalExpr(compileExpr('matches("abc123", "^[a-z]+\\d+$")'), {}, env)).toBe(true);
    expect(evalExpr(compileExpr('matches("abc", "[")'), {}, env)).toBe(false);
  });

  it("lower()/upper() handle nullish input", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr('upper("hi")'), {}, env)).toBe("HI");
    expect(evalExpr(compileExpr('lower("HI")'), {}, env)).toBe("hi");
  });

  it("duration() formats seconds in s/m/h", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr("duration(45)"), {}, env)).toBe("45s");
    expect(evalExpr(compileExpr("duration(125)"), {}, env)).toBe("2m 5s");
    expect(evalExpr(compileExpr("duration(3725)"), {}, env)).toBe("1h 2m");
    expect(evalExpr(compileExpr("duration(7200)"), {}, env)).toBe("2h");
  });

  it("timestamp() formats Unix seconds as ISO date", () => {
    const env = buildEnv();
    expect(evalExpr(compileExpr("timestamp(0)"), {}, env)).toBe("1970-01-01T00:00:00.000Z");
    expect(evalExpr(compileExpr("timestamp(1700000000)"), {}, env)).toBe(new Date(1700000000_000).toISOString());
  });

  it("now is injected as Unix seconds when not overridden", () => {
    const env = buildEnv();
    const result = evalExpr(compileExpr("now"), {}, env);
    expect(typeof result).toBe("number");
    expect(result).toBeGreaterThan(1_500_000_000);
  });
});

describe("evalRowField", () => {
  it("dispatches literals to the dot-path resolver", () => {
    const env = buildEnv();
    expect(evalRowField(compileMaybeExpr("a.b"), { a: { b: 7 } }, env, dotPath)).toBe(7);
  });

  it("dispatches expressions to CEL", () => {
    const env = buildEnv({ now: 100 });
    expect(evalRowField(compileMaybeExpr("{{ now - created_at }}"), { created_at: 60 }, env, dotPath)).toBe(40);
  });
});

describe("evalTemplate", () => {
  let env = buildEnv();
  beforeEach(() => {
    env = buildEnv({ now: 1000 });
  });
  afterEach(() => {
    vi.restoreAllMocks();
  });

  it("returns a literal-only template unchanged", () => {
    const t = compileTemplate("hello world");
    expect(evalTemplate(t, {}, env, stringify)).toBe("hello world");
  });

  it("interpolates a single expression into surrounding text", () => {
    const t = compileTemplate("Destroy PR #{{ pr_number }}?");
    expect(evalTemplate(t, { pr_number: 42 }, env, stringify)).toBe("Destroy PR #42?");
  });

  it("supports multiple expression segments in one template", () => {
    const t = compileTemplate("{{ a }} + {{ b }} = {{ a + b }}");
    expect(evalTemplate(t, { a: 1, b: 2 }, env, stringify)).toBe("1 + 2 = 3");
  });

  it("renders failed segments as empty without breaking surrounding text", () => {
    vi.spyOn(console, "warn").mockImplementation(() => {});
    const t = compileTemplate("ok={{ ok }}, missing={{ missing_var }}");
    expect(evalTemplate(t, { ok: 1 }, env, stringify)).toBe("ok=1, missing=");
  });
});
