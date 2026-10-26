import { LoadingButton } from "@/components/ui/loading-button";
import { useState } from "react";

import { FIRST_RUN_COPY } from "./firstRunCopy";
import { FirstRunTicketSourceList } from "./firstRunTicketChoices";
import { FirstRunTicketIntakeStepper } from "./FirstRunTicketIntakeStepper";
import { type FirstRunTicketProjectState, ticketSourceName } from "./firstRunTicketIntakeSteps";
import { FirstRunHeading, FirstRunShell } from "./FirstRunShell";
import type { FirstRunSphereProps } from "./FirstRunSpherePane";
import { canAnalyzeTicketSource } from "./firstRunTicketSource";
import type { FirstRunChrome, FirstRunTicketSource } from "./firstRunTypes";

export type FirstRunJiraProject = { id?: string; name?: string };

export type FirstRunFlaggedChoiceBlock = "loading" | "lookup-failed";

type FirstRunTicketsScreenProps = {
  /** How many existing issues the first import takes. */
  initialImportLimit?: number;
  skipInitialImport?: boolean;
  /** Absent on the first workspace: that import always runs. */
  onSkipInitialImportChange?: (skip: boolean) => void;
  automaticIntake?: boolean;
  onAutomaticIntakeChange?: (on: boolean) => void;
  ticketSource: FirstRunTicketSource | null;
  /**
   * Open on the source list even when a source is already selected, for
   * example a default the user did not choose. The card opens on click.
   */
  startOnSourceList?: boolean;
  /** Repository the GitHub intake reads, e.g. "acme/payments". */
  repository?: string;
  chrome?: FirstRunChrome;
  sphere?: FirstRunSphereProps;
  linearAvailable?: boolean;
  linearFeatureLoading?: boolean;
  linearChoiceBlock?: FirstRunFlaggedChoiceBlock | null;
  continueLabel?: string;
  /** True while this screen waits to learn whether the agent screen is next. */
  continuePending?: boolean;
  /** True while this screen provisions the workspace, on the last screen. */
  saving?: boolean;
  savingLabel?: string;
  /** True while the connect redirect for the chosen source opens. */
  connecting?: boolean;
  githubConnected?: boolean;
  githubAvailable?: boolean;
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
  onConnectGitHub?: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onToggleLinearProject?: (id: string) => void;
  onRetryLinearProjects?: () => void;
};

const TICKET_SCREEN_DEFAULTS = {
  initialImportLimit: 3,
  skipInitialImport: false,
  automaticIntake: true,
  linearAvailable: true,
  linearFeatureLoading: false,
  linearChoiceBlock: null as FirstRunFlaggedChoiceBlock | null,
  continueLabel: FIRST_RUN_COPY.tickets.analyze,
  continuePending: false,
  saving: false,
  savingLabel: FIRST_RUN_COPY.finish.saving,
  connecting: false,
  githubConnected: false,
  githubAvailable: true,
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
  githubConnected: boolean;
  jiraConnected: boolean;
  jiraProjectId: string;
  linearAvailable: boolean;
  linearChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  linearConnected: boolean;
  linearProjectIds: string[];
  linearProjectsLoading: boolean;
  linearProjectsError: boolean;
}): boolean {
  const linearSelectionBlocked =
    Boolean(args.linearChoiceBlock) || (args.ticketSource === "linear" && !args.linearAvailable);
  if (args.ticketSource === "linear" && (args.linearProjectsLoading || args.linearProjectsError)) return false;
  return (
    !linearSelectionBlocked &&
    canAnalyzeTicketSource({
      ticketSource: args.ticketSource,
      githubConnected: args.githubConnected,
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

function sourceConnected(
  source: FirstRunTicketSource,
  connected: { github: boolean; jira: boolean; linear: boolean },
): boolean {
  if (source === "jira") return connected.jira;
  if (source === "linear") return connected.linear;
  return connected.github;
}

/**
 * Two parts on one screen. First the list of ticket sources. After a
 * choice, the card with the steps for that source: connect, choose the
 * project, set intake settings, then the final action. Back walks the
 * card backwards and then returns to the list.
 */
function useTicketScreenPhase(ticketSource: FirstRunTicketSource | null, startOnSourceList: boolean) {
  const [browsing, setBrowsing] = useState(startOnSourceList);
  const [confirmedFor, setConfirmedFor] = useState<FirstRunTicketSource | null>(null);
  const projectsConfirmed = confirmedFor !== null && confirmedFor === ticketSource;
  const listPhase = ticketSource === null || browsing;
  return {
    listPhase,
    projectsConfirmed,
    choose: () => {
      setBrowsing(false);
      setConfirmedFor(null);
    },
    confirmProjects: () => setConfirmedFor(ticketSource),
    back: () => {
      if (projectsConfirmed) {
        setConfirmedFor(null);
        return;
      }
      setBrowsing(true);
    },
  };
}

type DefaultedTicketsProp = keyof typeof TICKET_SCREEN_DEFAULTS;

type ResolvedTicketsProps = Omit<FirstRunTicketsScreenProps, DefaultedTicketsProp> &
  Required<Pick<FirstRunTicketsScreenProps, DefaultedTicketsProp>>;

type TicketScreenPhase = ReturnType<typeof useTicketScreenPhase>;

export function FirstRunTicketsScreen(props: FirstRunTicketsScreenProps) {
  const resolved: ResolvedTicketsProps = { ...TICKET_SCREEN_DEFAULTS, ...props };
  const { ticketSource, chrome, sphere, saving, linearChoiceBlock } = resolved;
  const copy = FIRST_RUN_COPY.tickets;
  const phase = useTicketScreenPhase(ticketSource, Boolean(resolved.startOnSourceList));
  const stepperSource = phase.listPhase ? null : ticketSource;
  // On the card, Back walks the steps and then returns to the list.
  const screenChrome = chrome && stepperSource ? { ...chrome, onBack: phase.back } : chrome;

  return (
    <FirstRunShell testId="first-run-tickets" chrome={screenChrome} busy={saving} sphere={sphere} visual="preview">
      <FirstRunHeading headline={stepperSource ? copy.setupHeadline(ticketSourceName(stepperSource)) : copy.headline}>
        <p className="text-[13px] text-muted-foreground">{stepperSource ? copy.setupIntro : copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        {stepperSource ? (
          <TicketIntakeCard source={stepperSource} phase={phase} {...resolved} />
        ) : (
          <FirstRunTicketSourceList
            githubAvailable={resolved.githubAvailable}
            ticketSource={ticketSource}
            saving={saving}
            githubConnected={resolved.githubConnected}
            jiraConnected={resolved.jiraConnected}
            linearAvailable={resolved.linearAvailable}
            linearFeatureLoading={resolved.linearFeatureLoading}
            linearConnected={resolved.linearConnected}
            onSelectTicketSource={(source) => {
              phase.choose();
              resolved.onSelectTicketSource(source);
            }}
          />
        )}
        <FirstRunChoiceNotice
          block={linearChoiceBlock}
          loading={copy.linearLookupLoading}
          failed={copy.linearLookupFailed}
          testId="first-run-linear-choice-notice"
        />
      </div>
    </FirstRunShell>
  );
}

function TicketIntakeCard({
  source,
  phase,
  ...p
}: ResolvedTicketsProps & { source: FirstRunTicketSource; phase: TicketScreenPhase }) {
  const canAnalyze = ticketScanAllowed({
    ticketSource: source,
    githubConnected: p.githubConnected,
    jiraConnected: p.jiraConnected,
    jiraProjectId: p.jiraProjectId,
    linearAvailable: p.linearAvailable,
    linearChoiceBlock: p.linearChoiceBlock,
    linearConnected: p.linearConnected,
    linearProjectIds: p.linearProjectIds,
    linearProjectsLoading: p.linearProjectsLoading,
    linearProjectsError: p.linearProjectsError,
  });
  const continueButton = ticketContinueButton({
    canAnalyze,
    continuePending: p.continuePending,
    saving: p.saving,
    savingLabel: p.savingLabel,
  });
  const projects: FirstRunTicketProjectState = {
    jiraProjects: p.jiraProjects,
    jiraProjectsLoading: p.jiraProjectsLoading,
    jiraProjectsError: p.jiraProjectsError,
    jiraProjectId: p.jiraProjectId,
    linearProjects: p.linearProjects,
    linearProjectsLoading: p.linearProjectsLoading,
    linearProjectsError: p.linearProjectsError,
    linearProjectIds: p.linearProjectIds,
  };
  const connectFor: Record<FirstRunTicketSource, (() => void) | undefined> = {
    "github-issues": p.onConnectGitHub,
    jira: p.onConnectJira,
    linear: p.onConnectLinear,
  };
  return (
    <FirstRunTicketIntakeStepper
      source={source}
      connected={sourceConnected(source, {
        github: p.githubConnected,
        jira: p.jiraConnected,
        linear: p.linearConnected,
      })}
      connecting={p.connecting}
      repository={p.repository}
      projects={projects}
      projectsConfirmed={phase.projectsConfirmed}
      saving={p.saving}
      settings={{
        initialImportLimit: p.initialImportLimit,
        skipInitialImport: p.skipInitialImport,
        onSkipInitialImportChange: p.onSkipInitialImportChange,
        automaticIntake: p.automaticIntake,
        onAutomaticIntakeChange: p.onAutomaticIntakeChange,
      }}
      finishButton={
        <LoadingButton
          type="button"
          className="w-full"
          disabled={continueButton.disabled}
          loading={continueButton.loading}
          loadingText={continueButton.loadingText}
          onClick={p.onAnalyzeTickets}
          data-testid="first-run-analyze-tickets"
        >
          {p.continueLabel}
        </LoadingButton>
      }
      onConnect={connectFor[source]}
      onSelectJiraProject={p.onSelectJiraProject}
      onRetryJiraProjects={p.onRetryJiraProjects}
      onToggleLinearProject={p.onToggleLinearProject}
      onRetryLinearProjects={p.onRetryLinearProjects}
      onConfirmProjects={phase.confirmProjects}
    />
  );
}

function choiceNoticeCopy(block: FirstRunFlaggedChoiceBlock | null, loading: string, failed: string): string | null {
  if (block === "lookup-failed") return failed;
  if (block === "loading") return loading;
  return null;
}

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
  );
}
