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

  describe("epochMs builtin", () => {
    it("converts an ISO-8601 string to ms-since-epoch", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      const result = evalExpr(compiled, { value: "2026-01-01T00:00:00Z" }, env) as number;
      expect(result).toBe(Date.UTC(2026, 0, 1, 0, 0, 0));
    });

    it("supports timestamp arithmetic across two ISO strings", () => {
      // Authors hit this when they write `{{ epochMs(finishedAt) - epochMs(createdAt) }}`
      // on a runs row to compute elapsed time without the durationMs convenience.
      const compiled = compileExpr("epochMs(finishedAt) - epochMs(createdAt)");
      const env = buildEnv();
      const result = evalExpr(compiled, { createdAt: "2026-01-01T12:00:00Z", finishedAt: "2026-01-01T12:05:00Z" }, env);
      expect(result).toBe(5 * 60 * 1000);
    });

    it("composes with `duration()` for human-friendly output", () => {
      const template = compileTemplate("Took {{ duration((epochMs(finishedAt) - epochMs(createdAt)) / 1000) }}");
      const env = buildEnv();
      const result = evalTemplate(
        template,
        { createdAt: "2026-01-01T12:00:00Z", finishedAt: "2026-01-01T12:05:00Z" },
        env,
        String,
      );
      expect(result).toBe("Took 5m");
    });

    it("returns 0 for unparseable inputs so arithmetic stays defined", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      expect(evalExpr(compiled, { value: "not a date" }, env)).toBe(0);
      expect(evalExpr(compiled, { value: null }, env)).toBe(0);
    });

    it("accepts epoch numbers (seconds and ms) and Date instances", () => {
      const compiled = compileExpr("epochMs(value)");
      const env = buildEnv();
      const ms = Date.UTC(2026, 5, 1, 12, 0);
      expect(evalExpr(compiled, { value: ms }, env)).toBe(ms);
      expect(evalExpr(compiled, { value: ms / 1000 }, env)).toBe(ms);
      expect(evalExpr(compiled, { value: new Date(ms) }, env)).toBe(ms);
    });
  });

  describe("parseJson builtin", () => {
    // cel-js's grammar does not allow postfix `.foo` / `[i]` / `.method(...)`
    // after a function call result. So `parseJson(blob).items` is a parse
    // error. The builtin is still useful when composed with other functions
    // (`size(...)`, `string(...)`, equality), or when the whole expression
    // is `parseJson(value)` and the renderer consumes the structured result.
    it("parses a JSON array string and returns it wholesale", () => {
      const compiled = compileExpr("parseJson(tags)");
      expect(evalExpr(compiled, { tags: '["a","b"]' }, buildEnv())).toEqual(["a", "b"]);
    });

    it("parses a JSON object string and returns it wholesale", () => {
      const compiled = compileExpr("parseJson(blob)");
      const result = evalExpr(compiled, { blob: '{"items":[{"id":1}]}' }, buildEnv());
      expect(result).toEqual({ items: [{ id: 1 }] });
    });

    it("composes with size() to count parsed elements", () => {
      const compiled = compileExpr("size(parseJson(tags))");
      expect(evalExpr(compiled, { tags: '["a","b","c"]' }, buildEnv())).toBe(3);
    });

    it("passes already-parsed values through unchanged", () => {
      const compiled = compileExpr("size(parseJson(tags))");
      expect(evalExpr(compiled, { tags: ["a", "b", "c"] }, buildEnv())).toBe(3);
    });

    it("returns null for malformed JSON so equality checks stay defined", () => {
      const compiled = compileExpr("parseJson(bad) == null");
      expect(evalExpr(compiled, { bad: "not json" }, buildEnv())).toBe(true);
    });

    it("returns null for null inputs without throwing", () => {
      const compiled = compileExpr("parseJson(value)");
      expect(evalExpr(compiled, { value: null }, buildEnv())).toBeNull();
    });

    it("works inside templated interpolation when the whole expression is parseJson", () => {
      const template = compileTemplate("Tags: {{ parseJson(blob) }}");
      const result = evalTemplate(template, { blob: '["a","b"]' }, buildEnv(), String);
      // Template stringify uses String() on the parsed value; nested objects
      // serialize via JS's default Array#toString. Authors who need pretty
      // formatting should compose `string()` or shape the data upstream.
      expect(result).toBe("Tags: a,b");
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
