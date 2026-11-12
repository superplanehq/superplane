import { useFactoryWebhookData } from "@/hooks/useFactoryWebhookData";

import { GitHubWebhookFailureNotice } from "./GitHubWebhookFailureNotice";

export function FactoryWebhookWarnings({
  organizationId,
  factoryId,
  automationId,
  canRetry,
}: {
  organizationId: string;
  factoryId: string;
  automationId?: string;
  canRetry: boolean;
}) {
  const webhooks = useFactoryWebhookData(organizationId, factoryId);
  const warnings = automationId
    ? webhooks.warnings.filter((webhook) =>
        webhook.affectedAutomations?.some((automation) => automation.id === automationId),
      )
    : webhooks.warnings;

  return warnings.map((webhook) => (
    <div className="px-3 pb-3" key={webhook.id}>
      <GitHubWebhookFailureNotice
        webhook={webhook}
        canRetry={canRetry}
        retrying={webhooks.retrying(webhook.id)}
        retryTimedOut={webhooks.retryTimedOut(webhook.id)}
        onRetry={() => void webhooks.retry(webhook)}
        onRefresh={webhooks.refresh}
      />
    </div>
  ));
}
