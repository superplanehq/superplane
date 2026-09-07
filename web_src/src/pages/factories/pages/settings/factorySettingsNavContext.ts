import { createContext } from "react";

import type { FactorySettingsNavGroup } from "./settingsNavItems";

export const FactorySettingsNavContext = createContext<FactorySettingsNavGroup[] | undefined>(undefined);
