import type { IntakeCatalogAvailability } from "@/hooks/useIntakeCatalogAvailability";
import { intakeSurfaceEntries, intakeSurfaceState, type IntakeCatalogItem } from "@/lib/intakeCatalog";
import {
  FEATURE_FACTORY_DATADOG_INTAKE,
  FEATURE_FACTORY_JIRA_INTAKE,
  FEATURE_FACTORY_LINEAR_INTAKE,
  FEATURE_FACTORY_PAGERDUTY_INTAKE,
  FEATURE_FACTORY_PRODUCTIVE_INTAKE,
} from "@/lib/experimentalFeatures";

/** Intake keys whose visibility is an organization feature flag. */
const INTAKE_FEATURE_FLAGS: Record<string, string> = {
  "jira-issues": FEATURE_FACTORY_JIRA_INTAKE,
  "productive-tasks": FEATURE_FACTORY_PRODUCTIVE_INTAKE,
  datadog: FEATURE_FACTORY_DATADOG_INTAKE,
  "pagerduty-incidents": FEATURE_FACTORY_PAGERDUTY_INTAKE,
  "linear-issues": FEATURE_FACTORY_LINEAR_INTAKE,
};

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

function intakeVisible(key: string, status: string, granted: readonly string[]): boolean {
  if (status === "deprecated") {
    return false;
  }
  const flag = INTAKE_FEATURE_FLAGS[key];
  if (!flag) {
    return true;
  }
  return granted.includes(key);
}

/**
 * The seeded catalog as one company sees it. Keys in `granted` are the intakes
 * whose feature flag is on for that company. `overrides` change single entries.
 */
export function seededIntakeCatalog(
  granted: readonly string[] = [],
  overrides: Record<string, Partial<IntakeCatalogItem>> = {},
): IntakeCatalogItem[] {
  const items = SEED.map((item) => ({
    ...item,
    available: intakeVisible(item.key, item.status, granted),
    ...overrides[item.key],
  }));
  const extra = Object.entries(overrides)
    .filter(([key]) => !SEED.some((item) => item.key === key))
    .map(([key, item]) => {
      const status = item.status ?? "planned";
      return {
        key,
        name: key,
        category: "issue_tracking",
        status,
        available: item.available ?? intakeVisible(key, status, granted),
        ...item,
      };
    });
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
