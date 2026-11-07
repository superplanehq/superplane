import type { FactoriesFactoryWebhookStatus } from "@/api-client";
import { Button } from "@/components/ui/button";
import { githubRepoWebUrl } from "@/lib/githubHost";

const eventLabels: Record<string, string> = {
  issues: "issues",
  pull_request: "pull requests",
  pull_request_review: "pull request reviews",
  pull_request_review_comment: "pull request review comments",
  check_run: "check runs",
  check_suite: "check suites",
  status: "commit statuses",
};

type NoticeProps = {
  webhook: FactoriesFactoryWebhookStatus;
  retrying?: boolean;
  retryTimedOut?: boolean;
  canRetry?: boolean;
  onRetry: () => void;
  onRefresh: () => void;
};

function noticeDetails(webhook: FactoriesFactoryWebhookStatus) {
  const repository = webhook.repository?.trim() ?? "";
  return {
    repository,
    repoUrl: githubRepoWebUrl(repository, webhook.host),
    events: (webhook.eventTypes ?? []).map((event) => eventLabels[event] ?? event.replace(/_/g, " ")),
    automations: webhook.affectedAutomations?.filter((automation) => automation.name?.trim()) ?? [],
  };
}

export function GitHubWebhookFailureNotice({
  webhook,
  retrying = false,
  retryTimedOut = false,
  canRetry = true,
  onRetry,
  onRefresh,
}: NoticeProps) {
  if (webhook.state !== "failed" && !retrying && !retryTimedOut) {
    return null;
  }
  const { repository, repoUrl, events, automations } = noticeDetails(webhook);

  return (
    <section
      role="alert"
      className="rounded-lg border border-amber-300 bg-background px-3.5 py-3 dark:border-amber-700"
    >
      <h2 className="text-[15px] font-semibold tracking-[-0.02em] text-foreground">Webhook registration failed</h2>
      <p className="mt-1.5 text-[13px] text-muted-foreground">
        {webhook.message || "GitHub could not register this webhook. Check the repository settings, then try again."}
      </p>
      {repository ? <p className="mt-1.5 text-[13px] text-muted-foreground">Repository: {repository}</p> : null}
      {events.length > 0 ? <p className="mt-1 text-[13px] text-muted-foreground">Events: {events.join(", ")}</p> : null}
      {automations.length > 0 ? (
        <p className="mt-1 text-[13px] text-muted-foreground">
          Affected automations: {automations.map((automation) => automation.name).join(", ")}
        </p>
      ) : null}
      <WebhookRecoveryActions
        repoUrl={repoUrl}
        retrying={retrying}
        retryTimedOut={retryTimedOut}
        canRetry={canRetry}
        onRetry={onRetry}
        onRefresh={onRefresh}
      />
    </section>
  );
}

function WebhookRecoveryActions({
  repoUrl,
  retrying,
  retryTimedOut,
  canRetry,
  onRetry,
  onRefresh,
}: Omit<NoticeProps, "webhook"> & { repoUrl?: string }) {
  return (
    <div className="mt-3 flex flex-wrap items-center gap-2">
      {repoUrl ? (
        <Button asChild size="sm" variant="outline">
          <a href={`${repoUrl}/settings/hooks`} target="_blank" rel="noreferrer">
            Open repository webhooks
          </a>
        </Button>
      ) : null}
      {retryTimedOut ? (
        <Button type="button" size="sm" onClick={onRefresh}>
          Refresh status
        </Button>
      ) : (
        <Button type="button" size="sm" onClick={onRetry} disabled={retrying || !canRetry}>
          {retrying ? "Trying again" : "Try again"}
        </Button>
      )}
    </div>
  );
}
