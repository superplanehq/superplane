import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formatTimeAgo } from "@/lib/date";

describe("date", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-29T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("formats recent timestamps in seconds", () => {
    expect(formatTimeAgo(new Date("2026-03-29T11:59:45.000Z"))).toBe("15s ago");
  });

  it("formats older timestamps in minutes, hours, and days", () => {
    expect(formatTimeAgo(new Date("2026-03-29T11:30:00.000Z"))).toBe("30m ago");
    expect(formatTimeAgo(new Date("2026-03-29T10:00:00.000Z"))).toBe("2h ago");
    expect(formatTimeAgo(new Date("2026-03-27T12:00:00.000Z"))).toBe("2d ago");
  });

  it("omits the ago suffix when includeAgo is false", () => {
    expect(formatTimeAgo(new Date("2026-03-29T11:59:45.000Z"), false)).toBe("15s");
    expect(formatTimeAgo(new Date("2026-03-29T11:30:00.000Z"), false)).toBe("30m");
  });

  it("formats future timestamps with an in prefix instead of clamping to zero", () => {
    expect(formatTimeAgo(new Date("2026-03-29T12:00:15.000Z"))).toBe("in 15s");
    expect(formatTimeAgo(new Date("2026-03-29T12:30:00.000Z"))).toBe("in 30m");
    expect(formatTimeAgo(new Date("2026-03-29T14:00:00.000Z"))).toBe("in 2h");
    expect(formatTimeAgo(new Date("2026-03-31T12:00:00.000Z"))).toBe("in 2d");
  });

  it("keeps the in prefix for future times when includeAgo is false", () => {
    expect(formatTimeAgo(new Date("2026-03-29T12:30:00.000Z"), false)).toBe("in 30m");
    expect(formatTimeAgo(new Date("2026-03-29T14:00:00.000Z"), false)).toBe("in 2h");
  });
});
