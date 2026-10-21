import { LoadingButton } from "@/components/ui/loading-button";
import { EMPTY_LINEAR_INTAKE_SCOPE, type LinearIntakeScope } from "../../linearIntakeScope";

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
  githubConnected?: boolean;
  githubAvailable?: boolean;
  jiraConnected?: boolean;
  jiraProjects?: FirstRunJiraProject[];
  jiraProjectsLoading?: boolean;
  jiraProjectsError?: boolean;
  jiraProjectId?: string;
  linearConnected?: boolean;
  organizationId?: string;
  linearIntegrationId?: string;
  linearScope?: LinearIntakeScope;
  onSelectTicketSource: (source: FirstRunTicketSource) => void;
  onAnalyzeTickets: () => void;
  onConnectGitHub?: () => void;
  onConnectJira?: () => void;
  onSelectJiraProject?: (id: string) => void;
  onRetryJiraProjects?: () => void;
  onConnectLinear?: () => void;
  onChangeLinearScope?: (scope: LinearIntakeScope) => void;
};

const TICKET_SCREEN_DEFAULTS = {
  linearAvailable: true,
  linearFeatureLoading: false,
  linearChoiceBlock: null as FirstRunFlaggedChoiceBlock | null,
  continueLabel: FIRST_RUN_COPY.tickets.analyze,
  continuePending: false,
  saving: false,
  savingLabel: FIRST_RUN_COPY.finish.saving,
  githubConnected: false,
  githubAvailable: true,
  jiraConnected: false,
  jiraProjects: [] as FirstRunJiraProject[],
  jiraProjectsLoading: false,
  jiraProjectsError: false,
  jiraProjectId: "",
  linearConnected: false,
  organizationId: "",
  linearIntegrationId: "",
  linearScope: EMPTY_LINEAR_INTAKE_SCOPE,
};

function ticketScanAllowed(args: {
  ticketSource: FirstRunTicketSource | null;
  githubConnected: boolean;
  jiraConnected: boolean;
  jiraProjectId: string;
  linearAvailable: boolean;
  linearChoiceBlock: FirstRunFlaggedChoiceBlock | null;
  linearConnected: boolean;
  linearScope: LinearIntakeScope;
}): boolean {
  const linearSelectionBlocked =
    Boolean(args.linearChoiceBlock) || (args.ticketSource === "linear" && !args.linearAvailable);
  return (
    !linearSelectionBlocked &&
    canAnalyzeTicketSource({
      ticketSource: args.ticketSource,
      githubConnected: args.githubConnected,
      jiraConnected: args.jiraConnected,
      jiraProjectId: args.jiraProjectId,
      linearConnected: args.linearConnected,
      linearScope: args.linearScope,
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
    githubConnected,
    githubAvailable,
    jiraConnected,
    jiraProjects,
    jiraProjectsLoading,
    jiraProjectsError,
    jiraProjectId,
    linearConnected,
    organizationId,
    linearIntegrationId,
    linearScope,
    onSelectTicketSource,
    onAnalyzeTickets,
    onConnectGitHub,
    onConnectJira,
    onSelectJiraProject,
    onRetryJiraProjects,
    onConnectLinear,
    onChangeLinearScope,
  } = { ...TICKET_SCREEN_DEFAULTS, ...props };
  const copy = FIRST_RUN_COPY.tickets;
  const canAnalyze = ticketScanAllowed({
    ticketSource,
    githubConnected,
    jiraConnected,
    jiraProjectId,
    linearAvailable,
    linearChoiceBlock,
    linearConnected,
    linearScope,
  });
  const continueButton = ticketContinueButton({ canAnalyze, continuePending, saving, savingLabel });

  return (
    <FirstRunShell testId="first-run-tickets" chrome={chrome} busy={saving} sphere={sphere} visual="preview">
      <FirstRunHeading headline={copy.headline}>
        <p className="text-[13px] text-muted-foreground">{copy.intro}</p>
      </FirstRunHeading>

      <div className="mt-8 space-y-4">
        <FirstRunTicketChoices
          githubAvailable={githubAvailable}
          ticketSource={ticketSource}
          saving={saving}
          githubConnected={githubConnected}
          jiraConnected={jiraConnected}
          jiraProjects={jiraProjects}
          jiraProjectsLoading={jiraProjectsLoading}
          jiraProjectsError={jiraProjectsError}
          jiraProjectId={jiraProjectId}
          linearAvailable={linearAvailable}
          linearFeatureLoading={linearFeatureLoading}
          linearConnected={linearConnected}
          organizationId={organizationId}
          linearIntegrationId={linearIntegrationId}
          linearScope={linearScope}
          onSelectTicketSource={onSelectTicketSource}
          onConnectGitHub={onConnectGitHub}
          onConnectJira={onConnectJira}
          onSelectJiraProject={onSelectJiraProject}
          onRetryJiraProjects={onRetryJiraProjects}
          onConnectLinear={onConnectLinear}
          onChangeLinearScope={onChangeLinearScope}
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
