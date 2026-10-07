import { isIntakeSelectable, type IntakeSurfaceEntry, type IntakeSurfaceState } from "@/lib/intakeCatalog";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import { FirstRunTicketBody } from "./firstRunTicketRows";
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
  vcsAvailable: boolean;
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
  const vcsSelectionBlocked = args.ticketSource === "github-issues" && !args.vcsAvailable;
  const jiraSelectionBlocked = Boolean(args.jiraChoiceBlock) || (args.ticketSource === "jira" && !args.jiraAvailable);
  const linearSelectionBlocked =
    Boolean(args.linearChoiceBlock) || (args.ticketSource === "linear" && !args.linearAvailable);
  if (args.ticketSource === "linear" && (args.linearProjectsLoading || args.linearProjectsError)) return false;
  return (
    !vcsSelectionBlocked &&
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

function noIntakeState(): IntakeSurfaceState | undefined {
  return undefined;
}

export function FirstRunTicketsScreen(props: FirstRunTicketsScreenProps) {
  const screen = { ...TICKET_SCREEN_DEFAULTS, ...props };
  const intakeState = props.intakeState ?? noIntakeState;
  const catalogLoaded = screen.ticketIntakes != null;
  const vcsAvailable = isIntakeSelectable(intakeState("github-issues"));
  const jiraAvailable = isIntakeSelectable(intakeState("jira-issues"));
  const linearSelectable = catalogLoaded ? isIntakeSelectable(intakeState("linear-issues")) : screen.linearAvailable;
  const canAnalyze = ticketScanAllowed({
    ticketSource: screen.ticketSource,
    vcsAvailable,
    jiraAvailable,
    jiraChoiceBlock: screen.jiraChoiceBlock,
    jiraConnected: screen.jiraConnected,
    jiraProjectId: screen.jiraProjectId,
    linearAvailable: linearSelectable,
    linearChoiceBlock: screen.linearChoiceBlock,
    linearConnected: screen.linearConnected,
    linearProjectIds: screen.linearProjectIds,
    linearProjectsLoading: screen.linearProjectsLoading,
    linearProjectsError: screen.linearProjectsError,
  });
  const continueButton = ticketContinueButton({
    canAnalyze,
    continuePending: screen.continuePending,
    saving: screen.saving,
    savingLabel: screen.savingLabel,
  });

  return (
    <FirstRunShell testId="first-run-tickets" chrome={screen.chrome} busy={screen.saving} sphere={screen.sphere}>
      <FirstRunHeading headline={FIRST_RUN_COPY.tickets.headline}>
        <p className="text-[13px] text-muted-foreground">{FIRST_RUN_COPY.tickets.intro}</p>
      </FirstRunHeading>
      <FirstRunTicketBody
        ticketSource={screen.ticketSource}
        intakeState={intakeState}
        intakesLoading={screen.intakesLoading ?? false}
        ticketIntakes={screen.ticketIntakes}
        jiraChoiceBlock={screen.jiraChoiceBlock}
        linearChoiceBlock={screen.linearChoiceBlock}
        linearSelectable={linearSelectable}
        linearLoading={catalogLoaded ? false : screen.linearFeatureLoading}
        jiraAvailable={jiraAvailable}
        continueLabel={screen.continueLabel}
        continueDisabled={continueButton.disabled}
        continueLoading={continueButton.loading}
        continueLoadingText={continueButton.loadingText}
        saving={screen.saving}
        jiraConnected={screen.jiraConnected}
        jiraProjects={screen.jiraProjects}
        jiraProjectsLoading={screen.jiraProjectsLoading}
        jiraProjectsError={screen.jiraProjectsError}
        jiraProjectId={screen.jiraProjectId}
        linearConnected={screen.linearConnected}
        linearProjects={screen.linearProjects}
        linearProjectsLoading={screen.linearProjectsLoading}
        linearProjectsError={screen.linearProjectsError}
        linearProjectIds={screen.linearProjectIds}
        onSelectTicketSource={screen.onSelectTicketSource}
        onAnalyzeTickets={screen.onAnalyzeTickets}
        onConnectJira={screen.onConnectJira}
        onSelectJiraProject={screen.onSelectJiraProject}
        onRetryJiraProjects={screen.onRetryJiraProjects}
        onConnectLinear={screen.onConnectLinear}
        onToggleLinearProject={screen.onToggleLinearProject}
        onRetryLinearProjects={screen.onRetryLinearProjects}
      />
    </FirstRunShell>
  );
}
