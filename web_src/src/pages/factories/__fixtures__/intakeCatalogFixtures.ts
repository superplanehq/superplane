import type { FactoriesFactoryIntakeCatalogEntry } from "@/api-client";

type IntakeSeed = Omit<FactoriesFactoryIntakeCatalogEntry, "available">;

const INTAKE_FEATURE_FLAGS: Record<string, string> = {
  "jira-issues": "factory_jira_intake",
  "productive-tasks": "factory_productive_intake",
  datadog: "factory_datadog_intake",
  "pagerduty-incidents": "factory_pagerduty_intake",
  "linear-issues": "factory_linear_intake",
};

const SEED: readonly IntakeSeed[] = [
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
  if (status === "deprecated") return false;
  const flag = INTAKE_FEATURE_FLAGS[key];
  if (!flag) return true;
  return granted.includes(key);
}

export function storybookIntakeCatalogEntries(granted: readonly string[] = []): FactoriesFactoryIntakeCatalogEntry[] {
  return SEED.map((entry) => ({
    ...entry,
    available: intakeVisible(entry.key ?? "", entry.status ?? "planned", granted),
  }));
}
