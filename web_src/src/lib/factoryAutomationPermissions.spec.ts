import { describe, expect, it } from "bun:test";

import { resolveCanUpdateFactory, resolveCanUpdateFactoryAutomation } from "./factoryAutomationPermissions";

describe("resolveCanUpdateFactory", () => {
  it("is false while permissions load", () => {
    expect(resolveCanUpdateFactory(true, () => true)).toBe(false);
  });

  it("is false without factories update", () => {
    expect(resolveCanUpdateFactory(false, (resource, action) => resource === "factories" && action === "read")).toBe(
      false,
    );
  });

  it("is true after permissions load with factories update", () => {
    expect(resolveCanUpdateFactory(false, (resource, action) => resource === "factories" && action === "update")).toBe(
      true,
    );
  });
});

describe("resolveCanUpdateFactoryAutomation", () => {
  it("is false while permissions load", () => {
    expect(resolveCanUpdateFactoryAutomation(true, () => true)).toBe(false);
  });

  it("is false without canvases update", () => {
    expect(
      resolveCanUpdateFactoryAutomation(false, (resource, action) => resource === "canvases" && action === "read"),
    ).toBe(false);
  });

  it("is true after permissions load with canvases update", () => {
    expect(
      resolveCanUpdateFactoryAutomation(false, (resource, action) => resource === "canvases" && action === "update"),
    ).toBe(true);
  });
});
