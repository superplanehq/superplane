import { describe, expect, it } from "vitest";

import { getValueAtPath, interpolate, parsePath } from "./fieldPath";

describe("fieldPath helpers", () => {
  it("getValueAtPath resolves nested objects", () => {
    expect(getValueAtPath({ a: { b: { c: 1 } } }, "a.b.c")).toBe(1);
  });

  it("getValueAtPath resolves bracket notation", () => {
    expect(getValueAtPath({ a: [{ b: 2 }] }, "a[0].b")).toBe(2);
  });

  it("getValueAtPath returns undefined for missing paths", () => {
    expect(getValueAtPath({ a: 1 }, "a.b.c")).toBeUndefined();
  });

  it("parsePath handles mixed dot/bracket notation", () => {
    expect(parsePath('a.b[0]["c"]')).toEqual(["a", "b", "0", "c"]);
  });

  it("interpolate substitutes placeholders from a row", () => {
    expect(interpolate("https://x/{service}/{version}", { service: "api", version: "v2" })).toBe("https://x/api/v2");
  });

  it("interpolate omits missing fields", () => {
    expect(interpolate("/{a}/{b}", { a: 1 })).toBe("/1/");
  });
});
