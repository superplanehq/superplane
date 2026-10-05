import { LoadingButton } from "@/components/ui/loading-button";
import { isIntakeSelectable, type IntakeSurfaceEntry, type IntakeSurfaceState } from "@/lib/intakeCatalog";
import { findIntakePresentation } from "@/lib/intakePresentation";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketChoices } from "./firstRunTicketChoices";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { canAnalyzeTicketSource } from "./firstRunTicketSource";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export type FirstRunJiraProject = { id?: string; name?: string };

export type FirstRunFlaggedChoiceBlock = "loading" | "lookup-failed";

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
  jiraChoiceBlock?: FirstRunFlaggedChoiceBlock | null;
  linearAvailable?: boolean;
  linearFeatureLoading?: boolean;
  linearChoiceBlock?: FirstRunFlaggedChoiceBlock | null;
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
  linearConnected?: boolean;
  linearProjects?: FirstRunJiraProject[];
  linearProjectsLoading?: boolean;
  linearProjectsError?: boolean;
  linearProjectIds?: string[];
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
};

const TICKET_SCREEN_DEFAULTS = {
  jiraAvailable: false,
  jiraFeatureLoading: false,
  jiraChoiceBlock: null as FirstRunFlaggedChoiceBlock | null,
  linearAvailable: false,
  linearFeatureLoading: false,
  linearChoiceBlock: null as FirstRunFlaggedChoiceBlock | null,
  continueLabel: FIRST_RUN_COPY.tickets.analyze,
  continuePending: false,
  saving: false,
  savingLabel: FIRST_RUN_COPY.finish.saving,
  jiraConnected: false,
  jiraProjects: [] as FirstRunJiraProject[],
  jiraProjectsLoading: false,
  jiraProjectsError: false,
  jiraProjectId: "",
  linearConnected: false,
  linearProjects: [] as FirstRunJiraProject[],
  linearProjectsLoading: false,
  linearProjectsError: false,
  linearProjectIds: [] as string[],
};

function ticketScanAllowed(args: {
  ticketSource: FirstRunTicketSource | null;
  jiraAvailable: boolean;
  jiraChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  jiraConnected: boolean;
  jiraProjectId: string;
  linearAvailable: boolean;
  linearChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  linearConnected: boolean;
  linearProjectIds: string[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
}): boolean {
  const jiraSelectionBlocked = Boolean(args.jiraChoiceBlock) || (args.ticketSource === "jira" && !args.jiraAvailable);
  const linearSelectionBlocked =
    Boolean(args.linearChoiceBlock) || (args.ticketSource === "linear" && !args.linearAvailable);
  if (args.ticketSource === "linear" && (args.linearProjectsLoading || args.linearProjectsError)) return false;
  return (
    !jiraSelectionBlocked &&
    !linearSelectionBlocked &&
    canAnalyzeTicketSource({
      ticketSource: args.ticketSource,
      jiraConnected: args.jiraConnected,
      jiraProjectId: args.jiraProjectId,
      linearConnected: args.linearConnected,
      linearProjectIds: args.linearProjectIds,
    })
  );
}

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

<<<<<<< HEAD
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
=======
export function FirstRunTicketsScreen(props: FirstRunTicketsScreenProps) {
  const {
    ticketSource,
    chrome,
    sphere,
    jiraAvailable,
    jiraFeatureLoading,
    jiraChoiceBlock,
    linearAvailable,
    linearFeatureLoading,
    linearChoiceBlock,
    continueLabel,
    continuePending,
    saving,
    savingLabel,
    jiraConnected,
    jiraProjects,
    jiraProjectsLoading,
    jiraProjectsError,
    jiraProjectId,
    linearConnected,
    linearProjects,
    linearProjectsLoading,
    linearProjectsError,
    linearProjectIds,
    onSelectTicketSource,
    onAnalyzeTickets,
    onConnectJira,
    onSelectJiraProject,
    onRetryJiraProjects,
    onConnectLinear,
    onToggleLinearProject,
    onRetryLinearProjects,
  } = { ...TICKET_SCREEN_DEFAULTS, ...props };
  const copy = FIRST_RUN_COPY.tickets;
  const canAnalyze = ticketScanAllowed({
    ticketSource,
    jiraAvailable,
    jiraChoiceBlock,
    jiraConnected,
    jiraProjectId,
    linearAvailable,
    linearChoiceBlock,
    linearConnected,
    linearProjectIds,
    linearProjectsLoading,
    linearProjectsError,
  });
>>>>>>> origin/main
  const continueButton = ticketContinueButton({ canAnalyze, continuePending, saving, savingLabel });

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome} busy={saving} sphere={sphere}>
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
<<<<<<< HEAD
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

=======
        <FirstRunTicketChoices
          ticketSource={ticketSource}
          saving={saving}
          jiraAvailable={jiraAvailable}
          jiraFeatureLoading={jiraFeatureLoading}
          jiraConnected={jiraConnected}
          jiraProjects={jiraProjects}
          jiraProjectsLoading={jiraProjectsLoading}
          jiraProjectsError={jiraProjectsError}
          jiraProjectId={jiraProjectId}
          linearAvailable={linearAvailable}
          linearFeatureLoading={linearFeatureLoading}
          linearConnected={linearConnected}
          linearProjects={linearProjects}
          linearProjectsLoading={linearProjectsLoading}
          linearProjectsError={linearProjectsError}
          linearProjectIds={linearProjectIds}
          onSelectTicketSource={onSelectTicketSource}
          onConnectJira={onConnectJira}
          onSelectJiraProject={onSelectJiraProject}
          onRetryJiraProjects={onRetryJiraProjects}
          onConnectLinear={onConnectLinear}
          onToggleLinearProject={onToggleLinearProject}
          onRetryLinearProjects={onRetryLinearProjects}
        />
        <FirstRunChoiceNotice
          block={jiraChoiceBlock}
          loading={copy.jiraLookupLoading}
          failed={copy.jiraLookupFailed}
          testId="first-run-jira-choice-notice"
        />
        <FirstRunChoiceNotice
          block={linearChoiceBlock}
          loading={copy.linearLookupLoading}
          failed={copy.linearLookupFailed}
          testId="first-run-linear-choice-notice"
        />
>>>>>>> origin/main
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

function choiceNoticeCopy(block: FirstRunFlaggedChoiceBlock | null, loading: string, failed: string): string | null {
  if (block === "lookup-failed") return failed;
  if (block === "loading") return loading;
  return null;
}

<<<<<<< HEAD
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
=======
function FirstRunChoiceNotice({
  block,
  loading,
  failed,
  testId,
}: {
  block: FirstRunFlaggedChoiceBlock | null;
  loading: string;
  failed: string;
  testId: string;
}) {
  const notice = choiceNoticeCopy(block, loading, failed);
  if (!notice) return null;
  const lookupFailed = block === "lookup-failed";
  return (
    <p
      className={lookupFailed ? "text-[13px] text-destructive" : "text-[13px] text-muted-foreground"}
      role={lookupFailed ? "alert" : "status"}
      data-testid={testId}
    >
      {notice}
    </p>
>>>>>>> origin/main
  );
}
