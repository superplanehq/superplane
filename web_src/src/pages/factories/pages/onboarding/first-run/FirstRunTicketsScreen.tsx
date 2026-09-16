import { LoadingButton } from "@/components/ui/loading-button";

import { JiraProjectStep } from "../../JiraIntakeSetupSteps";
import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { canAnalyzeTicketSource } from "./firstRunTicketSource";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export type FirstRunJiraProject = { id?: string; name?: string };

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
  onSelectTicketSource,
  onAnalyzeTickets,
  onConnectJira,
  onSelectJiraProject,
  onRetryJiraProjects,
}: {
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
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;
  const jiraSelected = ticketSource === "jira";
  const canAnalyze = canAnalyzeTicketSource({ ticketSource, jiraConnected, jiraProjectId });

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
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="jira" />}
              title={copy.jira}
              detail={copy.jiraHelper}
              selected={jiraSelected}
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
          {jiraSelected && jiraConnected ? (
            <div className="mt-4 border-t border-border pt-4" data-testid="first-run-jira-projects">
              <p className="mb-3 text-[13px] font-medium">{copy.jiraProjectHeading}</p>
              <JiraProjectStep
                projects={jiraProjects}
                selectedId={jiraProjectId}
                loading={jiraProjectsLoading}
                error={jiraProjectsError}
                onSelect={(id) => onSelectJiraProject?.(id)}
                onRetry={() => onRetryJiraProjects?.()}
              />
            </div>
          ) : null}
        </FirstRunPanel>

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
