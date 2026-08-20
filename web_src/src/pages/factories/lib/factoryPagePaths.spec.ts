import { describe, expect, it } from "vitest";
import {
  factoryAppConfigurePath,
  factoryAppPath,
  factoryDetailPath,
  factorySettingsGeneralPathAfterKeyChange,
  legacyWorkOrderDetailPath,
  organizationSettingsPath,
  organizationSettingsSectionPath,
  workOrderDetailPath,
  workOrdersPath,
} from "./factoryPagePaths";

describe("factoryDetailPath", () => {
  it("builds the workspace URL from the workspace key", () => {
    expect(factoryDetailPath("org-1", "SP")).toBe("/org-1/workspaces/SP");
  });
});

describe("workOrderDetailPath", () => {
  it("builds the canonical permalink from the workspace key and work order number", () => {
    expect(workOrderDetailPath("org-1", "SP", 42)).toBe("/org-1/workspaces/SP/work-order/42");
  });

  it("accepts the number as a string", () => {
    expect(workOrderDetailPath("org-1", "SP", "42")).toBe("/org-1/workspaces/SP/work-order/42");
  });

  it("is a sibling of, not nested under, the plural work-orders list path", () => {
    expect(workOrderDetailPath("org-1", "SP", "42")).not.toContain(workOrdersPath("org-1", "SP"));
  });
});

describe("legacyWorkOrderDetailPath", () => {
  it("builds the old id-based shape for back-compat redirects", () => {
    expect(legacyWorkOrderDetailPath("org-1", "SP", "order-uuid")).toBe("/org-1/workspaces/SP/work-orders/order-uuid");
  });
});

describe("factoryAppPath", () => {
  it("encodes orderNumber (not orderId) in the query string", () => {
    expect(factoryAppPath("org-1", "SP", "app-1", { from: "work-order", orderNumber: "42" })).toBe(
      "/org-1/workspaces/SP/apps/app-1?from=work-order&orderNumber=42",
    );
  });
});

describe("factorySettingsGeneralPathAfterKeyChange", () => {
  it("returns the General settings URL when the key changes", () => {
    expect(factorySettingsGeneralPathAfterKeyChange("org-1", "RF", "AB")).toBe("/org-1/workspaces/AB/settings/general");
  });

  it("returns null when the key does not change", () => {
    expect(factorySettingsGeneralPathAfterKeyChange("org-1", "RF", "RF")).toBeNull();
  });
});

describe("factoryAppConfigurePath", () => {
  it("adds configure=1", () => {
    expect(factoryAppConfigurePath("org-1", "SP", "app-1")).toBe("/org-1/workspaces/SP/apps/app-1?configure=1");
  });
});

describe("organizationSettingsPath", () => {
  it("builds organization settings under the current workspace", () => {
    expect(organizationSettingsPath("org-1", "RF")).toBe("/org-1/workspaces/RF/organization");
    expect(organizationSettingsSectionPath("org-1", "RF", "general")).toBe("/org-1/workspaces/RF/organization/general");
  });
});
