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

  it("renders the timestamp in the requested timezone, not just labels it", () => {
    // 14:30 UTC is 10:30 in New York and 23:30 in Tokyo on this date.
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "UTC")).toBe("Mar 29, 2026, 14:30 UTC");
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "America/New_York")).toBe(
      "Mar 29, 2026, 10:30 America/New_York",
    );
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "Asia/Tokyo")).toBe(
      "Mar 29, 2026, 23:30 Asia/Tokyo",
    );
  });

  it("accepts a Date instance as well as an ISO string", () => {
    expect(formatTimestampInUserTimezone(new Date("2026-03-29T14:30:00.000Z"), "UTC")).toBe("Mar 29, 2026, 14:30 UTC");
  });

  it("defaults to the browser timezone when none is given", () => {
    const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z")).toBe(
      formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", browserTimezone),
    );
  });

  it("falls back to the browser timezone when the timezone is not a valid IANA name", () => {
    const browserTimezone = Intl.DateTimeFormat().resolvedOptions().timeZone;

    // The backend stores some timezones as numeric offsets, which Intl rejects.
    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "-5")).toBe(
      formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", browserTimezone),
    );
  });

  it("formats relative time in abbreviated and long forms", () => {
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z", true)).toBe("5m ago");
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z")).toBe("5 minutes ago");
    expect(formatRelativeTime(undefined)).toBe("N/A");
  });
});
