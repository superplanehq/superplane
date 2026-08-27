export type VcsHostId = "github" | "gitlab";
/** Matches SuperPlane integration registry names (Claude = `claude`, not anthropic). */
export type IntegrationId = "github" | "gitlab" | "claude" | "cursor" | "openai" | "openrouter" | "linear" | "jira";
export type AgentHarnessId = "claude-code" | "cursor" | "codex";
export type IssuesChoiceId = "vcs" | "linear" | "jira" | "skip";
export type WizardStepId = "vcs" | "repo" | "issues" | "agent" | "name";

export type IntegrationOption = {
  id: IntegrationId;
  label: string;
  detail: string;
  /** When true, shown but not connectable in this wireframe. */
  soon?: boolean;
};

export const VCS_OPTIONS: IntegrationOption[] = [
  { id: "github", label: "GitHub", detail: "Connect GitHub to list repositories and open pull requests." },
  { id: "gitlab", label: "GitLab", detail: "Connect GitLab to list repositories and open merge requests.", soon: true },
];

export const AGENT_OPTIONS: IntegrationOption[] = [
  {
    id: "claude",
    label: "Anthropic",
    detail: "Connect an Anthropic API key for Claude Code.",
  },
  {
    id: "openai",
    label: "OpenAI",
    detail: "Connect an OpenAI API key for Codex and OpenAI models.",
  },
  {
    id: "openrouter",
    label: "OpenRouter",
    detail: "Connect an OpenRouter API key for many model providers.",
  },
  {
    id: "cursor",
    label: "Cursor",
    detail: "Cloud agent that changes the app repository and opens pull requests.",
    soon: true,
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

export const WIZARD_STEPS = [
  {
    id: "vcs" as const,
    label: "VCS",
    purpose: "Choose GitHub or GitLab and connect it. Agents use this host for the app repository.",
  },
  {
    id: "repo" as const,
    label: "Repository",
    purpose: "Pick the app repository. SuperPlane analyzes that codebase. Agents change it and open pull requests.",
  },
  {
    id: "issues" as const,
    label: "Issues",
    purpose:
      "Optional. Point SuperPlane at a backlog so it can find small work that agents can solve. Or skip and create work orders yourself.",
  },
  {
    id: "agent" as const,
    label: "Agent",
    purpose: "SuperPlane will run the agent on this workspace. Work starts only after you approve a ticket.",
  },
  {
    id: "name" as const,
    label: "Name",
    purpose: "Name the workspace for the app or product area you want to improve.",
  },
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
    case "openrouter":
      return "OpenRouter";
    case "linear":
      return "Linear";
    case "jira":
      return "Jira";
  }
}
