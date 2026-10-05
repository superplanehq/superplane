import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { LinearProjectPicker } from "../../LinearProjectPicker";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunPanel } from "./FirstRunShell";
import type { FirstRunTicketSource } from "./firstRunTypes";

type TicketProject = { id?: string; name?: string };

export function FirstRunTicketChoices({
  ticketSource,
  saving,
  jiraAvailable,
  jiraFeatureLoading,
  jiraConnected,
  jiraProjects,
  jiraProjectsLoading,
  jiraProjectsError,
  jiraProjectId,
  linearAvailable,
  linearFeatureLoading,
  linearConnected,
  linearProjects,
  linearProjectsLoading,
  linearProjectsError,
  linearProjectIds,
  onSelectTicketSource,
  onConnectJira,
  onSelectJiraProject,
  onRetryJiraProjects,
  onConnectLinear,
  onToggleLinearProject,
  onRetryLinearProjects,
}: {
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  jiraAvailable: boolean;
  jiraFeatureLoading: boolean;
  jiraConnected: boolean;
  jiraProjects: TicketProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  linearAvailable: boolean;
  linearFeatureLoading: boolean;
  linearConnected: boolean;
  linearProjects: TicketProject[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
  linearProjectIds: string[];
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;
  return (
    <FirstRunPanel>
      <div className="space-y-3">
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="github" />}
          title={copy.githubIssues}
          detail={copy.githubIssuesHelper}
          selected={ticketSource === "github-issues"}
          disabled={saving}
          onSelect={() => onSelectTicketSource("github-issues")}
        />
        <FirstRunFlaggedTicketRow
          icon="jira"
          title={copy.jira}
          helper={copy.jiraHelper}
          soonHelper={copy.jiraSoonHelper}
          lookupLoading={copy.jiraLookupLoading}
          available={jiraAvailable}
          featureLoading={jiraFeatureLoading}
          selected={ticketSource === "jira"}
          connected={jiraConnected}
          saving={saving}
          onSelect={() => onSelectTicketSource("jira")}
          onConnect={onConnectJira}
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
          onConnect={onConnectLinear}
        />
      </div>
      <FirstRunJiraProjectFields
        jiraAvailable={jiraAvailable}
        visible={ticketSource === "jira" && jiraConnected}
        copy={copy}
        saving={saving}
        jiraProjects={jiraProjects}
        jiraProjectsLoading={jiraProjectsLoading}
        jiraProjectsError={jiraProjectsError}
        jiraProjectId={jiraProjectId}
        onSelectJiraProject={onSelectJiraProject}
        onRetryJiraProjects={onRetryJiraProjects}
      />
      <FirstRunLinearProjectFields
        linearAvailable={linearAvailable}
        visible={ticketSource === "linear" && linearConnected}
        copy={copy}
        saving={saving}
        linearProjects={linearProjects}
        linearProjectsLoading={linearProjectsLoading}
        linearProjectsError={linearProjectsError}
        linearProjectIds={linearProjectIds}
        onToggleLinearProject={onToggleLinearProject}
        onRetryLinearProjects={onRetryLinearProjects}
      />
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
  onConnect,
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
  onConnect?: () => void;
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
      selected={selected}
      connectLabel={title}
      connected={connected}
      disabled={saving}
      onSelect={onSelect}
      onConnect={onConnect}
    />
  );
}

function FirstRunLinearProjectFields({
  linearAvailable,
  visible,
  copy,
  saving,
  linearProjects,
  linearProjectsLoading,
  linearProjectsError,
  linearProjectIds,
  onToggleLinearProject,
  onRetryLinearProjects,
}: {
  linearAvailable: boolean;
  visible: boolean;
  copy: (typeof FIRST_RUN_COPY)["tickets"];
  saving: boolean;
  linearProjects: TicketProject[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
  linearProjectIds: string[];
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
}) {
  if (!linearAvailable || !visible) {
    return null;
  }

  return (
    <div className="mt-4 border-t border-border pt-4" data-testid="first-run-linear-projects">
      <p className="mb-3 text-[13px] font-medium">{copy.linearProjectHeading}</p>
      <fieldset disabled={saving} className="contents">
        <LinearProjectPicker
          projects={linearProjects}
          selectedIds={linearProjectIds}
          loading={linearProjectsLoading}
          error={linearProjectsError}
          onToggle={(id) => onToggleLinearProject?.(id)}
          onRetry={() => onRetryLinearProjects?.()}
        />
      </fieldset>
    </div>
  );
}

function FirstRunJiraProjectFields({
  jiraAvailable,
  visible,
  copy,
  saving,
  jiraProjects,
  jiraProjectsLoading,
  jiraProjectsError,
  jiraProjectId,
  onSelectJiraProject,
  onRetryJiraProjects,
}: {
  jiraAvailable: boolean;
  visible: boolean;
  copy: (typeof FIRST_RUN_COPY)["tickets"];
  saving: boolean;
  jiraProjects: TicketProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
}) {
  if (!jiraAvailable || !visible) {
    return null;
  }

  return (
    <div className="mt-4 border-t border-border pt-4" data-testid="first-run-jira-projects">
      <p className="mb-3 text-[13px] font-medium">{copy.jiraProjectHeading}</p>
      <fieldset disabled={saving} className="contents">
        <JiraProjectStep
          projects={jiraProjects}
          selectedId={jiraProjectId}
          loading={jiraProjectsLoading}
          error={jiraProjectsError}
          onSelect={(id) => onSelectJiraProject?.(id)}
          onRetry={() => onRetryJiraProjects?.()}
        />
      </fieldset>
    </div>
  );
}
