import type { IntegrationsIntegrationDefinition } from "@/api-client/types.gen";
import { knownIntegrationTypeEntries } from "@/lib/integrationDisplayName";

import type { FactorySettingsScope } from "../../lib/factoryPagePaths";
import {
  FACTORY_SETTINGS_NAV_GROUPS,
  type FactorySettingsNavGroup,
  type FactorySettingsSection,
} from "./settingsNavItems";

export type FactorySettingsSearchResult = {
  id: string;
  title: string;
  breadcrumb: string[];
  scope: FactorySettingsScope;
  section: FactorySettingsSection;
  /** DOM id to scroll to after navigation (without #). */
  anchor?: string;
  keywords: string[];
};

const GROUP_LABEL: Record<FactorySettingsScope, string> = {
  account: "Account",
  workspace: "Workspace",
  organization: "Organization",
};

/** Static section-level entries: field labels, cards, and page titles users search for. */
export const FACTORY_SETTINGS_SEARCH_ENTRIES: FactorySettingsSearchResult[] = [
  // Account · Profile (includes security and access)
  entry("account", "profile", "Account", undefined, ["identity", "name", "avatar", "preferences"]),
  entry("account", "profile", "Identity", "account-redesign-identity", ["name", "primary email", "email", "avatar"]),
  entry("account", "profile", "Appearance", "account-redesign-appearance", ["theme", "light", "dark", "system"]),
  entry("account", "profile", "GitHub for Velocity", "account-redesign-velocity-github", [
    "github",
    "velocity",
    "pull requests",
  ]),
  entry("account", "profile", "Security", "account-redesign-security", [
    "sign in",
    "password",
    "token",
    "sso",
    "login",
  ]),
  entry("account", "profile", "Sign in methods", "account-redesign-signin", [
    "security",
    "password",
    "change password",
    "github",
    "google",
    "sso",
  ]),
  entry("account", "profile", "Personal tokens", "account-redesign-tokens", [
    "create token",
    "revoke",
    "cli",
    "api",
    "token",
  ]),

  // Account · Notifications
  entry("account", "notifications", "Notifications", "account-redesign-notifications", [
    "email",
    "send task emails",
    "task emails",
    "events",
    "mentions",
    "comments",
  ]),

  // Workspace · General
  entry("workspace", "general", "General", "factory-settings-general-form", ["name", "workspace key", "description"]),
  entry("workspace", "general", "Workspace key", "factory-settings-key", [
    "workspace key",
    "key",
    "task identifier",
    "task ids",
  ]),
  entry("workspace", "general", "Delete workspace", "factory-settings-danger-zone", [
    "danger zone",
    "delete workspace",
  ]),

  // Workspace · Repository
  entry("workspace", "repository", "Repository", "factory-settings-repository", [
    "github repository",
    "repo",
    "git",
    "github",
    "issue intake",
  ]),

  // Workspace · Automations
  entry("workspace", "automations", "Automations", undefined, ["new automation", "triggers", "lines", "canvas"]),

  // Workspace · Models
  entry("workspace", "models", "Models", undefined, [
    "llm",
    "ai",
    "allowlist",
    "anthropic",
    "openai",
    "openrouter",
    "byok",
    "use your keys",
  ]),

  // Organization · General
  entry("organization", "general", "General", undefined, ["name", "organization slug", "slug", "workspace url"]),

  // Organization · Members
  entry("organization", "members", "Members", undefined, ["invite", "invite link", "people", "team", "role"]),

  // Organization · Integrations (page-level; providers are added dynamically)
  entry("organization", "integrations", "Integrations", undefined, ["connect", "filter integrations", "integration"]),

  // Organization · API keys / Secrets / Spending
  entry("organization", "api-keys", "API keys", undefined, ["create api key", "token", "credentials", "programmatic"]),
  entry("organization", "secrets", "Secrets", undefined, ["create secret", "credentials", "env", "key-value"]),
  entry("organization", "spending", "Spending", undefined, ["billing", "usage", "credit", "hosted credit", "byok"]),
];

function entry(
  scope: FactorySettingsScope,
  section: FactorySettingsSection,
  title: string,
  anchor: string | undefined,
  keywords: string[],
): FactorySettingsSearchResult {
  const pageLabel =
    FACTORY_SETTINGS_NAV_GROUPS.find((group) => group.id === scope)?.items.find((item) => item.section === section)
      ?.label ?? title;

  const groupLabel = GROUP_LABEL[scope];
  const showPageLabel = pageLabel !== title && pageLabel !== groupLabel;

  return {
    id: anchor ? `section:${scope}:${section}:${anchor}` : `nav:${scope}:${section}`,
    title,
    breadcrumb: showPageLabel ? [groupLabel, pageLabel] : [groupLabel],
    scope,
    section,
    anchor,
    keywords,
  };
}

/** One search hit per available integration provider (e.g. Claude). */
export function factorySettingsIntegrationSearchEntries(
  integrations: Array<Pick<IntegrationsIntegrationDefinition, "name" | "label" | "description">>,
): FactorySettingsSearchResult[] {
  return integrations
    .filter((integration) => Boolean(integration.name))
    .map((integration) => {
      const name = integration.name!;
      const label = integration.label || name;
      return {
        id: `integration:${name}`,
        title: label,
        breadcrumb: ["Organization", "Integrations"],
        scope: "organization" as const,
        section: "integrations" as const,
        anchor: `integration-${name}`,
        keywords: [name, label, integration.description ?? ""].filter(Boolean),
      };
    });
}

export function buildFactorySettingsSearchIndex(options: {
  navGroups: FactorySettingsNavGroup[];
  integrations?: Array<Pick<IntegrationsIntegrationDefinition, "name" | "label" | "description">>;
}): FactorySettingsSearchResult[] {
  const allowed = new Set(
    options.navGroups.flatMap((group) => group.items.map((item) => `${item.scope}/${item.section}`)),
  );

  const staticEntries = FACTORY_SETTINGS_SEARCH_ENTRIES.filter((result) =>
    allowed.has(`${result.scope}/${result.section}`),
  );

  const providersByName = new Map<string, Pick<IntegrationsIntegrationDefinition, "name" | "label" | "description">>();
  for (const known of knownIntegrationTypeEntries()) {
    providersByName.set(known.name, known);
  }
  for (const integration of options.integrations ?? []) {
    if (!integration.name) {
      continue;
    }
    providersByName.set(integration.name, {
      name: integration.name,
      label: integration.label || providersByName.get(integration.name)?.label || integration.name,
      description: integration.description || providersByName.get(integration.name)?.description,
    });
  }

  const integrationEntries = factorySettingsIntegrationSearchEntries([...providersByName.values()]).filter((result) =>
    allowed.has(`${result.scope}/${result.section}`),
  );

  return [...staticEntries, ...integrationEntries];
}

/**
 * Cursor-style settings search: returns matching section and integration hits.
 * Prefers title matches, then breadcrumb/keyword matches.
 * Drops a page-level hit when a more specific section on the same page also matches.
 */
export function searchFactorySettings(
  entries: FactorySettingsSearchResult[],
  query: string,
): FactorySettingsSearchResult[] {
  const normalized = query.trim().toLowerCase();
  if (!normalized) {
    return [];
  }

  const scored = entries
    .map((result) => ({ result, score: scoreSearchResult(result, normalized) }))
    .filter((row) => row.score > 0)
    .sort((a, b) => b.score - a.score || a.result.title.localeCompare(b.result.title));

  return dropRedundantPageHits(
    scored.map((row) => row.result),
    normalized,
  );
}

/**
 * If Identity (Account › Profile) matches, do not also list Profile (Account).
 * Keep the page hit when the query matches the page title (Security + Sign in methods).
 * Page-level entries have no anchor; section/integration entries do.
 */
function dropRedundantPageHits(
  results: FactorySettingsSearchResult[],
  normalizedQuery: string,
): FactorySettingsSearchResult[] {
  const pagesWithSectionHits = new Set(
    results.filter((result) => Boolean(result.anchor)).map((result) => `${result.scope}/${result.section}`),
  );

  return results.filter((result) => {
    if (result.anchor) {
      return true;
    }
    if (!pagesWithSectionHits.has(`${result.scope}/${result.section}`)) {
      return true;
    }
    return result.title.toLowerCase().includes(normalizedQuery);
  });
}

function scoreSearchResult(result: FactorySettingsSearchResult, normalizedQuery: string): number {
  const title = result.title.toLowerCase();
  if (title === normalizedQuery) {
    return 100;
  }
  if (title.startsWith(normalizedQuery)) {
    return 80;
  }
  if (title.includes(normalizedQuery)) {
    return 60;
  }

  const haystack = [result.title, ...result.breadcrumb, ...result.keywords].join(" ").toLowerCase();
  if (haystack.includes(normalizedQuery)) {
    return 40;
  }

  const words = normalizedQuery.split(/\s+/).filter(Boolean);
  if (words.length > 1 && words.every((word) => haystack.includes(word))) {
    return 30;
  }

  return 0;
}

export function factorySettingsSearchResultPath(
  organizationId: string,
  factoryKey: string,
  result: FactorySettingsSearchResult,
  sectionPath: (organizationId: string, factoryKey: string, scope: FactorySettingsScope, section: string) => string,
): string {
  const path = sectionPath(organizationId, factoryKey, result.scope, result.section);
  if (!result.anchor) {
    return path;
  }
  return `${path}?section=${encodeURIComponent(result.anchor)}`;
}
