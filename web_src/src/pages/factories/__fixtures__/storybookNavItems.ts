import { Crosshair } from "lucide-react";

import { factoryMissionsPath } from "../lib/factoryPagePaths";
import { FACTORIES_NAV_ITEMS, type FactoriesNavItem } from "../layout/factoriesNavItems";

/** Storybook sidebar: live items plus Missions after Work Orders. */
export const STORYBOOK_FACTORIES_NAV_ITEMS: FactoriesNavItem[] = [
  ...FACTORIES_NAV_ITEMS.slice(0, 2),
  {
    id: "missions",
    label: "Missions",
    Icon: Crosshair,
    buildHref: factoryMissionsPath,
  },
  ...FACTORIES_NAV_ITEMS.slice(2),
];
