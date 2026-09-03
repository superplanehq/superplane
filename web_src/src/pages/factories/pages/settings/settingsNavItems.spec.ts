import { describe, expect, it } from "vitest";

import { FACTORY_SETTINGS_NAV_GROUPS, factorySettingsRouteFromPathname } from "./settingsNavItems";

describe("factorySettingsRouteFromPathname", () => {
  it("reads a canonical scoped settings route", () => {
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/profile")?.id).toBe("account-profile");
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/security")?.id).toBe(
      "account-security",
    );
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/workspace/automations")?.id).toBe(
      "workspace-automations",
    );
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/organization/api-keys")?.id).toBe(
      "organization-api-keys",
    );
  });

  it("returns undefined for a non-canonical route", () => {
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/general")).toBeUndefined();
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/overview")).toBeUndefined();
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/linked-accounts")).toBeUndefined();
  });
});

describe("FACTORY_SETTINGS_NAV_GROUPS", () => {
  it("contains only the approved settings in display order", () => {
    expect(FACTORY_SETTINGS_NAV_GROUPS.map((group) => group.label)).toEqual(["Account", "Workspace", "Organization"]);
    expect(FACTORY_SETTINGS_NAV_GROUPS.flatMap((group) => group.items.map((item) => item.label))).toEqual([
      "Profile",
      "Security",
      "Notifications",
      "General",
      "Repository",
      "Automations",
      "Models",
      "Spending",
      "General",
      "Members",
      "Integrations",
      "API keys",
      "Secrets",
      "Spending",
    ]);
  });
});
