import { useContext } from "react";

import { FactorySettingsNavContext } from "./factorySettingsNavContext";
import { FACTORY_SETTINGS_NAV_GROUPS } from "./settingsNavItems";

export function useFactorySettingsNavGroups() {
  return useContext(FactorySettingsNavContext) ?? FACTORY_SETTINGS_NAV_GROUPS;
}
