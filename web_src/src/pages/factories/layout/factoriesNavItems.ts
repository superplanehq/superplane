import { BookOpen, ClipboardList, Crosshair, Gauge, Layers, LayoutDashboard, Workflow } from "lucide-react";
import type { LucideIcon } from "lucide-react";
import {
  automationsPath,
  factoryMissionsPath,
  factoryOverviewPath,
  factoryVelocityPath,
  factoryWikiPath,
  linesPath,
  workOrdersPath,
} from "../lib/factoryPagePaths";

export type FactoriesNavKind = "overview" | "missions" | "work-orders" | "lines" | "automations" | "wiki" | "velocity";

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
    Icon: LayoutDashboard,
    buildHref: factoryOverviewPath,
  },
  {
    id: "missions",
    label: "Missions",
    Icon: Crosshair,
    buildHref: factoryMissionsPath,
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
