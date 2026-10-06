import { ListTodo } from "lucide-react";

import type { IntakeSurfaceState } from "@/lib/intakeCatalog";

import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import { vcsLabel, type IntegrationId, type VcsHostId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

function noIntakeState(): IntakeSurfaceState | undefined {
  return undefined;
}

export function IssuesSourceOptions({
  setup,
  backlogRepo,
  host,
  intakeState = noIntakeState,
  onRequestConnect,
}: {
  setup: OnboardingSetupApi;
  backlogRepo: string;
  host: VcsHostId;
  /** State of each intake for the organization. Undefined until the intake catalog loads. */
  intakeState?: (key: string) => IntakeSurfaceState | undefined;
  onRequestConnect: (id: IntegrationId) => void;
}) {
  const hostIssuesState = host === "github" ? intakeState("github-issues") : undefined;
  const linearIssuesState = intakeState("linear-issues");
  const jiraIssuesState = intakeState("jira-issues");

  return (
    <>
      {hostIssuesState !== "hidden" ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name={host} />}
          title={`Use ${vcsLabel(host)} Issues`}
          detail={`Find agent-ready work in open issues on ${backlogRepo}.`}
          meta={hostIssuesState === "beta" ? "Beta" : undefined}
          selected={setup.issuesChoice === "vcs"}
          connectLabel={vcsLabel(host)}
          connected
          soon={hostIssuesState === "soon"}
          onSelect={() => setup.setIssuesChoice("vcs")}
        />
      ) : null}
      {linearIssuesState !== "hidden" ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="linear" />}
          title="Linear"
          detail="Find agent-ready work in your Linear backlog."
          meta={linearIssuesState === "beta" ? "Beta" : undefined}
          selected={setup.issuesChoice === "linear"}
          connectLabel="Linear"
          connected={setup.connected.has("linear")}
          soon={linearIssuesState === "soon"}
          onSelect={() => setup.setIssuesChoice("linear")}
          onConnect={() => onRequestConnect("linear")}
        />
      ) : null}
      {jiraIssuesState !== "hidden" ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="jira" />}
          title="Jira"
          detail="Find agent-ready work in your Jira backlog."
          meta={jiraIssuesState === "beta" ? "Beta" : undefined}
          selected={setup.issuesChoice === "jira"}
          connectLabel="Jira"
          connected={setup.connected.has("jira")}
          soon={jiraIssuesState === "soon"}
          onSelect={() => setup.setIssuesChoice("jira")}
          onConnect={() => onRequestConnect("jira")}
        />
      ) : null}
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
