import { LoadingButton } from "@/components/ui/loading-button";

import { JiraCompletionColumnFields } from "../../JiraCompletionColumnFields";
import { DEFAULT_JIRA_COMPLETION_SETTINGS } from "../../intakeSourceSettingsModel";
import type { JiraCompletionColumnValue } from "../../jiraCompletionColumn";
import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { canAnalyzeTicketSource } from "./firstRunTicketSource";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export type FirstRunJiraProject = { id?: string; name?: string };

export type FirstRunJiraChoiceBlock = "loading" | "lookup-failed";

type FirstRunTicketsScreenProps = {
  ticketSource: FirstRunTicketSource | null;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  /** True when the organization has the Jira intake feature. Hides Jira when false. */
  jiraAvailable?: boolean;
  /**
   * A saved Jira choice cannot continue because the feature lookup has not
   * confirmed Jira. The row stays hidden. The notice explains the block.
   */
  jiraChoiceBlock?: FirstRunJiraChoiceBlock | null;
  continueLabel?: string;
  /** True while this screen waits to learn whether the agent screen is next. */
  continuePending?: boolean;
  /** True while this screen provisions the workspace, on the last screen. */
  saving?: boolean;
  savingLabel?: string;
  jiraConnected?: boolean;
  jiraProjects?: FirstRunJiraProject[];
  jiraProjectsLoading?: boolean;
  jiraProjectsError?: boolean;
  jiraProjectId?: string;
  jiraCompletion?: JiraCompletionColumnValue;
  organizationId?: string;
  jiraIntegrationId?: string;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onJiraCompletionChange?: (next: JiraCompletionColumnValue) => void;
  onRetryJiraProjects?: () => void;
};

function ticketContinueButton(args: {
  canAnalyze: boolean;
  continuePending: boolean;
  saving: boolean;
  savingLabel: string;
}) {
  const waitingForAgentChoice = args.continuePending && !args.saving;
  return {
    disabled: !args.canAnalyze || args.continuePending,
    loading: args.saving || args.continuePending,
    loadingText: waitingForAgentChoice ? FIRST_RUN_COPY.agent.loading : args.savingLabel,
  };
}

export function FirstRunTicketsScreen({
  ticketSource,
  chrome,
  sphere,
  jiraAvailable = false,
  jiraChoiceBlock = null,
  continueLabel = FIRST_RUN_COPY.tickets.analyze,
  continuePending = false,
  saving = false,
  savingLabel = FIRST_RUN_COPY.finish.saving,
  jiraConnected = false,
  jiraProjects = [],
  jiraProjectsLoading = false,
  jiraProjectsError = false,
  jiraProjectId = "",
  jiraCompletion = DEFAULT_JIRA_COMPLETION_SETTINGS,
  organizationId = "",
  jiraIntegrationId = "",
  onSelectTicketSource,
  onAnalyzeTickets,
  onConnectJira,
  onSelectJiraProject,
  onJiraCompletionChange,
  onRetryJiraProjects,
}: FirstRunTicketsScreenProps) {
  const copy = FIRST_RUN_COPY.tickets;
  const jiraSelectionBlocked = Boolean(jiraChoiceBlock) || (ticketSource === "jira" && !jiraAvailable);
  const canAnalyze = !jiraSelectionBlocked && canAnalyzeTicketSource({ ticketSource, jiraConnected, jiraProjectId });
  const continueButton = ticketContinueButton({ canAnalyze, continuePending, saving, savingLabel });
  const jiraChoiceNotice = jiraChoiceNoticeCopy(jiraChoiceBlock);

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome} busy={saving} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
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
            <FirstRunJiraTicketRow
              jiraAvailable={jiraAvailable}
              ticketSource={ticketSource}
              saving={saving}
              jiraConnected={jiraConnected}
              onSelectTicketSource={onSelectTicketSource}
              onConnectJira={onConnectJira}
            />
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="linear" />}
              title={copy.linear}
              detail={copy.linearHelper}
              soon
              disabled={saving}
              onSelect={() => undefined}
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
            jiraCompletion={jiraCompletion}
            organizationId={organizationId}
            jiraIntegrationId={jiraIntegrationId}
            onSelectJiraProject={onSelectJiraProject}
            onJiraCompletionChange={onJiraCompletionChange}
            onRetryJiraProjects={onRetryJiraProjects}
          />
        </FirstRunPanel>

        {jiraChoiceNotice ? (
          <p
            className={
              jiraChoiceBlock === "lookup-failed" ? "text-[13px] text-destructive" : "text-[13px] text-muted-foreground"
            }
            role={jiraChoiceBlock === "lookup-failed" ? "alert" : "status"}
            data-testid="first-run-jira-choice-notice"
          >
            {jiraChoiceNotice}
          </p>
        ) : null}

        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={continueButton.disabled}
            loading={continueButton.loading}
            loadingText={continueButton.loadingText}
            onClick={onAnalyzeTickets}
            data-testid="first-run-analyze-tickets"
          >
            {continueLabel}
          </LoadingButton>
        </div>
      </div>
    </FirstRunShell>
  );
}

function jiraChoiceNoticeCopy(block: FirstRunJiraChoiceBlock | null): string | null {
  if (block === "lookup-failed") return FIRST_RUN_COPY.tickets.jiraLookupFailed;
  if (block === "loading") return FIRST_RUN_COPY.tickets.jiraLookupLoading;
  return null;
}

function FirstRunJiraTicketRow({
  jiraAvailable,
  ticketSource,
  saving,
  jiraConnected,
  onSelectTicketSource,
  onConnectJira,
}: {
  jiraAvailable: boolean;
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  jiraConnected: boolean;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onConnectJira?: () => void;
}) {
  if (!jiraAvailable) {
    return null;
  }
  const copy = FIRST_RUN_COPY.tickets;
  return (
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
  jiraCompletion,
  organizationId,
  jiraIntegrationId,
  onSelectJiraProject,
  onJiraCompletionChange,
  onRetryJiraProjects,
}: {
  jiraAvailable: boolean;
  visible: boolean;
  copy: (typeof FIRST_RUN_COPY)["tickets"];
  saving: boolean;
  jiraProjects: FirstRunJiraProject[];
  jiraProjectsLoading: boolean;
  jiraProjectsError: boolean;
  jiraProjectId: string;
  jiraCompletion: JiraCompletionColumnValue;
  organizationId: string;
  jiraIntegrationId: string;
  onSelectJiraProject?: (id: string) => void;
  onJiraCompletionChange?: (next: JiraCompletionColumnValue) => void;
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
        {jiraProjectId && organizationId && jiraIntegrationId ? (
          <div className="mt-4">
            <JiraCompletionColumnFields
              organizationId={organizationId}
              integrationId={jiraIntegrationId}
              projectId={jiraProjectId}
              value={jiraCompletion}
              onChange={(next) => onJiraCompletionChange?.(next)}
            />
          </div>
        ) : null}
      </fieldset>
    </div>
  );
}
