import type { LucideIcon } from "lucide-react";
import { BarChart3, Bell, Blocks, Box, CircleUser, Cpu, Grid3x3, Key, Plug, Users } from "lucide-react";

export type FactorySettingsSection =
  | "general"
  | "repositories"
  | "environments"
  | "models"
  | "members"
  | "integrations"
  | "secrets"
  | "usage"
  | "profile"
  | "notifications";

export type FactorySettingsNavGroup = "workspace" | "governance" | "you";

export interface FactorySettingsNavItem {
  id: FactorySettingsSection;
  label: string;
  Icon: LucideIcon;
  group: FactorySettingsNavGroup;
}

export const FACTORY_SETTINGS_NAV_ITEMS: FactorySettingsNavItem[] = [
  { id: "general", label: "General", Icon: Grid3x3, group: "workspace" },
  { id: "repositories", label: "Repositories", Icon: Blocks, group: "workspace" },
  { id: "environments", label: "Environments", Icon: Box, group: "workspace" },
  { id: "models", label: "Models", Icon: Cpu, group: "workspace" },
  { id: "members", label: "Members", Icon: Users, group: "governance" },
  { id: "integrations", label: "Integrations", Icon: Plug, group: "governance" },
  { id: "secrets", label: "Secrets", Icon: Key, group: "governance" },
  { id: "usage", label: "Usage", Icon: BarChart3, group: "governance" },
  { id: "profile", label: "General", Icon: CircleUser, group: "you" },
  { id: "notifications", label: "Notifications", Icon: Bell, group: "you" },
];

const IMPLEMENTED_SETTINGS_SECTIONS = new Set<FactorySettingsSection>(["general", "profile", "notifications"]);

export function isFactorySettingsComingSoon(item: FactorySettingsNavItem) {
  return !IMPLEMENTED_SETTINGS_SECTIONS.has(item.id);
}
