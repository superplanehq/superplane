import { describe, expect, it } from "vitest";

import { FACTORY_SETTINGS_NAV_GROUPS } from "./settingsNavItems";
import {
  buildFactorySettingsSearchIndex,
  factorySettingsIntegrationSearchEntries,
  factorySettingsSearchResultPath,
  searchFactorySettings,
} from "./settingsSearch";

const index = buildFactorySettingsSearchIndex({
  navGroups: FACTORY_SETTINGS_NAV_GROUPS,
  integrations: [
    { name: "claude", label: "Claude", description: "Use Claude models in workflows" },
    { name: "github", label: "GitHub", description: "Connect repositories" },
  ],
});

describe("searchFactorySettings", () => {
  it("returns Claude under Organization Integrations", () => {
    const results = searchFactorySettings(index, "claude");
    expect(results[0]?.title).toBe("Claude");
    expect(results[0]?.breadcrumb).toEqual(["Organization", "Integrations"]);
    expect(results[0]?.anchor).toBe("integration-claude");
    expect(results[0]?.section).toBe("integrations");
  });

  it("returns Workspace key under Workspace General", () => {
    const results = searchFactorySettings(index, "workspace key");
    expect(results[0]?.title).toBe("Workspace key");
    expect(results[0]?.breadcrumb).toEqual(["Workspace", "General"]);
    expect(results[0]?.anchor).toBe("factory-settings-key");
  });

  it("returns an empty list for a miss", () => {
    expect(searchFactorySettings(index, "zzzz-no-match")).toEqual([]);
  });

  it("indexes known providers like Claude even without a live catalog response", () => {
    const withoutCatalog = buildFactorySettingsSearchIndex({ navGroups: FACTORY_SETTINGS_NAV_GROUPS });
    const results = searchFactorySettings(withoutCatalog, "claude");
    expect(results[0]?.title).toBe("Claude");
    expect(results[0]?.anchor).toBe("integration-claude");
  });

  it("keeps the specific Identity hit and drops the Profile page hit for email", () => {
    const results = searchFactorySettings(index, "email");
    const titles = results.map((result) => result.title);
    expect(titles).toContain("Identity");
    expect(titles).not.toContain("Profile");
    expect(results.find((result) => result.title === "Identity")?.breadcrumb).toEqual(["Account", "Profile"]);
  });

  it("keeps the Security page and Sign in methods when the query matches the page title", () => {
    const titles = searchFactorySettings(index, "secur").map((result) => result.title);
    expect(titles).toContain("Security");
    expect(titles).toContain("Sign in methods");
  });
});

describe("factorySettingsIntegrationSearchEntries", () => {
  it("builds an anchor per provider", () => {
    expect(factorySettingsIntegrationSearchEntries([{ name: "claude", label: "Claude" }])).toEqual([
      expect.objectContaining({
        id: "integration:claude",
        title: "Claude",
        anchor: "integration-claude",
      }),
    ]);
  });
});

describe("factorySettingsSearchResultPath", () => {
  it("appends the section query when an anchor exists", () => {
    const result = searchFactorySettings(index, "claude")[0]!;
    expect(
      factorySettingsSearchResultPath("org", "RF", result, (_o, _f, scope, section) => `/settings/${scope}/${section}`),
    ).toBe("/settings/organization/integrations?section=integration-claude");
  });
});
