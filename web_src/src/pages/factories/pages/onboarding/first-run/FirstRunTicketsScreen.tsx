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

type FirstRunTicketsScreenProps = {
  ticketSource: FirstRunTicketSource | null;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  continueLabel?: string;
  /** True while this screen provisions the workspace, on the last screen. */
  saving?: boolean;
  savingLabel?: string;
  jiraConnected?: boolean;
  jiraProjects?: FirstRunJiraProject[];
  jiraProjectsLoading?: boolean;
  jiraProjectsError?: boolean;
  jiraProjectId?: string;
  jiraCompletion?: JiraCompletionColumnValue;
  /** True when this screen is last, so the completion column has no later step. */
  showJiraCompletion?: boolean;
  organizationId?: string;
  jiraIntegrationId?: string;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onJiraCompletionChange?: (next: JiraCompletionColumnValue) => void;
  onRetryJiraProjects?: () => void;
};

export function FirstRunTicketsScreen({
  ticketSource,
  chrome,
  sphere,
  continueLabel = FIRST_RUN_COPY.tickets.analyze,
  saving = false,
  savingLabel = FIRST_RUN_COPY.finish.saving,
  jiraConnected = false,
  jiraProjects = [],
  jiraProjectsLoading = false,
  jiraProjectsError = false,
  jiraProjectId = "",
  jiraCompletion = DEFAULT_JIRA_COMPLETION_SETTINGS,
  showJiraCompletion = false,
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
  const canAnalyze = canAnalyzeTicketSource({ ticketSource, jiraConnected, jiraProjectId });
  const jiraProjectVisible = ticketSource === "jira" && jiraConnected;
  const completionVisible =
    showJiraCompletion && jiraProjectVisible && Boolean(jiraProjectId && organizationId && jiraIntegrationId);

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome} busy={saving} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 flex flex-col gap-6">
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
            visible={jiraProjectVisible}
            copy={copy}
            saving={saving}
            jiraProjects={jiraProjects}
            jiraProjectsLoading={jiraProjectsLoading}
            jiraProjectsError={jiraProjectsError}
            jiraProjectId={jiraProjectId}
            onSelectJiraProject={onSelectJiraProject}
            onRetryJiraProjects={onRetryJiraProjects}
          />
        </FirstRunPanel>

        {completionVisible ? (
          <fieldset disabled={saving} className="m-0 min-w-0 border-0 p-0">
            <JiraCompletionColumnFields
              organizationId={organizationId}
              integrationId={jiraIntegrationId}
              projectId={jiraProjectId}
              value={jiraCompletion}
              onChange={(next) => onJiraCompletionChange?.(next)}
              layout="plain"
            />
          </fieldset>
        ) : null}

        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!canAnalyze}
            loading={saving}
            loadingText={savingLabel}
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
  jiraProjects: FirstRunJiraProject[];
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
