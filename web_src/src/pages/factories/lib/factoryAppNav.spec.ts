import { describe, expect, it } from "vitest";
import { resolveFactoryAppBackNav } from "./factoryAppNav";
import { factoryAppConfigurePath, factoryAppPath, factoryAppRunPath } from "./factoryPagePaths";

describe("resolveFactoryAppBackNav", () => {
  it("returns Automations for from=automations", () => {
    expect(resolveFactoryAppBackNav("org", "fac", { from: "automations" })).toEqual({
      label: "Automations",
      href: "/org/workspaces/fac/automations",
    });
  });

  it("returns automation detail when appId present", () => {
    expect(
      resolveFactoryAppBackNav("org", "fac", {
        from: "automations",
        appId: "app-1",
        appName: "Label to work order",
      }),
    ).toEqual({
      label: "Label to work order",
      href: "/org/workspaces/fac/automations/app-1",
    });
  });

  it("returns line detail when lineId present", () => {
    expect(resolveFactoryAppBackNav("org", "fac", { from: "lines", lineId: "line-1", lineName: "poc" })).toEqual({
      label: "Back",
      href: "/org/workspaces/fac/lines/line-1",
    });
  });

  it("falls back to the workspace index when from is missing", () => {
    expect(resolveFactoryAppBackNav("org", "fac", {})).toEqual({
      label: "Back",
      href: "/org/workspaces/fac",
    });
  });

  it("returns the line board when from=work-order", () => {
    expect(resolveFactoryAppBackNav("org", "fac", { from: "work-order", orderNumber: "42", lineId: "line-1" })).toEqual(
      {
        label: "Back",
        href: "/org/workspaces/fac/lines/line-1",
      },
    );
  });

  it("falls back to the workspace index when from=work-order has no line", () => {
    expect(resolveFactoryAppBackNav("org", "fac", { from: "work-order" })).toEqual({
      label: "Back",
      href: "/org/workspaces/fac",
    });
  });
});

describe("factoryAppPath", () => {
  it("builds embed path with run and from", () => {
    expect(factoryAppRunPath("org", "fac", "app-1", "run-1", { from: "automations" })).toBe(
      "/org/workspaces/fac/apps/app-1?run=run-1&from=automations",
    );
    expect(factoryAppPath("org", "fac", "app-1", { from: "lines", lineId: "line-1" })).toBe(
      "/org/workspaces/fac/apps/app-1?from=lines&lineId=line-1",
    );
  });

  it("builds configure path with configure=1 and components closed", () => {
    expect(factoryAppConfigurePath("org", "fac", "app-1", { from: "automations" })).toBe(
      "/org/workspaces/fac/apps/app-1?configure=1&from=automations",
    );
  });
});
