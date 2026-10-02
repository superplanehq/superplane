import { LoadingButton } from "@/components/ui/loading-button";
import { isIntakeSelectable, type IntakeSurfaceEntry, type IntakeSurfaceState } from "@/lib/intakeCatalog";
import { findIntakePresentation } from "@/lib/intakePresentation";

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
  /** State of each intake for the organization. Undefined until the intake catalog loads. */
  intakeState?: (key: string) => IntakeSurfaceState | undefined;
  /** True while the intake catalog loads. Rows do not show Coming soon. */
  intakesLoading?: boolean;
  /** Ticket intakes in display order. Null until the intake catalog loads. */
  ticketIntakes?: IntakeSurfaceEntry[] | null;
  /**
   * A saved Jira choice cannot continue until the intake catalog confirms Jira.
   * The notice explains the block.
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
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
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
  intakeState = noIntakeState,
  intakesLoading = false,
  ticketIntakes,
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
  onSelectTicketSource,
  onAnalyzeTickets,
  onConnectJira,
  onSelectJiraProject,
  onRetryJiraProjects,
}: FirstRunTicketsScreenProps) {
  const copy = FIRST_RUN_COPY.tickets;
  const jiraAvailable = isIntakeSelectable(intakeState("jira-issues"));
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
            {ticketRows(ticketIntakes).map(({ key, entry }) => (
              <FirstRunTicketRow
                key={key}
                intakeKey={key}
                entry={entry}
                state={intakeState(key)}
                intakesLoading={intakesLoading}
                ticketSource={ticketSource}
                saving={saving}
                jiraConnected={jiraConnected}
                onSelectTicketSource={onSelectTicketSource}
                onConnectJira={onConnectJira}
              />
            ))}
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
            onSelectJiraProject={onSelectJiraProject}
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

const DEFAULT_TICKET_ROWS = ["github-issues", "jira-issues", "linear-issues"];

function noIntakeState(): IntakeSurfaceState | undefined {
  return undefined;
}

function ticketRows(ticketIntakes: IntakeSurfaceEntry[] | null | undefined) {
  if (!ticketIntakes) {
    return DEFAULT_TICKET_ROWS.map((key) => ({ key, entry: undefined }));
  }
  return ticketIntakes.map((entry) => ({ key: entry.key, entry }));
}

type FirstRunTicketRowProps = {
  intakeKey: string;
  entry?: IntakeSurfaceEntry;
  state: IntakeSurfaceState | undefined;
  intakesLoading: boolean;
  ticketSource: FirstRunTicketSource | null;
  saving: boolean;
  jiraConnected: boolean;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onConnectJira?: () => void;
};

/** Exported for the admin preview, which renders the row read-only. */
export function FirstRunTicketRow(props: FirstRunTicketRowProps) {
  const copy = FIRST_RUN_COPY.tickets;
  const { intakeKey, state, saving, ticketSource } = props;
  const meta = state === "beta" ? copy.beta : undefined;

  if (intakeKey === "github-issues") {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="github" />}
        title={copy.githubIssues}
        detail={copy.githubIssuesHelper}
        meta={meta}
        selected={ticketSource === "github-issues"}
        soon={state === "soon"}
        disabled={saving}
        onSelect={() => props.onSelectTicketSource("github-issues")}
      />
    );
  }
  if (intakeKey === "jira-issues") {
    return <FirstRunJiraTicketRow {...props} meta={meta} />;
  }
  return <FirstRunSoonTicketRow intakeKey={intakeKey} entry={props.entry} saving={saving} />;
}

function FirstRunSoonTicketRow({
  intakeKey,
  entry,
  saving,
}: {
  intakeKey: string;
  entry?: IntakeSurfaceEntry;
  saving: boolean;
}) {
  const copy = FIRST_RUN_COPY.tickets;
  if (intakeKey === "linear-issues") {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="linear" />}
        title={copy.linear}
        detail={copy.linearHelper}
        soon
        disabled={saving}
        onSelect={() => undefined}
      />
    );
  }
  const name = entry?.name ?? findIntakePresentation(intakeKey)?.name ?? intakeKey;
  const iconSrc = entry?.iconSrc ?? findIntakePresentation(intakeKey)?.iconSrc;
  return (
    <ConnectOptionRow
      icon={<TicketIntakeGlyph name={name} iconSrc={iconSrc} />}
      title={name}
      detail={copy.intakeSoonHelper}
      soon
      disabled={saving}
      onSelect={() => undefined}
    />
  );
}

function TicketIntakeGlyph({ name, iconSrc }: { name: string; iconSrc?: string }) {
  if (iconSrc) {
    return <img src={iconSrc} alt="" className="size-5" />;
  }
  return (
    <span className="flex size-5 items-center justify-center rounded-md bg-muted text-[11px] font-medium text-muted-foreground">
      {name.charAt(0).toUpperCase()}
    </span>
  );
}

function FirstRunJiraTicketRow({
  state,
  intakesLoading,
  ticketSource,
  saving,
  jiraConnected,
  meta,
  onSelectTicketSource,
  onConnectJira,
}: FirstRunTicketRowProps & { meta?: string }) {
  const copy = FIRST_RUN_COPY.tickets;
  if (state === undefined && intakesLoading) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="jira" />}
        title={copy.jira}
        detail={copy.jiraLookupLoading}
        disabled
        onSelect={() => undefined}
      />
    );
  }
  if (!isIntakeSelectable(state)) {
    return (
      <ConnectOptionRow
        icon={<IntegrationChoiceIcon name="jira" />}
        title={copy.jira}
        detail={copy.jiraSoonHelper}
        soon
        disabled={saving}
        onSelect={() => undefined}
      />
    );
  }
  return (
    <ConnectOptionRow
      icon={<IntegrationChoiceIcon name="jira" />}
      title={copy.jira}
      detail={copy.jiraHelper}
      meta={meta}
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
  onSelectJiraProject,
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
  onSelectJiraProject?: (id: string) => void;
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
      </fieldset>
    </div>
  );
}
