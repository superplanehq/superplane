import { describe, expect, it } from "bun:test";

import { sessionAgeSeconds } from "./useSsoSettingsState";

describe("sessionAgeSeconds", () => {
  it("accepts whole hours from 1 to 168", () => {
    expect(sessionAgeSeconds("1")).toBe(3600);
    expect(sessionAgeSeconds("24")).toBe(86400);
    expect(sessionAgeSeconds("168")).toBe(604800);
  });

  it("rejects empty, partial, and out-of-range values", () => {
    expect(sessionAgeSeconds("")).toBeNull();
    expect(sessionAgeSeconds("abc")).toBeNull();
    expect(sessionAgeSeconds("0")).toBeNull();
    expect(sessionAgeSeconds("169")).toBeNull();
  });
});
