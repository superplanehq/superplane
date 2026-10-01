import { describe, expect, it } from "bun:test";
import type { FactoriesFactory } from "@/api-client";
import {
  factoryRouteNeedsCanonicalRedirect,
  factoryRouteSegment,
  replaceFactoryKeySegment,
  resolveFactoryByKey,
  workspaceRouteSegment,
} from "./factoryKeyResolution";

const FACTORIES: FactoriesFactory[] = [
  { id: "factory-1", key: "SP", urlId: "k7m2xqab", name: "Superplane" },
  { id: "factory-2", key: "RF", urlId: "p8n3wrcd", name: "Refunds" },
];

describe("workspaceRouteSegment", () => {
  it("joins the lowercase key and url id", () => {
    expect(workspaceRouteSegment("SP", "k7m2xqab")).toBe("sp-k7m2xqab");
    expect(factoryRouteSegment(FACTORIES[0])).toBe("sp-k7m2xqab");
  });

  it("falls back to the lowercase key when the url id is missing", () => {
    expect(workspaceRouteSegment("SP")).toBe("sp");
  });
});

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

  it("matches a key-urlId segment even when the key prefix is stale", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "newwo-k7m2xqab", false);
    expect(resolution).toEqual({ status: "found", factory: FACTORIES[0], matchedBy: "urlId" });
  });

  it("falls back to a legacy id match when no key matches", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "factory-2", false);
    expect(resolution).toEqual({ status: "found", factory: FACTORIES[1], matchedBy: "id" });
  });

  it("does not treat a UUID as a workspace key even when its letters match a key", () => {
    const uuid = "abcde000-0000-4000-8000-000000000000";
    const factories: FactoriesFactory[] = [
      { id: "factory-letters", key: "ABCDE", urlId: "abcde001", name: "Letters" },
      { id: uuid, key: "SP", urlId: "k7m2xqab", name: "Superplane" },
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
  it("is false for the canonical key-urlId segment", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "sp-k7m2xqab", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "sp-k7m2xqab")).toBe(false);
  });

  it("is true for a lowercase key-only segment when a url id is present", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "sp", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "sp")).toBe(true);
  });

  it("is true for an uppercase route segment", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "SP", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "SP")).toBe(true);
  });

  it("is true for a stale key prefix on a matching url id", () => {
    const resolution = resolveFactoryByKey(FACTORIES, "newwo-k7m2xqab", false);
    expect(factoryRouteNeedsCanonicalRedirect(resolution, "newwo-k7m2xqab")).toBe(true);
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
      replaceFactoryKeySegment("/org-1/workspaces/factory-2/work-orders/order-1", "org-1", "factory-2", "rf-p8n3wrcd"),
    ).toBe("/org-1/workspaces/rf-p8n3wrcd/work-orders/order-1");
  });

  it("keeps the workspace root when there is no trailing path", () => {
    expect(replaceFactoryKeySegment("/org-1/workspaces/SP", "org-1", "SP", "sp-k7m2xqab")).toBe(
      "/org-1/workspaces/sp-k7m2xqab",
    );
  });

  it("does not treat a key-urlId segment as a prefix of a shorter key", () => {
    expect(replaceFactoryKeySegment("/org-1/workspaces/rf-p8n3wrcd/settings", "org-1", "rf", "sp-k7m2xqab")).toBe(
      "/org-1/workspaces/sp-k7m2xqab",
    );
  });

  it("lowercases the canonical key even when the route segment did not match the prefix", () => {
    expect(replaceFactoryKeySegment("/org-1/other", "org-1", "SP", "sp-k7m2xqab")).toBe(
      "/org-1/workspaces/sp-k7m2xqab",
    );
  });
});
