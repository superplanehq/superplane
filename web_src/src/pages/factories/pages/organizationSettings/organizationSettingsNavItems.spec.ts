import { LayoutGrid, Settings } from "lucide-react";
import { describe, expect, it } from "vitest";

import { isOrganizationSettingsComingSoon, ORGANIZATION_SETTINGS_NAV_ITEMS } from "./organizationSettingsNavItems";

describe("ORGANIZATION_SETTINGS_NAV_ITEMS", () => {
  it("lists organization sections in the settings menu order", () => {
    expect(ORGANIZATION_SETTINGS_NAV_ITEMS.map((item) => item.label)).toEqual([
      "General",
      "Workspaces",
      "Members",
      "API Keys",
      "Groups",
      "Roles",
      "Integrations",
      "Usage",
      "Secrets",
    ]);
  });

  it("uses a settings icon for General and a grid icon for Workspaces", () => {
    expect(ORGANIZATION_SETTINGS_NAV_ITEMS[0].Icon).toBe(Settings);
    expect(ORGANIZATION_SETTINGS_NAV_ITEMS[1].Icon).toBe(LayoutGrid);
  });
});

describe("isOrganizationSettingsComingSoon", () => {
  it("is false for General and Workspaces", () => {
    expect(isOrganizationSettingsComingSoon(ORGANIZATION_SETTINGS_NAV_ITEMS[0])).toBe(false);
    expect(isOrganizationSettingsComingSoon(ORGANIZATION_SETTINGS_NAV_ITEMS[1])).toBe(false);
  });

  it("is true for the remaining sections", () => {
    expect(ORGANIZATION_SETTINGS_NAV_ITEMS.filter(isOrganizationSettingsComingSoon).map((item) => item.id)).toEqual([
      "members",
      "api-keys",
      "groups",
      "roles",
      "integrations",
      "usage",
      "secrets",
    ]);
  });
});
