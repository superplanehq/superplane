import { FEATURE_ORGANIZATION_BYOK, FEATURE_SUPERPLANE_MCP_SERVER } from "@/lib/experimentalFeatures";
import { describe, expect, it } from "vitest";

import { FACTORY_SETTINGS_NAV_GROUPS } from "./settingsNavItems";
import { visibleFactorySettingsNavGroups } from "./useVisibleFactorySettingsNavGroups";

const enabledFeatures = new Set([FEATURE_ORGANIZATION_BYOK, FEATURE_SUPERPLANE_MCP_SERVER]);

function itemIds(hideBillingPages: boolean): string[] {
  return visibleFactorySettingsNavGroups(
    FACTORY_SETTINGS_NAV_GROUPS,
    (featureId) => enabledFeatures.has(featureId),
    hideBillingPages,
  ).flatMap((group) => group.items.map((item) => item.id));
}

describe("visibleFactorySettingsNavGroups", () => {
  it("keeps Billing when billing pages are visible", () => {
    expect(itemIds(false)).toContain("organization-billing");
  });

  it("drops Billing when billing pages are hidden", () => {
    expect(itemIds(true)).not.toContain("organization-billing");
  });
});
