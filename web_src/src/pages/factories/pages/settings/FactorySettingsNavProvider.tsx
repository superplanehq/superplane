import type { ReactNode } from "react";

import { FactorySettingsNavContext } from "./factorySettingsNavContext";
import type { FactorySettingsNavGroup } from "./settingsNavItems";

export function FactorySettingsNavProvider({
  groups,
  children,
}: {
  groups: FactorySettingsNavGroup[];
  children: ReactNode;
}) {
  return <FactorySettingsNavContext.Provider value={groups}>{children}</FactorySettingsNavContext.Provider>;
}
