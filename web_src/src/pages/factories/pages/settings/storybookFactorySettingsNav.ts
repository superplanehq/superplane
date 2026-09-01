import { FACTORY_SETTINGS_NAV_GROUPS, type FactorySettingsNavGroup } from "./settingsNavItems";

/** Storybook can still override nav groups. Production Account nav is now the source. */
export const STORYBOOK_FACTORY_SETTINGS_NAV_GROUPS: FactorySettingsNavGroup[] = FACTORY_SETTINGS_NAV_GROUPS;
