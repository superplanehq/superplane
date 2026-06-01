import { afterEach, describe, expect, it, vi } from "vitest";

import { evaluateShow, tryEvaluateShow } from "./showExpression";

describe("evaluateShow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
  });

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

  it("logs parse failures at debug level so Sentry doesn't capture authoring mistakes", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const debug = vi.spyOn(console, "debug").mockImplementation(() => {});
    evaluateShow("int(started_at) >= now() - 604800", {}, false);
    expect(warn).not.toHaveBeenCalled();
    expect(debug).toHaveBeenCalledOnce();
    expect(debug.mock.calls[0][0]).toContain("Dashboard widget expression failed");
  });
});

describe("tryEvaluateShow", () => {
  it("returns ok:true with the boolean value when the expression parses", () => {
    expect(tryEvaluateShow('row.status == "ok"', { status: "ok" })).toEqual({
      ok: true,
      value: true,
    });
  });

  it("returns ok:false with an error when the expression cannot be parsed", () => {
    const result = tryEvaluateShow("int(started_at) >= now() - 604800", {});
    expect(result.ok).toBe(false);
    if (!result.ok) {
      expect(result.error).toContain("Unexpected character '-'");
    }
  });
});
