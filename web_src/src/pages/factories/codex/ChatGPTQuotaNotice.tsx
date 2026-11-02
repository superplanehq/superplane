import type { CanvasesSubscriptionFailure } from "@/api-client";
import { Link } from "@/components/Link/link";
import { usePermissions } from "@/contexts/usePermissions";
import { factorySettingsSectionPath } from "../lib/factoryPagePaths";
import { ReadinessDot } from "../workOrders/ReadinessMark";

export function ChatGPTQuotaNotice({
  failure,
  organizationId,
  factoryKey,
}: {
  failure?: CanvasesSubscriptionFailure | null;
  organizationId?: string;
  factoryKey?: string;
}) {
  const { currentUserId } = usePermissions();
  if (failure?.reason !== "chatgpt_quota_exhausted") return null;
  const usageHref =
    currentUserId && currentUserId === failure.subscriptionOwnerId && organizationId && factoryKey
      ? factorySettingsSectionPath(organizationId, factoryKey, "account", "codex-usage")
      : undefined;
  const reset = failure.resetsAt ? new Date(failure.resetsAt) : undefined;
  const knownReset = reset && Number.isFinite(reset.getTime());
  return (
    <div
      role="alert"
      aria-label="ChatGPT limit reached"
      className="my-3 flex min-w-0 items-start gap-2 rounded-md border border-border bg-muted/40 px-3.5 py-2.5"
    >
      <ReadinessDot tone="blocked" className="mt-1.5" />
      <div className="min-w-0 space-y-1">
        <p className="text-[13px] font-medium text-foreground">ChatGPT limit reached</p>
        <p className="text-[12px] leading-4 text-muted-foreground">
          This run stopped because the ChatGPT subscription reached its usage limit.
        </p>
        <p className="text-[12px] leading-4 text-muted-foreground">
          {knownReset ? (
            <>
              Resets at <time dateTime={reset.toISOString()}>{reset.toLocaleString()}</time>.
            </>
          ) : (
            "The reset time is unavailable."
          )}
        </p>
        {usageHref ? (
          <Link href={usageHref} className="text-[12px] font-medium text-foreground underline underline-offset-2">
            View ChatGPT usage
          </Link>
        ) : (
          <p className="text-[12px] leading-4 text-muted-foreground">
            Ask the subscription owner to check ChatGPT usage. Start another attempt after the limit resets.
          </p>
        )}
      </div>
    </div>
  );
}

export function ChatGPTQuotaHistory({
  attempts = [],
  organizationId,
  factoryKey,
}: {
  attempts?: Array<{ runId: string; failure: CanvasesSubscriptionFailure }>;
  organizationId?: string;
  factoryKey?: string;
}) {
  return attempts.map((attempt) => (
    <section key={attempt.runId} aria-label="Earlier attempt">
      <p className="text-[12px] font-medium text-muted-foreground">Earlier attempt</p>
      <ChatGPTQuotaNotice failure={attempt.failure} organizationId={organizationId} factoryKey={factoryKey} />
    </section>
  ));
}
