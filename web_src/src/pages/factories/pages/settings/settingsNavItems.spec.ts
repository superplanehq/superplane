import { describe, expect, it } from "vitest";

import {
  FACTORY_SETTINGS_NAV_GROUPS,
  factorySettingsRouteFromPathname,
  filterFactorySettingsNavGroups,
} from "./settingsNavItems";

describe("factorySettingsRouteFromPathname", () => {
  it("reads a canonical scoped settings route", () => {
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/profile")?.id).toBe("account-profile");
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/notifications")?.id).toBe(
      "account-notifications",
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
    expect(factorySettingsRouteFromPathname("/org/workspaces/RF/settings/account/security")).toBeUndefined();
  });
});

describe("FACTORY_SETTINGS_NAV_GROUPS", () => {
  it("contains only the approved settings in display order", () => {
    expect(FACTORY_SETTINGS_NAV_GROUPS.map((group) => group.label)).toEqual(["Account", "Workspace", "Organization"]);
    expect(FACTORY_SETTINGS_NAV_GROUPS.flatMap((group) => group.items.map((item) => item.label))).toEqual([
      "Account",
      "Notifications",
      "General",
      "Repository",
      "Automations",
      "Models",
      "General",
      "Members",
      "Integrations",
      "API keys",
      "Secrets",
      "Spending",
    ]);
  });
});

describe("filterFactorySettingsNavGroups", () => {
  it("returns the full list when the query is empty or whitespace", () => {
    expect(filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "")).toEqual(FACTORY_SETTINGS_NAV_GROUPS);
    expect(filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "   ")).toEqual(FACTORY_SETTINGS_NAV_GROUPS);
  });

  it("keeps items whose label matches the query", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "notif");
    expect(filtered.map((group) => group.id)).toEqual(["account"]);
    expect(filtered[0]?.items.map((item) => item.id)).toEqual(["account-notifications"]);
  });

  it("matches security keywords on the combined Account page", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "secur");
    expect(filtered.map((group) => group.id)).toEqual(["account"]);
    expect(filtered[0]?.items.map((item) => item.id)).toEqual(["account-profile"]);
  });

  it("keeps every item in a group when the group label matches", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "workspace");
    const workspaceGroup = filtered.find((group) => group.id === "workspace");
    expect(workspaceGroup?.items.map((item) => item.id)).toEqual([
      "workspace-general",
      "workspace-repository",
      "workspace-automations",
      "workspace-models",
    ]);
  });

  it("matches keyword aliases such as billing", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "billing");
    expect(filtered.flatMap((group) => group.items.map((item) => item.id))).toEqual(["organization-spending"]);
  });

  it("drops groups that have no matching items", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "api keys");
    expect(filtered.map((group) => group.id)).toEqual(["organization"]);
    expect(filtered[0]?.items.map((item) => item.id)).toEqual(["organization-api-keys"]);
  });

  it("returns an empty list when nothing matches", () => {
    expect(filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "zzzz-no-match")).toEqual([]);
  });

  it("matches in-page field labels such as workspace key", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "workspace key");
    expect(filtered.map((group) => group.id)).toEqual(["workspace"]);
    expect(filtered[0]?.items.map((item) => item.id)).toEqual(["workspace-general"]);
  });

  it("matches in-page phrases when query words appear in any order", () => {
    const filtered = filterFactorySettingsNavGroups(FACTORY_SETTINGS_NAV_GROUPS, "key workspace");
    expect(filtered.flatMap((group) => group.items.map((item) => item.id))).toContain("workspace-general");
  });
});
