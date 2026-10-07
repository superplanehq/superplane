import { LoadingButton } from "@/components/ui/loading-button";

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

export function FirstRunTicketsScreen(props: FirstRunTicketsScreenProps) {
  const {
    ticketSource,
    chrome,
    sphere,
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
    jiraConnected,
    jiraProjectId,
    linearAvailable,
    linearChoiceBlock,
    linearConnected,
    linearProjectIds,
    linearProjectsLoading,
    linearProjectsError,
  });
  const continueButton = ticketContinueButton({ canAnalyze, continuePending, saving, savingLabel });

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome} busy={saving} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <FirstRunTicketChoices
          ticketSource={ticketSource}
          saving={saving}
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
          block={linearChoiceBlock}
          loading={copy.linearLookupLoading}
          failed={copy.linearLookupFailed}
          testId="first-run-linear-choice-notice"
        />
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
