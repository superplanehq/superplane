import { FACTORY_SETTINGS_NAV_GROUPS, type FactorySettingsNavGroup } from "./settingsNavItems";

/** Storybook can still override nav groups. Production Account nav is now the source. */
export const STORYBOOK_FACTORY_SETTINGS_NAV_GROUPS: FactorySettingsNavGroup[] = FACTORY_SETTINGS_NAV_GROUPS;

/** Spending lives once, under Organization. Workspace spend limits stay out of this explorer. */
export const ORG_SPENDING_ONLY_NAV_GROUPS: FactorySettingsNavGroup[] = FACTORY_SETTINGS_NAV_GROUPS.map((group) => {
  if (group.id !== "workspace") {
    return group;
  }
  return {
    ...group,
    items: group.items.filter((item) => item.id !== "workspace-spending"),
  };
});
