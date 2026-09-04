import { describe, expect, it } from "vitest";

import { isReservedAppPathSegment } from "./reservedAppPaths";

describe("isReservedAppPathSegment", () => {
  it("treats admin and other app roots as reserved", () => {
    expect(isReservedAppPathSegment("admin")).toBe(true);
    expect(isReservedAppPathSegment("login")).toBe(true);
    expect(isReservedAppPathSegment("onboarding")).toBe(true);
    expect(isReservedAppPathSegment("setup")).toBe(true);
    expect(isReservedAppPathSegment("create")).toBe(true);
  });

  it("treats infrastructure roots as reserved", () => {
    expect(isReservedAppPathSegment("api")).toBe(true);
    expect(isReservedAppPathSegment("health")).toBe(true);
    expect(isReservedAppPathSegment("assets")).toBe(true);
    expect(isReservedAppPathSegment("logout")).toBe(true);
  });

  it("allows organization ids and slugs", () => {
    expect(isReservedAppPathSegment("3ee1aa47-3a60-4c1f-b645-0b9859ab91f8")).toBe(false);
    expect(isReservedAppPathSegment("acme-corp")).toBe(false);
    expect(isReservedAppPathSegment(undefined)).toBe(false);
  });
});
