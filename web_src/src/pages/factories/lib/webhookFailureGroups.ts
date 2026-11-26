import type { OrganizationsIntegrationWebhookFailure } from "@/api-client";

export type GitHubWebhookFailure = OrganizationsIntegrationWebhookFailure & {
  integrationId: string;
  host: string;
};

export interface RepositoryWebhookFailures {
  key: string;
  repository: string;
  host: string;
  failures: GitHubWebhookFailure[];
}

export function groupWebhookFailuresByRepository(failures: GitHubWebhookFailure[]): RepositoryWebhookFailures[] {
  const groups = new Map<string, RepositoryWebhookFailures>();
  for (const failure of failures) {
    const repositories = failure.repositories?.length ? failure.repositories : [""];
    for (const repository of repositories) {
      const key = `${failure.integrationId}/${failure.host}/${repository}`;
      let group = groups.get(key);
      if (!group) {
        group = { key, repository, host: failure.host, failures: [] };
        groups.set(key, group);
      }
      group.failures.push(failure);
    }
  }
  return [...groups.values()];
}

const EVENT_LABELS: Record<string, string> = {
  issues: "Issues",
  issue_comment: "Issue comments",
  pull_request: "Pull requests",
  pull_request_review: "Pull request reviews",
  pull_request_review_comment: "Review comments",
  check_run: "Check runs",
  check_suite: "Check suites",
  status: "Commit statuses",
};

export function webhookEventLabel(event: string): string {
  return EVENT_LABELS[event] ?? event.replace(/_/g, " ").replace(/^./, (letter: string) => letter.toUpperCase());
}
