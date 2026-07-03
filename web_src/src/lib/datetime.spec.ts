import { describe, expect, it } from "vitest";
import { formatAbsolute, formatISO, formatRelative, formatUTC, toDate } from "@/lib/datetime";

describe("datetime", () => {
  const iso = "2026-06-02T10:01:10.561Z";

  describe("toDate", () => {
    it("returns null for missing or invalid values", () => {
      expect(toDate(null)).toBeNull();
      expect(toDate(undefined)).toBeNull();
      expect(toDate("")).toBeNull();
      expect(toDate("not-a-date")).toBeNull();
    });

    it("accepts Date, string, and epoch millis", () => {
      const date = new Date(iso);
      expect(toDate(date)).toEqual(date);
      expect(toDate(iso)).toEqual(date);
      expect(toDate(date.getTime())).toEqual(date);
    });
  });

  describe("formatUTC", () => {
    it("renders the timestamp in UTC without a timezone suffix", () => {
      // Locale fixed for a deterministic assertion; UTC is timezone-stable.
      expect(formatUTC(iso, "en-GB")).toBe("02 Jun 2026, 10:01:10");
    });

    it("returns an empty string for invalid input", () => {
      expect(formatUTC("nope", "en-GB")).toBe("");
    });
  });

  describe("formatAbsolute", () => {
    it("includes a short timezone name", () => {
      // Local timezone is environment-dependent, so assert structure only.
      const formatted = formatAbsolute(iso, "en-GB");
      expect(formatted).toMatch(/Jun 2026/);
      expect(formatted.trim().length).toBeGreaterThan("02 Jun 2026, 10:01:10".length);
    });
  });

  describe("formatRelative", () => {
    const now = new Date(iso).getTime();

    it('reads "1 day ago" rather than "yesterday"', () => {
      const yesterday = now - 24 * 60 * 60 * 1000;
      expect(formatRelative(yesterday, "en", now)).toBe("1 day ago");
    });

    it("handles seconds, minutes, hours, and future times", () => {
      expect(formatRelative(now - 30 * 1000, "en", now)).toBe("30 seconds ago");
      expect(formatRelative(now - 5 * 60 * 1000, "en", now)).toBe("5 minutes ago");
      expect(formatRelative(now - 3 * 60 * 60 * 1000, "en", now)).toBe("3 hours ago");
      expect(formatRelative(now + 2 * 60 * 60 * 1000, "en", now)).toBe("in 2 hours");
    });

    it("scales up to weeks, months, and years", () => {
      expect(formatRelative(now - 10 * 24 * 60 * 60 * 1000, "en", now)).toBe("1 week ago");
      expect(formatRelative(now - 60 * 24 * 60 * 60 * 1000, "en", now)).toBe("2 months ago");
      expect(formatRelative(now - 400 * 24 * 60 * 60 * 1000, "en", now)).toBe("1 year ago");
    });
  });

  describe("formatISO", () => {
    it("returns the full-precision ISO string", () => {
      expect(formatISO(iso)).toBe(iso);
      expect(formatISO(new Date(iso).getTime())).toBe(iso);
    });
  });
});
