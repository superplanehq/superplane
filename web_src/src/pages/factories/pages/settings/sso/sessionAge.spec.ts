import { describe, expect, it } from "bun:test";

import { formatSessionAge, sessionAgeSeconds } from "./useSsoSettingsState";

describe("sessionAgeSeconds", () => {
  it("accepts whole hours, fractional hours, and minutes", () => {
    expect(sessionAgeSeconds("1")).toBe(3600);
    expect(sessionAgeSeconds("24")).toBe(86400);
    expect(sessionAgeSeconds("168")).toBe(604800);
    expect(sessionAgeSeconds("0.25")).toBe(900);
    expect(sessionAgeSeconds("1.5")).toBe(5400);
    expect(sessionAgeSeconds("15m")).toBe(900);
    expect(sessionAgeSeconds("90m")).toBe(5400);
  });

  it("keeps a stored age that is not a whole hour", () => {
    for (const seconds of [900, 901, 3599, 5400, 604800]) {
      expect(sessionAgeSeconds(formatSessionAge(seconds))).toBe(seconds);
    }
  });

  it("rejects empty, partial, and out-of-range values", () => {
    expect(sessionAgeSeconds("")).toBeNull();
    expect(sessionAgeSeconds("abc")).toBeNull();
    expect(sessionAgeSeconds("0")).toBeNull();
    expect(sessionAgeSeconds("14m")).toBeNull();
    expect(sessionAgeSeconds("169")).toBeNull();
  });
});
