import type { LucideIcon } from "lucide-react";
import { Gauge, Key, KeyRound, LayoutGrid, Plug, Settings, Shield, User, Users } from "lucide-react";

export type OrganizationSettingsSection =
  | "general"
  | "workspaces"
  | "members"
  | "api-keys"
  | "groups"
  | "roles"
  | "integrations"
  | "usage"
  | "secrets";

export interface OrganizationSettingsNavItem {
  id: OrganizationSettingsSection;
  label: string;
  Icon: LucideIcon;
}

export const ORGANIZATION_SETTINGS_NAV_ITEMS: OrganizationSettingsNavItem[] = [
  { id: "general", label: "General", Icon: Settings },
  { id: "workspaces", label: "Workspaces", Icon: LayoutGrid },
  { id: "members", label: "Members", Icon: User },
  { id: "api-keys", label: "API Keys", Icon: KeyRound },
  { id: "groups", label: "Groups", Icon: Users },
  { id: "roles", label: "Roles", Icon: Shield },
  { id: "integrations", label: "Integrations", Icon: Plug },
  { id: "usage", label: "Usage", Icon: Gauge },
  { id: "secrets", label: "Secrets", Icon: Key },
];

const IMPLEMENTED_ORGANIZATION_SETTINGS_SECTIONS = new Set<OrganizationSettingsSection>(["general", "workspaces"]);

export function isOrganizationSettingsComingSoon(item: OrganizationSettingsNavItem) {
  return !IMPLEMENTED_ORGANIZATION_SETTINGS_SECTIONS.has(item.id);
}
