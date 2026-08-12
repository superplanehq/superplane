import { describe, expect, it } from "vitest";
import { createElement } from "react";
import { resolveFactoryNodeMetrics } from "./resolveFactoryNodeMetrics";

describe("resolveFactoryNodeMetrics", () => {
  it("keeps ReactNode eventSubtitle (TimeAgo from mappers)", () => {
    const node = createElement("span", null, "3m ago");
    expect(
      resolveFactoryNodeMetrics({
        status: "passed",
        section: { eventSubtitle: node },
        liveDurationMs: null,
      }),
    ).toBe(node);
  });

  it("uses string eventSubtitle when present", () => {
    expect(
      resolveFactoryNodeMetrics({
        status: "passed",
        section: { eventSubtitle: "  7s  " },
        liveDurationMs: null,
      }),
    ).toBe("7s");
  });

  it("prefers live duration when showAutomaticTime is set", () => {
    expect(
      resolveFactoryNodeMetrics({
        status: "running",
        section: { eventSubtitle: "stale", showAutomaticTime: true },
        liveDurationMs: 9 * 60_000,
      }),
    ).toBe("9m so far");
  });

  it("falls back to Timestamp from receivedAt when subtitle missing", () => {
    const receivedAt = new Date("2026-01-01T12:00:00Z");
    const result = resolveFactoryNodeMetrics({
      status: "triggered",
      section: { receivedAt },
      liveDurationMs: null,
    });
    expect(result).toBeTruthy();
    expect(typeof result).not.toBe("string");
  });

  it("returns null when nothing to show", () => {
    expect(
      resolveFactoryNodeMetrics({
        status: "pending",
        section: undefined,
        liveDurationMs: null,
      }),
    ).toBeNull();
  });
});
