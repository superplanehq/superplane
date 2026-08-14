import { AGENT_OPTIONS, vcsLabel } from "../../onboarding/onboardingFixtures";
import type { GitProvider, OnboardingRepo, WorkspaceConnections } from "../../onboarding/onboardingMocks";
import { providerLabel } from "../../onboarding/onboardingMocks";

export type { WorkspaceConnections };

export const STORYBOOK_APP_REPO: OnboardingRepo = {
  id: "gh-acme-api",
  name: "api",
  org: "acme",
  provider: "github",
};

export const STORYBOOK_WORKSPACE_CONNECTIONS: WorkspaceConnections = {
  repos: [STORYBOOK_APP_REPO],
  issuesChoice: "vcs",
  issuesRepo: "acme/api",
  agent: "claude-code",
};

export function appRepositoryFullName(repo: OnboardingRepo | undefined): string {
  if (!repo) {
    return "";
  }
  return `${repo.org}/${repo.name}`;
}

export function appRepositorySummary(connections: WorkspaceConnections): string {
  const repo = connections.repos[0];
  if (!repo) {
    return "No app repository.";
  }
  return `${providerLabel(repo.provider)} · ${appRepositoryFullName(repo)}`;
}

export function backlogSummary(connections: WorkspaceConnections): string {
  if (connections.issuesChoice === "skip") {
    return "No backlog. Create each work order yourself.";
  }
  if (connections.issuesChoice === "linear") {
    return "Linear";
  }
  if (connections.issuesChoice === "jira") {
    return "Jira";
  }
  if (connections.issuesChoice === "vcs") {
    const repo = connections.issuesRepo ?? appRepositoryFullName(connections.repos[0]);
    const provider = connections.repos[0]?.provider;
    const host = provider ? `${vcsLabel(provider)} Issues` : "Issues";
    return repo ? `${host} · ${repo}` : host;
  }
  return "No backlog selected.";
}

export function agentSummary(connections: WorkspaceConnections): string {
  return AGENT_OPTIONS.find((option) => option.id === connections.agent)?.label ?? "No coding agent.";
}

export function repoFromFullName(fullName: string, provider: GitProvider): OnboardingRepo {
  const [org, name] = fullName.split("/");
  return {
    id: `${provider === "github" ? "gh" : "gl"}-${org}-${name}`,
    org: org ?? "",
    name: name ?? fullName,
    provider,
  };
}
