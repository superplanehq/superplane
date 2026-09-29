import { ListTodo } from "lucide-react";

import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import { vcsLabel, type IntegrationId, type VcsHostId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

export function IssuesSourceOptions({
  setup,
  backlogRepo,
  host,
  onRequestConnect,
}: {
  setup: OnboardingSetupApi;
  backlogRepo: string;
  host: VcsHostId;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  return (
    <>
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name={host} />}
        title={`Use ${vcsLabel(host)} Issues`}
        detail={`Find agent-ready work in open issues on ${backlogRepo}.`}
        selected={setup.issuesChoice === "vcs"}
        connectLabel={vcsLabel(host)}
        connected
        onSelect={() => setup.setIssuesChoice("vcs")}
      />
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="linear" />}
        title="Linear"
        detail="Find agent-ready work in your Linear backlog."
        selected={setup.issuesChoice === "linear"}
        connectLabel="Linear"
        connected={setup.connected.has("linear")}
        soon
        onSelect={() => setup.setIssuesChoice("linear")}
        onConnect={() => onRequestConnect("linear")}
      />
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="jira" />}
        title="Jira"
        detail="Find agent-ready work in your Jira backlog."
        selected={setup.issuesChoice === "jira"}
        connectLabel="Jira"
        connected={setup.connected.has("jira")}
        soon
        onSelect={() => setup.setIssuesChoice("jira")}
        onConnect={() => onRequestConnect("jira")}
      />
      <ConnectOptionRow
        icon={<ListTodo className="size-5 text-muted-foreground" aria-hidden />}
        title="Skip for now"
        detail="Do not import a backlog. Create tasks yourself instead."
        selected={setup.issuesChoice === "skip"}
        onSelect={() => setup.setIssuesChoice("skip")}
      />
    </>
  );
}
