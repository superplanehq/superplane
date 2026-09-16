import { LoadingButton } from "@/components/ui/loading-button";

import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export function FirstRunTicketsScreen({
  ticketSource,
  chrome,
  sphere,
  continueLabel = FIRST_RUN_COPY.tickets.analyze,
  saving = false,
  savingLabel = FIRST_RUN_COPY.finish.saving,
  onSelectTicketSource,
  onAnalyzeTickets,
}: {
  ticketSource: FirstRunTicketSource | null;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  continueLabel?: string;
  /** True while this screen provisions the workspace, on the last screen. */
  saving?: boolean;
  savingLabel?: string;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;

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
              soon
              disabled={saving}
              onSelect={() => undefined}
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
        </FirstRunPanel>

        <div className="space-y-3">
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!ticketSource}
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
