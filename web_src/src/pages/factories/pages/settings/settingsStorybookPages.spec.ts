import { describe, expect, it } from "vitest";

import {
  STORYBOOK_FACTORY_SETTINGS_PAGES,
  STORYBOOK_FACTORY_SETTINGS_SECTIONS,
  resolveFactorySettingsPage,
} from "./settingsStorybookPages";
import { FACTORY_SETTINGS_NAV_ITEMS } from "./settingsNavItems";

describe("resolveFactorySettingsPage", () => {
  it("overrides Repositories, Models, Members, Integrations, and Secrets", () => {
    expect([...STORYBOOK_FACTORY_SETTINGS_SECTIONS]).toEqual([
      "repositories",
      "models",
      "members",
      "integrations",
      "secrets",
    ]);
    expect(Object.keys(STORYBOOK_FACTORY_SETTINGS_PAGES).sort()).toEqual(
      [...STORYBOOK_FACTORY_SETTINGS_SECTIONS].sort(),
    );
  });

  it("uses the Storybook page when the section is overridden", () => {
    expect(resolveFactorySettingsPage("repositories", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBe(
      STORYBOOK_FACTORY_SETTINGS_PAGES.repositories,
    );
    expect(resolveFactorySettingsPage("models", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBe(
      STORYBOOK_FACTORY_SETTINGS_PAGES.models,
    );
    expect(resolveFactorySettingsPage("members", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBe(
      STORYBOOK_FACTORY_SETTINGS_PAGES.members,
    );
    expect(resolveFactorySettingsPage("integrations", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBe(
      STORYBOOK_FACTORY_SETTINGS_PAGES.integrations,
    );
    expect(resolveFactorySettingsPage("secrets", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBe(
      STORYBOOK_FACTORY_SETTINGS_PAGES.secrets,
    );
  });

  it("keeps General and other sections on Coming Soon when no override is set", () => {
    expect(resolveFactorySettingsPage("general", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBeUndefined();
    expect(resolveFactorySettingsPage("environments", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBeUndefined();
    expect(resolveFactorySettingsPage("usage", STORYBOOK_FACTORY_SETTINGS_PAGES)).toBeUndefined();
    expect(resolveFactorySettingsPage("members")).toBeUndefined();
  });

  it("does not take General away from the live page even if an override is passed", () => {
    const GeneralOverride = () => null;
    expect(resolveFactorySettingsPage("general", { general: GeneralOverride })).toBeUndefined();
  });

  it("covers every settings nav item as General, override, or Coming Soon", () => {
    const kinds = FACTORY_SETTINGS_NAV_ITEMS.map((item) => {
      if (item.id === "general") return "general";
      return resolveFactorySettingsPage(item.id, STORYBOOK_FACTORY_SETTINGS_PAGES) ? "override" : "soon";
    });

    expect(kinds).toEqual(["general", "override", "soon", "override", "override", "override", "override", "soon"]);
  });
});
