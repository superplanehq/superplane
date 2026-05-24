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
});
