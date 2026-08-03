import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { formatRelativeTime, formatTimestampInUserTimezone } from "@/lib/timezone";

describe("timezone", () => {
  beforeEach(() => {
    vi.useFakeTimers();
    vi.setSystemTime(new Date("2026-03-29T12:00:00.000Z"));
  });

  afterEach(() => {
    vi.useRealTimers();
  });

  it("formats timestamps in the requested timezone", () => {
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "UTC")).toBe("Mar 29, 2026, 14:30 UTC");
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "America/New_York")).toBe(
      "Mar 29, 2026, 10:30 America/New_York",
    );
  });

  it("formats timestamps for GMT offset labels", () => {
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "GMT+2")).toBe("Mar 29, 2026, 16:30 GMT+2");
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "GMT-5:30")).toBe("Mar 29, 2026, 09:00 GMT-5:30");
  });

  it("falls back to the browser timezone for unknown timezones", () => {
    const inBrowserTimezone = new Date("2026-03-29T14:30:00.000Z").toLocaleDateString("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    });

    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "Not/AZone")).toBe(
      `${inBrowserTimezone} Not/AZone`,
    );
  });

  it("formats relative time in abbreviated and long forms", () => {
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z", true)).toBe("5m ago");
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z")).toBe("5 minutes ago");
    expect(formatRelativeTime(undefined)).toBe("N/A");
  });
});
