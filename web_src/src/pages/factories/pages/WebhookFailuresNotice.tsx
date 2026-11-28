import type { OrganizationsIntegration } from "@/api-client";
import { Link } from "react-router";

import { Button } from "@/components/ui/button";
import { usePermissions } from "@/contexts/usePermissions";
import { useConnectedIntegrations, useRetryIntegrationWebhooks } from "@/hooks/useIntegrations";
import { githubHostFromIntegration, githubRepoWebUrl } from "@/lib/githubHost";
import { integrationDetailPath } from "@/lib/integrationSettingsPaths";
import { webhookSetupInProgress } from "@/lib/webhookSetup";

import {
  groupWebhookFailuresByRepository,
  webhookEventLabel,
  type RepositoryWebhookFailures,
} from "../lib/webhookFailureGroups";

/** Keeps webhook failures and their recovery action together above the board. */
export function WebhookFailuresNotice({
  organizationId,
  factoryId,
  integrationsBasePath,
}: {
  organizationId: string;
  factoryId: string;
  integrationsBasePath: string;
}) {
  const { canAct } = usePermissions();
  const canRead = canAct("integrations", "read");
  const { data: integrations = [] } = useConnectedIntegrations(organizationId, { enabled: canRead, factoryId });
  if (!canRead) return null;

  const affectedIntegrations = integrations.filter(
    (integration) => integration.metadata?.integrationName === "github" && integration.status?.webhookFailures?.length,
  );
  if (affectedIntegrations.length === 0) return null;

  return (
    <section className="space-y-3 px-3 pb-3" data-testid="webhook-failures-notice">
      {affectedIntegrations.map((integration) => (
        <ConnectionWebhookFailures
          key={integration.metadata?.id}
          organizationId={organizationId}
          integrationsBasePath={integrationsBasePath}
          integration={integration}
        />
      ))}
    </section>
  );
}

function ConnectionWebhookFailures({
  organizationId,
  integrationsBasePath,
  integration,
}: {
  organizationId: string;
  integrationsBasePath: string;
  integration: OrganizationsIntegration;
}) {
  const { canAct } = usePermissions();
  const retry = useRetryIntegrationWebhooks(organizationId);
  const integrationId = integration.metadata?.id ?? "";
  const connectionLabel = `${integration.metadata?.name || "Unnamed connection"} (GitHub)`;
  const failures = (integration.status?.webhookFailures ?? []).map((failure) => ({
    ...failure,
    integrationId,
    host: githubHostFromIntegration(integration),
  }));

  const retryable = failures.filter(
    (failure) => failure.id && failure.integrationId && !webhookSetupInProgress(failure.state),
  );
  const setupInProgress = retry.isPending || failures.every((failure) => webhookSetupInProgress(failure.state));
  const hasLimitError = failures.some((failure) => failure.limitEvent);
  const groups = groupWebhookFailuresByRepository(failures);
  const repositories = new Map(
    groups.flatMap(({ repository, host }) => {
      const url = githubRepoWebUrl(repository, host);
      return url ? [[`${url}/settings/hooks`, repository] as const] : [];
    }),
  );

  return (
    <div
      role="alert"
      aria-label={connectionLabel}
      className="rounded-lg border border-amber-300 bg-background p-4 dark:border-amber-700"
    >
      <h2 className="text-[15px] font-semibold tracking-[-0.02em] text-foreground">
        {setupInProgress ? "Retrying webhook setup" : "Webhook setup failed"}
      </h2>
      <h3 className="mt-2 break-words text-sm font-medium text-foreground">{connectionLabel}</h3>
      <p className="mt-1 text-sm text-muted-foreground">
        {hasLimitError
          ? "GitHub allows 20 webhooks per event in each repository. Remove unused webhooks for the events at the limit, then try again."
          : "Some repository updates cannot arrive. Review the errors below, then try again."}
      </p>
      <div className="mt-4 space-y-4">
        {groups.map((group) => (
          <RepositoryFailureSummary key={group.key} group={group} />
        ))}
      </div>
      <div className="mt-4 flex flex-wrap items-center gap-2">
        {integrationId ? (
          <Button asChild size="sm" variant="outline">
            <Link to={integrationDetailPath(integrationsBasePath, integrationId)}>Open integration settings</Link>
          </Button>
        ) : null}
        {[...repositories].map(([url, repository]) => (
          <Button key={url} asChild size="sm" variant="outline">
            <a href={url} target="_blank" rel="noreferrer" aria-label={`Open ${repository} webhooks`}>
              {repositories.size === 1 ? "Open webhooks" : `Open ${repository} webhooks`}
            </a>
          </Button>
        ))}
        {canAct("integrations", "update") ? (
          <Button
            type="button"
            size="sm"
            disabled={retry.isPending || retryable.length === 0}
            onClick={() =>
              retry.mutate(
                retryable.map((failure) => ({ integrationId: failure.integrationId, webhookId: failure.id! })),
              )
            }
          >
            {setupInProgress ? "Trying again" : "Try again"}
          </Button>
        ) : null}
      </div>
      {retry.isError ? (
        <p role="status" className="mt-2 text-[13px] text-destructive">
          SuperPlane could not retry all webhooks. Try again.
        </p>
      ) : null}
    </div>
  );
}

function RepositoryFailureSummary({ group }: { group: RepositoryWebhookFailures }) {
  const limitedEvents = [
    ...new Set(group.failures.flatMap((failure) => (failure.limitEvent ? [failure.limitEvent] : []))),
  ];
  const otherEvents = [...new Set(group.failures.flatMap((failure) => failure.events ?? []))].filter(
    (event) => !limitedEvents.includes(event),
  );
  const eventRows = [
    { label: "Events at the limit", events: limitedEvents },
    { label: limitedEvents.length ? "Other affected events" : "Affected events", events: otherEvents },
  ].filter((row) => row.events.length > 0);
  const messages = [
    ...new Set(group.failures.flatMap((failure) => (!failure.limitEvent && failure.message ? [failure.message] : []))),
  ];
  const inProgress = group.failures.filter((failure) => webhookSetupInProgress(failure.state)).length;

  return (
    <div className="space-y-2">
      <h4 className="break-words text-sm font-medium text-foreground">
        {group.repository || "Repository unavailable"}
      </h4>
      {eventRows.length > 0 ? (
        <dl className="space-y-1 text-sm leading-relaxed">
          {eventRows.map(({ label, events }) => (
            <div key={label} className="flex flex-wrap gap-x-2">
              <dt className="text-muted-foreground">{label}</dt>
              <dd className="text-foreground">{events.map(webhookEventLabel).join(", ")}</dd>
            </div>
          ))}
        </dl>
      ) : null}
      {messages.map((message) => (
        <p key={message} className="text-sm text-muted-foreground">
          {message}
        </p>
      ))}
      {inProgress > 0 ? (
        <p className="text-sm text-muted-foreground">
          Setup in progress for {inProgress} of {group.failures.length} webhooks.
        </p>
      ) : null}
    </div>
  );
}
