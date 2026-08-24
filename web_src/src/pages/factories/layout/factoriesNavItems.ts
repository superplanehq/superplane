import { BookOpen, ClipboardList, Gauge, House, Layers, LayoutDashboard, Workflow } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import {
  automationsPath,
  factoryHomePath,
  factoryOverviewPath,
  factoryVelocityPath,
  factoryWikiPath,
  linesPath,
  workOrdersPath,
} from "../lib/factoryPagePaths";

export type FactoriesNavKind = "home" | "overview" | "work-orders" | "lines" | "automations" | "wiki" | "velocity";

export interface FactoriesNavItem {
  id: FactoriesNavKind;
  label: string;
  Icon: LucideIcon;
  buildHref: (organizationId: string, factoryKey: string) => string;
}

export const FACTORIES_NAV_ITEMS: FactoriesNavItem[] = [
  {
    id: "home",
    label: "Home",
    Icon: House,
    buildHref: factoryHomePath,
  },
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
