import {
  Building2,
  CircleUser,
  Gauge,
  Key,
  KeyRound,
  Network,
  Plug,
  Settings,
  Shield,
  Terminal,
  User,
  Users,
} from "lucide-react";
import type { LucideIcon } from "lucide-react";
import type { PermissionCheck } from "./types";

export const COMMAND_SHORTCUT = "/";
export const DOCS_URL = "https://docs.superplane.com";

export const PUBLIC_TOP_LEVEL_SEGMENTS = new Set([
  "",
  "admin",
  "create",
  "invite",
  "install",
  "login",
  "setup",
  "signup",
  "welcome",
]);

export const ORGANIZATION_SETTINGS_LINKS: Array<{
  id: string;
  label: string;
  description: string;
  path: string;
  icon: LucideIcon;
  permission?: PermissionCheck;
}> = [
  {
    id: "general",
    label: "Settings",
    description: "Organization basics and identity",
    path: "settings/general",
    icon: Settings,
    permission: { resource: "org", action: "read" },
  },
  {
    id: "members",
    label: "Members",
    description: "Invite people and manage access",
    path: "settings/members",
    icon: User,
    permission: { resource: "members", action: "read" },
  },
  {
    id: "service-accounts",
    label: "API Keys",
    description: "Programmatic API access",
    path: "settings/service-accounts",
    icon: KeyRound,
    permission: { resource: "service_accounts", action: "read" },
  },
  {
    id: "groups",
    label: "Groups",
    description: "Organize members for permissions",
    path: "settings/groups",
    icon: Users,
    permission: { resource: "groups", action: "read" },
  },
  {
    id: "roles",
    label: "Roles",
    description: "Configure fine-grained access",
    path: "settings/roles",
    icon: Shield,
    permission: { resource: "roles", action: "read" },
  },
  {
    id: "integrations",
    label: "Integrations",
    description: "Connect external services",
    path: "settings/integrations",
    icon: Plug,
    permission: { resource: "integrations", action: "read" },
  },
  {
    id: "usage",
    label: "Usage",
    description: "Limits and tracked usage",
    path: "settings/billing",
    icon: Gauge,
    permission: { resource: "org", action: "read" },
  },
  {
    id: "secrets",
    label: "Secrets",
    description: "Encrypted values for workflows",
    path: "settings/secrets",
    icon: Key,
    permission: { resource: "secrets", action: "read" },
  },
  {
    id: "profile",
    label: "Profile",
    description: "Personal account settings",
    path: "settings/profile",
    icon: CircleUser,
  },
];

export const ADMIN_LINKS: Array<{
  id: string;
  label: string;
  description: string;
  href: string;
  icon: LucideIcon;
}> = [
  {
    id: "organizations",
    label: "Organizations",
    description: "Review organizations in this installation",
    href: "/admin",
    icon: Building2,
  },
  {
    id: "accounts",
    label: "Accounts",
    description: "Manage accounts and installation admins",
    href: "/admin/accounts",
    icon: Users,
  },
  {
    id: "settings",
    label: "Settings",
    description: "Installation network and SMTP settings",
    href: "/admin/settings",
    icon: Network,
  },
  {
    id: "runner-tasks",
    label: "Runner Tasks",
    description: "Inspect runner task activity",
    href: "/admin/runner-tasks",
    icon: Terminal,
  },
];
