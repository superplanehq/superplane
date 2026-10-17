import { Button } from "@/components/ui/button";

const FALLBACK_MESSAGE = "GitHub did not create the webhook. SuperPlane cannot receive new issues. Try again.";

function githubRepositoryWebhooksUrl(repository: string | undefined): string | undefined {
  const name = repository?.trim() ?? "";
  if (!/^[A-Za-z0-9_.-]+\/[A-Za-z0-9_.-]+$/.test(name)) {
    return undefined;
  }
  return `https://github.com/${name}/settings/hooks`;
}

/** Tells the user why new GitHub issues do not become tasks, and how to try again. */
export function IntakeWebhookFailureNotice({
  message,
  repository,
  retrying = false,
  onRetry,
}: {
  message?: string;
  repository?: string;
  retrying?: boolean;
  onRetry: () => void;
}) {
  const webhooksUrl = githubRepositoryWebhooksUrl(repository);

  return (
    <section className="px-3 pb-3" data-testid="intake-webhook-failure">
      <div role="alert" className="rounded-lg border border-amber-300 bg-background px-3.5 py-3 dark:border-amber-700">
        <h2 className="text-[15px] font-semibold tracking-[-0.02em] text-foreground">New issues do not arrive</h2>
        <p className="mt-1.5 text-[13px] text-muted-foreground">{message?.trim() || FALLBACK_MESSAGE}</p>
        <div className="mt-3 flex flex-wrap items-center gap-2">
          {webhooksUrl ? (
            <Button asChild size="sm" variant="outline">
              <a href={webhooksUrl} target="_blank" rel="noreferrer">
                Open repository webhooks
              </a>
            </Button>
          ) : null}
          <Button
            type="button"
            size="sm"
            onClick={onRetry}
            disabled={retrying}
            data-testid="intake-webhook-failure-retry"
          >
            {retrying ? "Trying again" : "Try again"}
          </Button>
        </div>
      </div>
    </section>
  );
}
