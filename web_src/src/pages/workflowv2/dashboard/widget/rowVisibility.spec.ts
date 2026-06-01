import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

import { evaluateRowShow } from "./rowVisibility";

describe("evaluateRowShow", () => {
  afterEach(() => {
    vi.restoreAllMocks();
    vi.useRealTimers();
  });

  it("dispatches to the mini-eval for `row.foo` style comparisons it can parse", () => {
    expect(evaluateRowShow("row.count > 5", { count: 10 })).toBe(true);
    expect(evaluateRowShow("row.count > 5", { count: 1 })).toBe(false);
    expect(evaluateRowShow('(row.status == "failed") && !row.retried', { status: "failed", retried: false })).toBe(
      true,
    );
  });

  it('handles the legacy `field == "value"` shorthand', () => {
    expect(evaluateRowShow('status == "running"', { status: "running" })).toBe(true);
    expect(evaluateRowShow('status == "running"', { status: "passed" })).toBe(false);
  });

  it("evaluates CEL templates wrapped in `{{ … }}`", () => {
    expect(evaluateRowShow('{{ status == "failed" }}', { status: "failed" })).toBe(true);
  });

  it("falls back to CEL for bare expressions with function calls and arithmetic", () => {
    const nowEpoch = Math.floor(Date.now() / 1000);
    const recentEpoch = nowEpoch - 60;
    const oldEpoch = nowEpoch - 60 * 60 * 24 * 14;

    expect(evaluateRowShow("int(started_at) >= now() - 604800", { started_at: recentEpoch }, false)).toBe(true);
    expect(evaluateRowShow("int(started_at) >= now() - 604800", { started_at: oldEpoch }, false)).toBe(false);
    expect(evaluateRowShow("int(started_at) >= now - 604800", { started_at: recentEpoch }, false)).toBe(true);
  });

  it("returns the default and logs at debug level when neither evaluator can parse", () => {
    const warn = vi.spyOn(console, "warn").mockImplementation(() => {});
    const debug = vi.spyOn(console, "debug").mockImplementation(() => {});
    expect(evaluateRowShow("totally :: bogus ## expression", {}, false)).toBe(false);
    expect(warn).not.toHaveBeenCalled();
    expect(debug).toHaveBeenCalled();
  });

  it("returns the default for empty or whitespace expressions", () => {
    expect(evaluateRowShow("", {}, true)).toBe(true);
    expect(evaluateRowShow("   ", {}, false)).toBe(false);
    expect(evaluateRowShow(undefined, {}, true)).toBe(true);
  });

  beforeEach(() => {
    // ensure each test starts fresh in case a previous test mocked the clock
    vi.useRealTimers();
  });
});
