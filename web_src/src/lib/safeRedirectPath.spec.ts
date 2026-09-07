import { describe, expect, it } from "vitest";

import { getSafeRedirectPath, isSafeRedirectPath } from "./safeRedirectPath";

describe("isSafeRedirectPath", () => {
  it("accepts relative in-app paths", () => {
    expect(isSafeRedirectPath("/acme")).toBe(true);
    expect(isSafeRedirectPath("/acme/apps/deploy?run=1&node=approve-1")).toBe(true);
  });

  it("rejects empty, missing, and non-relative values", () => {
    expect(isSafeRedirectPath(null)).toBe(false);
    expect(isSafeRedirectPath(undefined)).toBe(false);
    expect(isSafeRedirectPath("")).toBe(false);
    expect(isSafeRedirectPath("acme")).toBe(false);
  });

  it("rejects protocol-relative and absolute URLs", () => {
    expect(isSafeRedirectPath("//evil.com")).toBe(false);
    expect(isSafeRedirectPath("\\\\evil.com")).toBe(false);
    expect(isSafeRedirectPath("https://evil.com")).toBe(false);
  });
});

describe("getSafeRedirectPath", () => {
  it("decodes and validates a URL-encoded path", () => {
    expect(getSafeRedirectPath(encodeURIComponent("/acme/apps/deploy?run=1"))).toBe("/acme/apps/deploy?run=1");
  });

  it("returns null for missing or unsafe values", () => {
    expect(getSafeRedirectPath(null)).toBeNull();
    expect(getSafeRedirectPath("")).toBeNull();
    expect(getSafeRedirectPath(encodeURIComponent("//evil.com"))).toBeNull();
  });

  it("returns null for a value that cannot be decoded", () => {
    expect(getSafeRedirectPath("%")).toBeNull();
  });
});
