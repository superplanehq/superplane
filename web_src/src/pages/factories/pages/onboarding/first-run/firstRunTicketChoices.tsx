import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { LinearIntakeScopeFields } from "../../LinearIntakeScopeFields";
import type { LinearIntakeScope } from "../../linearIntakeScope";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunPanel } from "./FirstRunShell";
import type { FirstRunTicketSource } from "./firstRunTypes";

type TicketProject = { id?: string; name?: string };

export function FirstRunTicketChoices({
  githubAvailable,
  ticketSource,
  saving,
  githubConnected,
  jiraConnected,
  jiraProjects,
  jiraProjectsLoading,
  jiraProjectsError,
  jiraProjectId,
  linearAvailable,
  linearFeatureLoading,
  linearConnected,
  organizationId,
  linearIntegrationId,
  linearScope,
  onSelectTicketSource,
  onConnectGitHub,
  onConnectJira,
  onSelectJiraProject,
  onRetryJiraProjects,
  onConnectLinear,
  onChangeLinearScope,
}: {
  githubAvailable: boolean;
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  githubConnected: boolean;
  jiraConnected: boolean;
  jiraProjects: TicketProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  linearAvailable: boolean;
  linearFeatureLoading: boolean;
  linearConnected: boolean;
  organizationId: string;
  linearIntegrationId: string;
  linearScope: LinearIntakeScope;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onConnectGitHub?: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onChangeLinearScope?: (scope: LinearIntakeScope) => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;
  return (
    <FirstRunPanel>
      <div className="space-y-3">
        {githubAvailable && (
          <ConnectOptionRow
            icon={<IntegrationChoiceIcon name="github" />}
            title={copy.githubIssues}
            detail={copy.githubIssuesHelper}
            connectLabel="GitHub"
            connected={githubConnected}
            onConnect={onConnectGitHub}
            selected={ticketSource === "github-issues"}
            disabled={saving}
            onSelect={() => onSelectTicketSource("github-issues")}
          />
        )}
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="jira" />}
          title={copy.jira}
          detail={copy.jiraHelper}
          selected={ticketSource === "jira"}
          connectLabel={copy.jira}
          connected={jiraConnected}
          disabled={saving}
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
      <FirstRunLinearScopeFields
        visible={linearAvailable && ticketSource === "linear" && linearConnected}
        saving={saving}
        organizationId={organizationId}
        linearIntegrationId={linearIntegrationId}
        linearScope={linearScope}
        onChangeLinearScope={onChangeLinearScope}
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

function FirstRunLinearScopeFields({
  visible,
  saving,
  organizationId,
  linearIntegrationId,
  linearScope,
  onChangeLinearScope,
}: {
  visible: boolean;
  saving: boolean;
  organizationId: string;
  linearIntegrationId: string;
  linearScope: LinearIntakeScope;
  onChangeLinearScope?: (scope: LinearIntakeScope) => void;
}) {
  if (!visible) {
    return null;
  }

  return (
    <div className="mt-4 border-t border-border pt-4" data-testid="first-run-linear-scope">
      <fieldset disabled={saving} className="contents">
        <LinearIntakeScopeFields
          organizationId={organizationId}
          integrationId={linearIntegrationId}
          scope={linearScope}
          onChange={(scope) => onChangeLinearScope?.(scope)}
        />
      </fieldset>
    </div>
  );
}

function FirstRunJiraProjectFields({
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
  if (!visible) {
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
