import { createContext, useContext } from "react";
import { BookOpen, ClipboardList, Gauge, Layers, LayoutDashboard, Workflow } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import {
  automationsPath,
  factoryOverviewPath,
  factoryVelocityPath,
  factoryWikiPath,
  linesPath,
  workOrdersPath,
} from "../lib/factoryPagePaths";

export type FactoriesNavKind = "overview" | "work-orders" | "missions" | "lines" | "automations" | "wiki" | "velocity";

export interface FactoriesNavItem {
  id: FactoriesNavKind;
  label: string;
  Icon: LucideIcon;
  buildHref: (organizationId: string, factoryKey: string) => string;
}

export const FACTORIES_NAV_ITEMS: FactoriesNavItem[] = [
  {
    id: "overview",
    label: "Overview",
    Icon: LayoutDashboard,
    buildHref: factoryOverviewPath,
  },
  {
    id: "work-orders",
    label: "Work Orders",
    Icon: ClipboardList,
    buildHref: workOrdersPath,
  },
  {
    id: "lines",
    label: "Lines",
    Icon: Layers,
    buildHref: linesPath,
  },
  {
    id: "automations",
    label: "Automations",
    Icon: Workflow,
    buildHref: automationsPath,
  },
  {
    id: "wiki",
    label: "Wiki",
    Icon: BookOpen,
    buildHref: factoryWikiPath,
  },
  {
    id: "velocity",
    label: "Velocity",
    Icon: Gauge,
    buildHref: factoryVelocityPath,
  },
];

/** Storybook can replace this list. The live app keeps `FACTORIES_NAV_ITEMS`. */
export const FactoriesNavItemsContext = createContext<readonly FactoriesNavItem[]>(FACTORIES_NAV_ITEMS);

export function useFactoriesNavItems() {
  return useContext(FactoriesNavItemsContext);
}
