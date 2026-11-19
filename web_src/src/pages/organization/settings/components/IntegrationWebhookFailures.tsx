import { useState, type ReactNode } from "react";
import type { OrganizationsIntegration, OrganizationsIntegrationWebhookFailure } from "@/api-client";
import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useIntegrationResources, useRetryIntegrationWebhook } from "@/hooks/useIntegrations";
import { githubHostFromIntegration, githubRepoWebUrl } from "@/lib/githubHost";
import { webhookSetupInProgress } from "@/lib/webhookSetup";

export function WebhookFailureNotice({
  failure,
  host,
  canRetry,
  retrying = false,
  retryError = false,
  onRetry,
  children,
}: {
  failure: OrganizationsIntegrationWebhookFailure;
  host?: string;
  canRetry: boolean;
  retrying?: boolean;
  retryError?: boolean;
  onRetry: () => void;
  children?: ReactNode;
}) {
  const setupInProgress = retrying || webhookSetupInProgress(failure.state);
  return (
    <section role="alert" className="rounded-lg border border-amber-300 bg-background p-4 dark:border-amber-700">
      <h2 className="font-semibold">{setupInProgress ? "Retrying webhook setup" : "Webhook setup failed"}</h2>
      <p className="mt-1 text-sm text-muted-foreground">{failure.message}</p>
      <p className="mt-2 text-sm">Events: {failure.events?.join(", ") || "Not available"}</p>
      <div className="mt-3 flex flex-wrap items-center gap-2">
        {failure.repositories?.map((repository) => {
          const url = githubRepoWebUrl(repository, host);
          return url ? (
            <Button key={repository} asChild size="sm" variant="outline">
              <a href={`${url}/settings/hooks`} target="_blank" rel="noreferrer">
                Open {repository} webhooks
              </a>
            </Button>
          ) : null;
        })}
        <Button type="button" size="sm" disabled={!canRetry || setupInProgress} onClick={onRetry}>
          {setupInProgress ? "Trying again" : "Try again"}
        </Button>
      </div>
      {retryError ? (
        <p className="mt-2 text-sm text-destructive">SuperPlane could not retry webhook setup. Try again.</p>
      ) : null}
      {children}
    </section>
  );
}

function RepositoryWebhooks({
  organizationId,
  integrationId,
  repository,
  event,
}: {
  organizationId: string;
  integrationId: string;
  repository: string;
  event: string;
}) {
  const [expanded, setExpanded] = useState(false);
  const {
    data: webhooks = [],
    isFetching,
    isError,
    refetch,
  } = useIntegrationResources(
    organizationId,
    integrationId,
    "repository_webhook",
    { repository, event },
    { enabled: expanded },
  );
  return (
    <div className="mt-3 text-sm">
      <Button type="button" variant="outline" size="sm" onClick={() => setExpanded(!expanded)}>
        {expanded ? "Hide existing webhooks" : "Show existing webhooks"} ({repository})
      </Button>
      {expanded ? (
        <div className="mt-3 space-y-2">
          <p>Open an unused webhook and delete it in GitHub. Then try setup again.</p>
          {isFetching ? <p role="status">Loading webhooks...</p> : null}
          {isError ? (
            <p role="alert">SuperPlane could not list webhooks. Open repository webhooks to review them.</p>
          ) : null}
          {!isFetching && !isError && webhooks.length === 0 ? <p>No matching webhooks found.</p> : null}
          <ul className="space-y-1 break-all">
            {webhooks.map((webhook) => (
              <li key={webhook.id}>
                <a className="text-link underline" href={webhook.url} target="_blank" rel="noreferrer">
                  {webhook.name}
                </a>
              </li>
            ))}
          </ul>
          <Button type="button" size="sm" variant="outline" disabled={isFetching} onClick={() => void refetch()}>
            Refresh list
          </Button>
        </div>
      ) : null}
    </div>
  );
}

function IntegrationWebhookFailure({
  organizationId,
  integration,
  failure,
  canRetry,
}: {
  organizationId: string;
  integration: OrganizationsIntegration;
  failure: OrganizationsIntegrationWebhookFailure;
  canRetry: boolean;
}) {
  const integrationId = integration.metadata?.id ?? "";
  const retry = useRetryIntegrationWebhook(organizationId, integrationId, failure.id ?? "");
  return (
    <WebhookFailureNotice
      failure={failure}
      host={githubHostFromIntegration(integration)}
      canRetry={canRetry}
      retrying={retry.isPending}
      retryError={retry.isError}
      onRetry={() => retry.mutate()}
    >
      {failure.limitEvent
        ? failure.repositories?.map((repository) => (
            <RepositoryWebhooks
              key={repository}
              organizationId={organizationId}
              integrationId={integrationId}
              repository={repository}
              event={failure.limitEvent!}
            />
          ))
        : null}
    </WebhookFailureNotice>
  );
}

export function IntegrationWebhookFailures({
  organizationId,
  integration,
}: {
  organizationId: string;
  integration: OrganizationsIntegration;
}) {
  const { canAct } = usePermissions();
  const failures = integration.status?.webhookFailures ?? [];
  const setups = integration.status?.webhookSetups ?? [];
  if (failures.length === 0 && setups.length === 0) return null;
  return (
    <div className="space-y-3">
      {failures.map((failure) => (
        <IntegrationWebhookFailure
          key={failure.id}
          organizationId={organizationId}
          integration={integration}
          failure={failure}
          canRetry={canAct("integrations", "update")}
        />
      ))}
      {setups.length > 0 ? (
        <details className="rounded-lg border p-4 text-sm">
          <summary className="cursor-pointer font-semibold">Webhook setup</summary>
          <ul className="mt-3 space-y-2">
            {setups.map((setup) => (
              <li key={setup.id} className="flex flex-wrap justify-between gap-x-3">
                <span>{[setup.repositories?.join(", "), setup.events?.join(", ")].filter(Boolean).join(" · ")}</span>
                <span>{webhookSetupLabel(setup.state)}</span>
              </li>
            ))}
          </ul>
        </details>
      ) : null}
    </div>
  );
}

function webhookSetupLabel(state: string | undefined): string {
  if (state === "ready") return "Registered";
  if (state === "failed") return "Setup failed";
  return webhookSetupInProgress(state) ? "Setup in progress" : "Status unavailable";
}
