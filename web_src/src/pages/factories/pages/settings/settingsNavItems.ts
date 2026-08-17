import type { LucideIcon } from "lucide-react";
import { BarChart3, Bell, Blocks, Box, Cpu, Grid3x3, Key, Plug, Users } from "lucide-react";

export type FactorySettingsSection =
  | "general"
  | "notifications"
  | "repositories"
  | "environments"
  | "models"
  | "members"
  | "integrations"
  | "secrets"
  | "usage";

export interface FactorySettingsNavItem {
  id: FactorySettingsSection;
  label: string;
  Icon: LucideIcon;
  group: "workspace" | "governance";
}

export const FACTORY_SETTINGS_NAV_ITEMS: FactorySettingsNavItem[] = [
  { id: "general", label: "General", Icon: Grid3x3, group: "workspace" },
  { id: "notifications", label: "Notifications", Icon: Bell, group: "workspace" },
  { id: "repositories", label: "Repositories", Icon: Blocks, group: "workspace" },
  { id: "environments", label: "Environments", Icon: Box, group: "workspace" },
  { id: "models", label: "Models", Icon: Cpu, group: "workspace" },
  { id: "members", label: "Members", Icon: Users, group: "governance" },
  { id: "integrations", label: "Integrations", Icon: Plug, group: "governance" },
  { id: "secrets", label: "Secrets", Icon: Key, group: "governance" },
  { id: "usage", label: "Usage", Icon: BarChart3, group: "governance" },
];
