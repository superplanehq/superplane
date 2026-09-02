import type { LucideIcon } from "lucide-react";
import {
  BarChart3,
  Bell,
  Blocks,
  CircleUser,
  Cpu,
  Grid3x3,
  Key,
  KeyRound,
  Plug,
  Settings,
  Shield,
  Users,
  Workflow,
} from "lucide-react";

import type { FactorySettingsScope } from "../../lib/factoryPagePaths";

export type FactorySettingsSection =
  | "general"
  | "profile"
  | "security"
  | "notifications"
  | "repository"
  | "automations"
  | "models"
  | "spending"
  | "members"
  | "integrations"
  | "api-keys"
  | "secrets";

export interface FactorySettingsNavItem {
  id: string;
  label: string;
  Icon: LucideIcon;
  scope: FactorySettingsScope;
  section: FactorySettingsSection;
}

export interface FactorySettingsNavGroup {
  id: FactorySettingsScope;
  label: string;
  items: FactorySettingsNavItem[];
}

export const FACTORY_SETTINGS_NAV_GROUPS: FactorySettingsNavGroup[] = [
  {
    id: "account",
    label: "Account",
    items: [
      { id: "account-profile", label: "Profile", Icon: CircleUser, scope: "account", section: "profile" },
      { id: "account-security", label: "Security", Icon: Shield, scope: "account", section: "security" },
      { id: "account-notifications", label: "Notifications", Icon: Bell, scope: "account", section: "notifications" },
    ],
  },
  {
    id: "workspace",
    label: "Workspace",
    items: [
      { id: "workspace-general", label: "General", Icon: Grid3x3, scope: "workspace", section: "general" },
      { id: "workspace-repository", label: "Repository", Icon: Blocks, scope: "workspace", section: "repository" },
      { id: "workspace-automations", label: "Automations", Icon: Workflow, scope: "workspace", section: "automations" },
      { id: "workspace-models", label: "Models", Icon: Cpu, scope: "workspace", section: "models" },
      { id: "workspace-spending", label: "Spending", Icon: BarChart3, scope: "workspace", section: "spending" },
    ],
  },
  {
    id: "organization",
    label: "Organization",
    items: [
      { id: "organization-general", label: "General", Icon: Settings, scope: "organization", section: "general" },
      { id: "organization-members", label: "Members", Icon: Users, scope: "organization", section: "members" },
      {
        id: "organization-integrations",
        label: "Integrations",
        Icon: Plug,
        scope: "organization",
        section: "integrations",
      },
      { id: "organization-api-keys", label: "API keys", Icon: KeyRound, scope: "organization", section: "api-keys" },
      { id: "organization-secrets", label: "Secrets", Icon: Key, scope: "organization", section: "secrets" },
      { id: "organization-spending", label: "Spending", Icon: BarChart3, scope: "organization", section: "spending" },
    ],
  },
];

export const FACTORY_SETTINGS_NAV_ITEMS = FACTORY_SETTINGS_NAV_GROUPS.flatMap((group) => group.items);

export function factorySettingsRouteFromPathname(pathname: string) {
  const segments = pathname.split("/").filter(Boolean);
  const settingsIndex = segments.lastIndexOf("settings");
  if (settingsIndex === -1) {
    return undefined;
  }

  const scope = segments[settingsIndex + 1];
  const section = segments[settingsIndex + 2];
  return FACTORY_SETTINGS_NAV_ITEMS.find((item) => item.scope === scope && item.section === section);
}
