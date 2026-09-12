import { describe, expect, it } from "vitest";
import type { FactoriesFactory } from "@/api-client";
import {
  factoryRouteNeedsCanonicalRedirect,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
} from "./factoryKeyResolution";

const FACTORIES: FactoriesFactory[] = [
  { id: "factory-1", key: "SP", name: "Superplane" },
  { id: "factory-2", key: "RF", name: "Refunds" },
];

describe("resolveFactoryByKey", () => {
  it("matches an exact key", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "SP", false);
    expect(resolution).toEqual({ status: "found", factory: FACTORIES[0], matchedBy: "key" });
  });

  it("matches a key case-insensitively", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "sp", false);
    expect(resolution.status).toBe("found");
    expect(resolution.factory).toBe(FACTORIES[0]);
    expect(resolution.matchedBy).toBe("key");
  });

  it("falls back to a legacy id match when no key matches", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "factory-2", false);
    expect(resolution).toEqual({ status: "found", factory: FACTORIES[1], matchedBy: "id" });
  });

  it("does not treat a UUID as a workspace key even when its letters match a key", () => {
    const uuid = "abcde000-0000-4000-8000-000000000000";
    const factories: FactoriesFactory[] = [
      { id: "factory-letters", key: "ABCDE", name: "Letters" },
      { id: uuid, key: "SP", name: "Superplane" },
    ];

    const resolution = resolveFactoryByKey(factories, uuid, false);

    expect(resolution).toEqual({ status: "found", factory: factories[1], matchedBy: "id" });
  });

  it("returns not-found once loaded and nothing matches", () => {
    expect(resolveFactoryByKey(FACTORIES, "nope", false)).toEqual({
      status: "not-found",
      factory: null,
      matchedBy: null,
    });
  });

  it("returns loading instead of not-found while the list is still fetching", () => {
    expect(resolveFactoryByKey([], "SP", true)).toEqual({ status: "loading", factory: null, matchedBy: null });
  });

  it("returns not-found for an empty route segment once loaded", () => {
    expect(resolveFactoryByKey(FACTORIES, undefined, false).status).toBe("not-found");
  });
});

describe("factoryRouteNeedsCanonicalRedirect", () => {
  it("is false for a lowercase key match — the canonical URL form", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "sp", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "sp")).toBe(false);
  });

  it("is true for an uppercase route segment, even though it matches `factory.key` exactly", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "SP", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "SP")).toBe(true);
  });

  it("is true for a mixed-case route segment", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "Sp", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "Sp")).toBe(true);
  });

  it("is true for a legacy id match", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "factory-2", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "factory-2")).toBe(true);
  });

  it("is false when nothing resolved", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "nope", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "nope")).toBe(false);
  });
});

describe("replaceFactoryKeySegment", () => {
  it("swaps the factory key segment and keeps the rest of the path, lowercased", () => {
    expect(
      replaceFactoryKeySegment("/org-1/workspaces/factory-2/work-orders/order-1", "org-1", "factory-2", "RF"),
    ).toBe("/org-1/workspaces/rf/work-orders/order-1");
  });

  it("keeps the workspace root when there is no trailing path", () => {
    expect(replaceFactoryKeySegment("/org-1/workspaces/SP", "org-1", "SP", "SP")).toBe("/org-1/workspaces/sp");
  });

  it("lowercases the canonical key even when the route segment did not match the prefix", () => {
    expect(replaceFactoryKeySegment("/org-1/other", "org-1", "SP", "SP")).toBe("/org-1/workspaces/sp");
  });
});
