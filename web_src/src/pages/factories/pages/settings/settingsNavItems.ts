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
  /**
   * Searchable page content: field labels, card titles, and short aliases.
   * Find settings matches these in addition to the nav label and group label.
   */
  keywords?: string[];
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
        keywords: ["new automation", "triggers", "lines", "canvas", "canvases"],
      },
      {
        id: "workspace-models",
        label: "Models",
        Icon: Cpu,
        scope: "workspace",
        section: "models",
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
      {
        id: "workspace-spending",
        label: "Spending",
        Icon: BarChart3,
        scope: "workspace",
        section: "spending",
        keywords: [
          "billing",
          "usage",
          "cost",
          "credit",
          "tokens",
          "vm time",
          "estimated spend",
          "hosted spend limit",
          "no limit",
          "limit in usd",
          "remaining hosted credit",
          "hosted billed spend",
          "by model",
          "by machine type",
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
        keywords: ["name", "organization slug", "slug", "workspace url"],
      },
      {
        id: "organization-members",
        label: "Members",
        Icon: Users,
        scope: "organization",
        section: "members",
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
        keywords: ["connect", "filter integrations", "github", "slack", "request it"],
      },
      {
        id: "organization-api-keys",
        label: "API keys",
        Icon: KeyRound,
        scope: "organization",
        section: "api-keys",
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
        keywords: ["create secret", "secret name", "key-value pairs", "credentials", "env"],
      },
      {
        id: "organization-spending",
        label: "Spending",
        Icon: BarChart3,
        scope: "organization",
        section: "spending",
        keywords: [
          "billing",
          "usage",
          "cost",
          "credit",
          "tokens",
          "vm time",
          "estimated spend",
          "remaining hosted credit",
          "superplane grant",
          "purchased hosted credit",
          "add hosted credit",
          "polar invoices",
          "manage invoices",
          "use your keys",
          "byok",
          "anthropic",
          "openai",
          "openrouter",
          "by model",
          "by machine type",
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
