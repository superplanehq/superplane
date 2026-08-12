export type VcsHostId = "github" | "gitlab";
/** Matches SuperPlane integration registry names (Claude = `claude`, not anthropic). */
export type IntegrationId = "github" | "gitlab" | "claude" | "cursor" | "openai" | "linear" | "jira";
export type AgentHarnessId = "claude-code" | "cursor" | "codex";
export type IssuesChoiceId = "vcs" | "linear" | "jira" | "skip";

export type IntegrationOption = {
  id: IntegrationId;
  label: string;
  detail: string;
  /** When true, shown but not connectable in this wireframe. */
  soon?: boolean;
};

export const VCS_OPTIONS: IntegrationOption[] = [
  { id: "github", label: "GitHub", detail: "Recommended. Connect the app repository on GitHub." },
  { id: "gitlab", label: "GitLab", detail: "Connect the app repository on GitLab." },
];

export const AGENT_OPTIONS: {
  id: AgentHarnessId;
  label: string;
  integrationId: IntegrationId;
  detail: string;
  recommended?: boolean;
}[] = [
  {
    id: "claude-code",
    label: "Claude Code",
    integrationId: "claude",
    detail: "Cloud agent that changes the app repository and opens pull requests.",
    recommended: true,
  },
  {
    id: "cursor",
    label: "Cursor",
    integrationId: "cursor",
    detail: "Cloud agent that changes the app repository and opens pull requests.",
  },
  {
    id: "codex",
    label: "Codex",
    integrationId: "openai",
    detail: "Cloud agent (OpenAI) that changes the app repository and opens pull requests.",
  },
];

/** Large enough that Storybook can demonstrate search filtering. */
export const FIXTURE_REPOS: Record<VcsHostId, string[]> = {
  github: [
    "acme/api",
    "acme/web",
    "acme/payments-service",
    "acme/billing",
    "acme/checkout",
    "acme/auth-service",
    "acme/notifications",
    "acme/analytics",
    "acme/mobile-ios",
    "acme/mobile-android",
    "acme/docs",
    "acme/infra",
    "acme/design-system",
    "acme/cli",
    "acme/worker",
    "acme/edge-gateway",
    "acme/data-pipeline",
    "acme/ml-serving",
  ],
  gitlab: [
    "acme-ops/backend",
    "acme-ops/frontend",
    "acme-ops/platform",
    "acme-ops/deploy",
    "acme-ops/observability",
    "acme-ops/security",
    "acme-ops/runbooks",
    "acme-ops/terraform",
    "acme-ops/helm-charts",
    "acme-ops/ci-templates",
    "acme-ops/secrets",
    "acme-ops/dns",
    "acme-ops/identity",
    "acme-ops/billing-ops",
    "acme-ops/support-tools",
    "acme-ops/status-page",
    "acme-ops/incident-bot",
    "acme-ops/audit",
  ],
};

const FIXTURE_ISSUE_COUNT_OVERRIDES: Record<string, number> = {
  "acme/api": 47,
  "acme/web": 12,
  "acme/payments-service": 31,
  "acme-ops/backend": 22,
  "acme-ops/frontend": 9,
  "acme-ops/platform": 18,
};

export function fixtureIssueCount(repo: string): number {
  if (FIXTURE_ISSUE_COUNT_OVERRIDES[repo] !== undefined) {
    return FIXTURE_ISSUE_COUNT_OVERRIDES[repo];
  }
  let hash = 0;
  for (let i = 0; i < repo.length; i += 1) {
    hash = (hash * 31 + repo.charCodeAt(i)) >>> 0;
  }
  return 5 + (hash % 40);
}

export const FIXTURE_INVITE_URL = "https://app.superplane.dev/invite/storybook-demo-token";

export const RAIL_STEPS = [
  { id: "name", label: "Workspace", detail: "Name the workspace for continuous AI work" },
  { id: "repo", label: "Version control", detail: "App repository agents analyze and change" },
  { id: "issues", label: "Issues", detail: "Backlog SuperPlane can turn into work" },
  { id: "agent", label: "Coding agent", detail: "Runs work in the app repository" },
] as const;

export function vcsLabel(host: VcsHostId) {
  return host === "github" ? "GitHub" : "GitLab";
}

export function integrationLabel(id: IntegrationId) {
  switch (id) {
    case "github":
      return "GitHub";
    case "gitlab":
      return "GitLab";
    case "claude":
      return "Claude";
    case "cursor":
      return "Cursor";
    case "openai":
      return "OpenAI";
    case "linear":
      return "Linear";
    case "jira":
      return "Jira";
  }
}
