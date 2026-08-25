import { describe, expect, it } from "vitest";

import { isReservedAppPathSegment } from "./reservedAppPaths";

describe("isReservedAppPathSegment", () => {
  it("treats admin and other app roots as reserved", () => {
    expect(isReservedAppPathSegment("admin")).toBe(true);
    expect(isReservedAppPathSegment("login")).toBe(true);
    expect(isReservedAppPathSegment("setup")).toBe(true);
    expect(isReservedAppPathSegment("create")).toBe(true);
  });

  it("allows organization ids", () => {
    expect(isReservedAppPathSegment("3ee1aa47-3a60-4c1f-b645-0b9859ab91f8")).toBe(false);
    expect(isReservedAppPathSegment(undefined)).toBe(false);
  });
});
