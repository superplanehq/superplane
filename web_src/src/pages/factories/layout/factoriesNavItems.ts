import { BarChart3, BookOpen, ClipboardList, LayoutGrid, Rocket, Workflow } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import {
  automationsPath,
  factoryMissionsPath,
  factoryOverviewPath,
  factoryVelocityPath,
  factoryWikiPath,
  workOrdersPath,
} from "../lib/factoryPagePaths";

export type FactoriesNavKind = "overview" | "missions" | "work-orders" | "automations" | "wiki" | "velocity";

export interface FactoriesNavItem {
  id: FactoriesNavKind;
  label: string;
  Icon: LucideIcon;
  buildHref: (organizationId: string, factoryId: string) => string;
}

export const FACTORIES_NAV_ITEMS: FactoriesNavItem[] = [
  {
    id: "overview",
    label: "Overview",
    Icon: LayoutGrid,
    buildHref: factoryOverviewPath,
  },
  {
    id: "missions",
    label: "Missions",
    Icon: Rocket,
    buildHref: factoryMissionsPath,
  },
  {
    id: "work-orders",
    label: "Work Orders",
    Icon: ClipboardList,
    buildHref: workOrdersPath,
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
    Icon: BarChart3,
    buildHref: factoryVelocityPath,
  },
];
