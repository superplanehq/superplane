import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunPanel } from "./FirstRunShell";
import type { FirstRunTicketSource } from "./firstRunTypes";

const copy = FIRST_RUN_COPY.tickets;

function connectedMeta(connected: boolean): string | undefined {
  return connected ? copy.connectedMeta : undefined;
}

/**
 * The ticket sources as plain choices. Connection and project setup happen
 * on the stepper card after a choice, so no row carries its own button.
 */
export function FirstRunTicketSourceList({
  githubAvailable,
  ticketSource,
  saving,
  githubConnected,
  jiraConnected,
  linearAvailable,
  linearFeatureLoading,
  linearConnected,
  onSelectTicketSource,
}: {
  githubAvailable: boolean;
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  githubConnected: boolean;
  jiraConnected: boolean;
  linearAvailable: boolean;
  linearFeatureLoading: boolean;
  linearConnected: boolean;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
}) {
  return (
    <FirstRunPanel>
      <div className="space-y-3" data-testid="first-run-ticket-sources">
        {githubAvailable && (
          <ConnectOptionRow
            icon={<IntegrationChoiceIcon name="github" />}
            title={copy.githubIssues}
            detail={copy.githubIssuesHelper}
            meta={connectedMeta(githubConnected)}
            selected={ticketSource === "github-issues"}
            disabled={saving}
            onSelect={() => onSelectTicketSource("github-issues")}
          />
        )}
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="jira" />}
          title={copy.jira}
          detail={copy.jiraHelper}
          meta={connectedMeta(jiraConnected)}
          selected={ticketSource === "jira"}
          disabled={saving}
          onSelect={() => onSelectTicketSource("jira")}
        />
        <FirstRunFlaggedTicketRow
          icon="linear"
          title={copy.linear}
          helper={copy.linearHelper}
          soonHelper={copy.linearSoonHelper}
          lookupLoading={copy.linearLookupLoading}
          available={linearAvailable}
          featureLoading={linearFeatureLoading}
          selected={ticketSource === "linear"}
          connected={linearConnected}
          saving={saving}
          onSelect={() => onSelectTicketSource("linear")}
        />
      </div>
    </FirstRunPanel>
  );
}

function FirstRunFlaggedTicketRow({
  icon,
  title,
  helper,
  soonHelper,
  lookupLoading,
  available,
  featureLoading,
  selected,
  connected,
  saving,
  onSelect,
}: {
  icon: "jira" | "linear";
  title: string;
  helper: string;
  soonHelper: string;
  lookupLoading: string;
  available: boolean;
  featureLoading: boolean;
  selected: boolean;
  connected: boolean;
  saving: boolean;
  onSelect: () => void;
}) {
  if (featureLoading) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name={icon} />}
        title={title}
        detail={lookupLoading}
        disabled
        onSelect={() => undefined}
      />
    );
  }
  if (!available) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name={icon} />}
        title={title}
        detail={soonHelper}
        soon
        disabled={saving}
        onSelect={() => undefined}
      />
    );
  }
  return (
    <ConnectOptionRow
      icon={<IntegrationChoiceIcon name={icon} />}
      title={title}
      detail={helper}
      meta={connectedMeta(connected)}
      selected={selected}
      disabled={saving}
      onSelect={onSelect}
    />
  );
}
