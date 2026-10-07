import { ListTodo } from "lucide-react";

import type { IntakeSurfaceState } from "@/lib/intakeCatalog";

import { ConnectOptionRow, IntegrationChoiceIcon } from "./onboardingSteps";
import { vcsLabel, type IntegrationId, type VcsHostId } from "./onboardingFixtures";
import type { OnboardingSetupApi } from "./useOnboardingSetupState";

function noIntakeState(): IntakeSurfaceState | undefined {
  return undefined;
}

function intakeRowState(state: IntakeSurfaceState | undefined): {
  hidden: boolean;
  soon: boolean;
  disabled: boolean;
  meta?: string;
} {
  if (state === undefined) {
    // Intake catalog has not loaded yet. Keep the row visible but unavailable.
    return { hidden: false, soon: false, disabled: true };
  }
  return {
    hidden: state === "hidden",
    soon: state === "soon",
    disabled: state === "soon",
    meta: state === "beta" ? "Beta" : undefined,
  };
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
  const hostIssues = intakeRowState(host === "github" ? intakeState("github-issues") : "available");
  const linearIssues = intakeRowState(intakeState("linear-issues"));
  const jiraIssues = intakeRowState(intakeState("jira-issues"));

  return (
    <>
      {!hostIssues.hidden ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name={host} />}
          title={`Use ${vcsLabel(host)} Issues`}
          detail={`Find agent-ready work in open issues on ${backlogRepo}.`}
          meta={hostIssues.meta}
          selected={setup.issuesChoice === "vcs"}
          connectLabel={vcsLabel(host)}
          connected
          soon={hostIssues.soon}
          disabled={hostIssues.disabled}
          onSelect={() => setup.setIssuesChoice("vcs")}
        />
      ) : null}
      {!linearIssues.hidden ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="linear" />}
          title="Linear"
          detail="Find agent-ready work in your Linear backlog."
          meta={linearIssues.meta}
          selected={setup.issuesChoice === "linear"}
          connectLabel="Linear"
          connected={setup.connected.has("linear")}
          soon={linearIssues.soon}
          disabled={linearIssues.disabled}
          onSelect={() => setup.setIssuesChoice("linear")}
          onConnect={() => onRequestConnect("linear")}
        />
      ) : null}
      {!jiraIssues.hidden ? (
        <ConnectOptionRow
          icon={<IntegrationChoiceIcon name="jira" />}
          title="Jira"
          detail="Find agent-ready work in your Jira backlog."
          meta={jiraIssues.meta}
          selected={setup.issuesChoice === "jira"}
          connectLabel="Jira"
          connected={setup.connected.has("jira")}
          soon={jiraIssues.soon}
          disabled={jiraIssues.disabled}
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
