import { ConnectOptionRow, IntegrationChoiceIcon } from "../onboardingSteps";
import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunPanel, FirstRunShell } from "./FirstRunShell";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export function FirstRunTicketsScreen({
  repository,
  ticketSource,
  chrome,
  onSelectTicketSource,
}: {
  repository: string;
  ticketSource: FirstRunTicketSource | null;
  chrome?: FirstRunChrome;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
}) {
  const copy = FIRST_RUN_COPY.tickets;

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] font-medium tracking-[-0.01em]">{copy.repositoryCaption(repository)}</p>
        <p className="text-[15px] leading-6 text-muted-foreground">{copy.body}</p>
      </FirstRunHeading>

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
        <p className="text-[13px] text-muted-foreground">{copy.trust}</p>
        <p className="text-[12px] text-muted-foreground">{copy.startHint}</p>
      </div>
    </FirstRunShell>
  );
}
