import { describe, expect, it, vi } from "vitest";

import { evaluateRowShow } from "./rowVisibility";

describe("evaluateRowShow", () => {
  it("evaluates CEL templates wrapped in braces", () => {
    expect(evaluateRowShow('{{ status == "failed" }}', { status: "failed" })).toBe(true);
    expect(evaluateRowShow('{{ status == "failed" }}', { status: "passed" })).toBe(false);
  });

  it('evaluates simple legacy `field == "value"` expressions', () => {
    expect(evaluateRowShow('status == "failed"', { status: "failed" })).toBe(true);
    expect(evaluateRowShow('status == "failed"', { status: "passed" })).toBe(false);
  });

  it("keeps mini-expression semantics for `row.`-prefixed and bare fields", () => {
    expect(evaluateRowShow("row.count > 5", { count: 10 })).toBe(true);
    expect(evaluateRowShow("row.count > 5", { count: 1 })).toBe(false);
    expect(evaluateRowShow('row.status == "failed" || row.status == "running"', { status: "running" })).toBe(true);
  });

  it("evaluates bare CEL expressions with arithmetic and builtins", () => {
    // Regression for #6233: the mini tokenizer lacks `-`/`*` and function
    // calls, so this must fall back to CEL instead of throwing.
    const recent = { createdAt: new Date().toISOString() };
    const old = { createdAt: new Date(Date.now() - 30 * 24 * 3600 * 1000).toISOString() };
    const expr = "epochMs(createdAt) > (float(now) - 604800.0) * 1000.0";
    expect(evaluateRowShow(expr, recent)).toBe(true);
    expect(evaluateRowShow(expr, old)).toBe(false);
  });

  it("does not warn when a bare CEL expression resolves cleanly", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    evaluateRowShow("epochMs(createdAt) > (float(now) - 604800.0) * 1000.0", {
      createdAt: new Date().toISOString(),
    });
    expect(warn).not.toHaveBeenCalled();
    warn.mockRestore();
  });

  it("returns the default and warns once when nothing can parse the expression", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    expect(evaluateRowShow("totally bogus $$$ ===", {}, false)).toBe(false);
    expect(warn).toHaveBeenCalledTimes(1);
    warn.mockRestore();
  });

  it("returns the default value for empty expressions", () => {
    expect(evaluateRowShow(undefined, {})).toBe(true);
    expect(evaluateRowShow("   ", {}, false)).toBe(false);
  });
});
