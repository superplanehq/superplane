import type { IntakeCatalogAvailability } from "@/hooks/useIntakeCatalogAvailability";
import { intakeSurfaceEntries, intakeSurfaceState, type IntakeCatalogItem } from "@/lib/intakeCatalog";

const SEED: readonly Omit<IntakeCatalogItem, "available">[] = [
  { key: "github-issues", name: "GitHub issues", category: "issue_tracking", status: "ga" },
  { key: "jira-issues", name: "Jira issues", category: "issue_tracking", status: "beta" },
  { key: "linear-issues", name: "Linear issues", category: "issue_tracking", status: "planned" },
  { key: "notion", name: "Notion", category: "issue_tracking", status: "planned" },
  { key: "productive-tasks", name: "Productive tasks", category: "issue_tracking", status: "beta" },
  { key: "sentry-exceptions", name: "Sentry exceptions", category: "error_tracking", status: "ga" },
  { key: "datadog", name: "Datadog errors", category: "error_tracking", status: "beta" },
  { key: "pagerduty-incidents", name: "PagerDuty incidents", category: "incident_management", status: "alpha" },
  { key: "dependabot-alerts", name: "Dependabot alerts", category: "security_alerts", status: "ga" },
  { key: "github", name: "GitHub", category: "repository_provider", status: "ga" },
  { key: "gitlab", name: "GitLab", category: "repository_provider", status: "planned" },
  { key: "bitbucket", name: "Bitbucket", category: "repository_provider", status: "planned" },
];

/**
 * The seeded catalog as one company sees it. Generally available entries are
 * available. Keys in `granted` are available too, as if an admin added the
 * company. `overrides` change single entries.
 */
export function seededIntakeCatalog(
  granted: readonly string[] = [],
  overrides: Record<string, Partial<IntakeCatalogItem>> = {},
): IntakeCatalogItem[] {
  const items = SEED.map((item) => ({
    ...item,
    available: item.status === "ga" || granted.includes(item.key),
    ...overrides[item.key],
  }));
  const extra = Object.entries(overrides)
    .filter(([key]) => !SEED.some((item) => item.key === key))
    .map(([key, item]) => ({
      key,
      name: key,
      category: "issue_tracking",
      status: "planned",
      available: false,
      ...item,
    }));
  return [...items, ...extra];
}

/** The value that `useIntakeCatalogAvailability` returns for a loaded catalog. */
export function intakeCatalogAvailability(catalog: readonly IntakeCatalogItem[] | null): IntakeCatalogAvailability {
  if (!catalog) {
    return { catalog: [], loaded: false, loading: true, error: false, stateOf: () => undefined, entriesFor: () => [] };
  }
  return {
    catalog: [...catalog],
    loaded: true,
    loading: false,
    error: false,
    stateOf: (key) => intakeSurfaceState(catalog.find((item) => item.key === key)),
    entriesFor: (surface) => intakeSurfaceEntries(surface, catalog),
  };
}
