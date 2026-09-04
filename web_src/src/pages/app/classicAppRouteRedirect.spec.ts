import { describe, expect, it } from "vitest";

import { decideClassicAppRouteRedirect, factoryKeyForId } from "./classicAppRouteRedirect";

const BASE = {
  featureLoading: false,
  factoriesEnabled: false,
  factoriesLoading: false,
  factoryOwnedApp: false,
  factoryKey: undefined as string | undefined,
  organizationId: "org-1",
  appId: "canvas-1",
  runId: null as string | null,
};

describe("factoryKeyForId", () => {
  it("returns the workspace key for a matching factory id", () => {
    expect(factoryKeyForId([{ id: "factory-1", key: "RF" }], "factory-1")).toBe("RF");
  });

  it("returns undefined when the factory id is missing or unknown", () => {
    expect(factoryKeyForId([{ id: "factory-1", key: "RF" }], "other")).toBeUndefined();
    expect(factoryKeyForId([{ id: "factory-1", key: "RF" }], null)).toBeUndefined();
  });
});

describe("decideClassicAppRouteRedirect", () => {
  it("waits while the feature flag is loading", () => {
    expect(decideClassicAppRouteRedirect({ ...BASE, featureLoading: true })).toEqual({ kind: "wait" });
  });

  it("leaves classic leftover apps on the classic route when factories are off", () => {
    expect(decideClassicAppRouteRedirect(BASE)).toEqual({ kind: "none" });
  });

  it("sends leftover classic apps to /workspaces when factories are on", () => {
    expect(decideClassicAppRouteRedirect({ ...BASE, factoriesEnabled: true })).toEqual({
      kind: "redirect",
      to: "/org-1/workspaces",
    });
  });

  it("sends factory-owned apps to org home when factories are off", () => {
    expect(decideClassicAppRouteRedirect({ ...BASE, factoryOwnedApp: true, factoryKey: "RF" })).toEqual({
      kind: "redirect",
      to: "/org-1",
    });
  });

  it("waits for the factory list before opening a factory-owned app when factories are on", () => {
    expect(
      decideClassicAppRouteRedirect({
        ...BASE,
        factoriesEnabled: true,
        factoryOwnedApp: true,
        factoriesLoading: true,
      }),
    ).toEqual({ kind: "wait" });
  });

  it("opens the workspace editor for a factory-owned app when factories are on", () => {
    expect(
      decideClassicAppRouteRedirect({
        ...BASE,
        factoriesEnabled: true,
        factoryOwnedApp: true,
        factoryKey: "RF",
      }),
    ).toEqual({
      kind: "redirect",
      to: "/org-1/workspaces/RF/apps/canvas-1?configure=1&agent=1",
    });
  });

  it("keeps a run query on the workspace app URL", () => {
    expect(
      decideClassicAppRouteRedirect({
        ...BASE,
        factoriesEnabled: true,
        factoryOwnedApp: true,
        factoryKey: "RF",
        runId: "run-9",
      }),
    ).toEqual({
      kind: "redirect",
      to: "/org-1/workspaces/RF/apps/canvas-1?run=run-9",
    });
  });

  it("sends factory-owned apps to /workspaces when the factory key cannot be resolved", () => {
    expect(
      decideClassicAppRouteRedirect({
        ...BASE,
        factoriesEnabled: true,
        factoryOwnedApp: true,
      }),
    ).toEqual({
      kind: "redirect",
      to: "/org-1/workspaces",
    });
  });
});
