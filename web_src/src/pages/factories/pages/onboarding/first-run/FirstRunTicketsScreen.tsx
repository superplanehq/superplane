import { LoadingButton } from "@/components/ui/loading-button";

import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export function FirstRunTicketsScreen({
  ticketSource,
  chrome,
  continueLabel = FIRST_RUN_COPY.tickets.analyze,
  saving = false,
  onSelectTicketSource,
  onAnalyzeTickets,
}: {
  ticketSource: FirstRunTicketSource | null;
  chrome?: FirstRunChrome;
  continueLabel?: string;
  /** True while this screen provisions the workspace, on the last screen. */
  saving?: boolean;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome}>
      <FirstRunHeading headline={copy.headline} />

      <div className="mt-8 space-y-4">
        <FirstRunPanel>
          <div className="space-y-3">
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="github" />}
              title={copy.githubIssues}
              detail={copy.githubIssuesHelper}
              selected={ticketSource === "github-issues"}
              onSelect={() => onSelectTicketSource("github-issues")}
            />
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="jira" />}
              title={copy.jira}
              detail={copy.jiraHelper}
              soon
              onSelect={() => undefined}
            />
            <ConnectOptionRow
              icon={<IntegrationChoiceIcon name="linear" />}
              title={copy.linear}
              detail={copy.linearHelper}
              soon
              onSelect={() => undefined}
            />
          </div>
        </FirstRunPanel>

        <div className="space-y-3">
          <div className="space-y-1 text-[13px] leading-5 text-muted-foreground">
            <p>{copy.trust}</p>
            <p>{copy.scoreHint}</p>
          </div>
          <LoadingButton
            type="button"
            className="w-full"
            disabled={!ticketSource}
            loading={saving}
            loadingText={FIRST_RUN_COPY.finish.saving}
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
