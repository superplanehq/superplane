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
    const toLocaleDateStringSpy = vi.spyOn(Date.prototype, "toLocaleDateString").mockReturnValue("Mar 29, 2026, 14:30");

    expect(formatTimestampInUserTimezone("2026-03-29T14:30:00.000Z", "UTC")).toBe("Mar 29, 2026, 14:30 UTC");
    expect(toLocaleDateStringSpy).toHaveBeenCalledWith("en-US", {
      month: "short",
      day: "numeric",
      year: "numeric",
      hour: "2-digit",
      minute: "2-digit",
      hour12: false,
    });
  });

  it("formats relative time in abbreviated and long forms", () => {
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z", true)).toBe("5m ago");
    expect(formatRelativeTime("2026-03-29T11:55:00.000Z")).toBe("5 minutes ago");
    expect(formatRelativeTime(undefined)).toBe("N/A");
  });
});
