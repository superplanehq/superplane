import type { LucideIcon } from "lucide-react";
import {
  BarChart3,
  Bell,
  Blocks,
  CircleDollarSign,
  CircleUser,
  Cpu,
  Grid3x3,
  Key,
  KeyRound,
  Plug,
  Receipt,
  Settings,
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
  | "usage"
  | "billing"
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
  /**
   * Searchable page content: field labels, card titles, and short aliases.
   * Find settings matches these in addition to the nav label and group label.
   */
  keywords?: string[];
  /**
   * When set, hide this item unless the current user can perform the action.
   * Account pages stay ungated.
   */
  permission?: { resource: string; action: string };
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
      {
        id: "account-profile",
        label: "Account",
        Icon: CircleUser,
        scope: "account",
        section: "profile",
        keywords: [
          "identity",
          "name",
          "primary email",
          "avatar",
          "appearance",
          "theme",
          "light",
          "dark",
          "system",
          "github for velocity",
          "velocity",
          "pull requests",
          "preferences",
          "profile",
          "security",
          "password",
          "sso",
          "token",
          "access",
          "github",
          "google",
          "sign in methods",
          "personal tokens",
          "cli",
          "api",
        ],
      },
      {
        id: "account-notifications",
        label: "Notifications",
        Icon: Bell,
        scope: "account",
        section: "notifications",
        keywords: [
          "email",
          "send task emails",
          "task emails",
          "events",
          "mentions",
          "comments",
          "task owner",
          "artifacts",
          "review requests",
          "status changes",
          "workspaces",
          "all workspaces",
          "selected workspaces",
        ],
      },
    ],
  },
  {
    id: "workspace",
    label: "Workspace",
    items: [
      {
        id: "workspace-general",
        label: "General",
        Icon: Grid3x3,
        scope: "workspace",
        section: "general",
        permission: { resource: "factories", action: "update" },
        keywords: [
          "name",
          "workspace key",
          "key",
          "slug",
          "description",
          "task identifier",
          "task ids",
          "danger zone",
          "delete workspace",
        ],
      },
      {
        id: "workspace-repository",
        label: "Repository",
        Icon: Blocks,
        scope: "workspace",
        section: "repository",
        permission: { resource: "factories", action: "update" },
        keywords: [
          "github repository",
          "repo",
          "git",
          "github",
          "issue intake",
          "pull request automations",
          "save repository",
        ],
      },
      {
        id: "workspace-automations",
        label: "Automations",
        Icon: Workflow,
        scope: "workspace",
        section: "automations",
        permission: { resource: "factories", action: "update" },
        keywords: ["new automation", "triggers", "lines", "canvas", "canvases"],
      },
      {
        id: "workspace-models",
        label: "Models",
        Icon: Cpu,
        scope: "workspace",
        section: "models",
        permission: { resource: "factories", action: "update" },
        keywords: [
          "llm",
          "ai",
          "allowlist",
          "anthropic",
          "openai",
          "openrouter",
          "superplane-hosted models",
          "use your keys",
          "byok",
          "organization list",
          "save models",
        ],
      },
    ],
  },
  {
    id: "organization",
    label: "Organization",
    items: [
      {
        id: "organization-general",
        label: "General",
        Icon: Settings,
        scope: "organization",
        section: "general",
        permission: { resource: "org", action: "read" },
        keywords: ["name", "organization slug", "slug", "workspace url"],
      },
      {
        id: "organization-members",
        label: "Members",
        Icon: Users,
        scope: "organization",
        section: "members",
        permission: { resource: "members", action: "read" },
        keywords: [
          "invite",
          "invite link",
          "copy link",
          "people",
          "team",
          "role",
          "email",
          "name",
          "remove",
          "owner",
          "admin",
        ],
      },
      {
        id: "organization-integrations",
        label: "Integrations",
        Icon: Plug,
        scope: "organization",
        section: "integrations",
        permission: { resource: "integrations", action: "read" },
        keywords: ["connect", "filter integrations", "github", "slack", "request it"],
      },
      {
        id: "organization-models",
        label: "LLM Models",
        Icon: Cpu,
        scope: "organization",
        section: "models",
        permission: { resource: "org", action: "read" },
        keywords: [
          "llm",
          "ai",
          "allowlist",
          "claude",
          "openai",
          "openrouter",
          "anthropic",
          "use your keys",
          "byok",
          "your keys",
          "select models",
        ],
      },
      {
        id: "organization-api-keys",
        label: "API keys",
        Icon: KeyRound,
        scope: "organization",
        section: "api-keys",
        permission: { resource: "api_keys", action: "read" },
        keywords: [
          "create api key",
          "programmatic access",
          "name",
          "description",
          "role",
          "access",
          "expiration",
          "organization-wide",
          "selected apps",
          "credentials",
          "token",
        ],
      },
      {
        id: "organization-secrets",
        label: "Secrets",
        Icon: Key,
        scope: "organization",
        section: "secrets",
        permission: { resource: "secrets", action: "read" },
        keywords: ["create secret", "secret name", "key-value pairs", "credentials", "env"],
      },
      {
        id: "organization-billing",
        label: "Billing",
        Icon: CircleDollarSign,
        scope: "organization",
        section: "billing",
        permission: { resource: "org", action: "read" },
        keywords: [
          "billing",
          "credit",
          "hosted credit",
          "invoices",
          "remaining hosted credit",
          "superplane grant",
          "purchased hosted credit",
          "buy more",
        ],
      },
      {
        id: "organization-usage",
        label: "Usage",
        Icon: Receipt,
        scope: "organization",
        section: "usage",
        permission: { resource: "org", action: "read" },
        keywords: ["usage", "cost", "tokens", "vm time", "task spend", "your keys", "model", "machine"],
      },
      {
        id: "organization-spending",
        label: "Spending",
        Icon: BarChart3,
        scope: "organization",
        section: "spending",
        permission: { resource: "org", action: "read" },
        keywords: [
          "usage",
          "cost",
          "tokens",
          "vm time",
          "estimated spend",
          "by model",
          "by machine type",
          "model usage",
          "vm usage",
        ],
      },
    ],
  },
];

export const FACTORY_SETTINGS_NAV_ITEMS = FACTORY_SETTINGS_NAV_GROUPS.flatMap((group) => group.items);

/**
 * Filters settings nav groups for the Find settings field.
 * Matches item labels, indexed page content, and group labels (group match keeps all items).
 */
export function filterFactorySettingsNavGroups(
  groups: FactorySettingsNavGroup[],
  query: string,
): FactorySettingsNavGroup[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return groups;
  }

  return groups
    .map((group) => {
      if (group.label.toLowerCase().includes(normalized)) {
        return group;
      }

      const items = group.items.filter((item) => navItemMatchesQuery(item, normalized));
      return { ...group, items };
    })
    .filter((group) => group.items.length > 0);
}

function navItemSearchText(item: FactorySettingsNavItem): string {
  return [item.label, ...(item.keywords ?? [])].join(" ").toLowerCase();
}

function navItemMatchesQuery(item: FactorySettingsNavItem, normalizedQuery: string): boolean {
  const haystack = navItemSearchText(item);
  if (haystack.includes(normalizedQuery)) {
    return true;
  }

  const words = normalizedQuery.split(/\s+/).filter(Boolean);
  return words.length > 1 && words.every((word) => haystack.includes(word));
}

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

export function filterFactorySettingsNavGroupsByPermission(
  groups: FactorySettingsNavGroup[],
  canAct: (resource: string, action: string) => boolean,
  permissionsLoading: boolean,
): FactorySettingsNavGroup[] {
  if (permissionsLoading) {
    return groups;
  }

  return groups
    .map((group) => ({
      ...group,
      items: group.items.filter((item) => {
        if (!item.permission) {
          return true;
        }
        return canAct(item.permission.resource, item.permission.action);
      }),
    }))
    .filter((group) => group.items.length > 0);
}
