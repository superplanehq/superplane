import { useMemo, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";
import { Button } from "@/components/ui/button";
import { useWorkOrder } from "@/hooks/useFactoryData";

import type { ManualCustomAutomation } from "../../lib/customAutomation";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
import { phasesWithRunArtifacts } from "./attachStreamArtifacts";
import { AutomationsConsoleVariant } from "./redesign/AutomationsConsoleVariant";
import { BacklogCustomRunCards } from "./BacklogCustomRunCards";
import { ANALYSIS_PLANNING_COPY } from "./useAnalysisPlanningSession";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import type { SplitRunFixture } from "./splitRunMocks";
import { refinePopupShowsAutomations } from "./splitRunPopupModel";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import type { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { AutomationRunChat } from "./AutomationRunChat";
import { ButtonGroup } from "@/components/ui/button-group";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { ListTree, MessageSquare } from "lucide-react";
import type { useAutomationChatSessions } from "./useAutomationChatSessions";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

type SplitRunPopupTabsProps = {
  fixture: SplitRunFixture;
  edits: ReturnType<typeof useSplitRunWorkOrderEdits>;
  popupData: ReturnType<typeof useSplitRunPopupData>;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  lineId?: string;
  canUpdate: boolean;
  manualCustomAutomations?: readonly ManualCustomAutomation[];
  backlogCustomRunsStatus?: { isError: boolean; isFetching: boolean; retry: () => void };
  footerActions: SplitRunFooterActions;
  resultFooter?: ReactNode;
  chatFooter?: ReactNode;
  sidebarNote?: ReactNode;
  analysis?: IntentAnalysisChat;
  sourceOnly?: boolean;
  sessionLookupError?: string;
  /** Decision note and actions for the console summary panel. */
  panelReview?: ReactNode;
  header: (views: ReactNode) => ReactNode;
  automationChats: ReturnType<typeof useAutomationChatSessions>;
};

function SplitRunPopupOverview({
  fixture,
  edits,
  popupData,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  files,
  resultFooter,
  sidebarNote,
  analysis,
  sourceOnly,
  automationRuns,
}: Pick<
  SplitRunPopupTabsProps,
  | "fixture"
  | "edits"
  | "popupData"
  | "organizationId"
  | "factoryId"
  | "factoryKey"
  | "orderId"
  | "orderNumber"
  | "resultFooter"
  | "sidebarNote"
  | "analysis"
  | "sourceOnly"
> & { files?: FilesFile[]; automationRuns?: ReactNode }) {
  return (
    <WorkOrderSplitRunOverview
      title={edits.title}
      description={edits.description}
      artifacts={popupData.artifacts}
      artifactsLoading={popupData.artifactsLoading}
      pullRequests={popupData.pullRequests}
      pullRequestsLoading={popupData.pullRequestsLoading}
      pullRequestsError={popupData.pullRequestsError}
      checks={fixture.checks}
      isAnalyzing={fixture.footer.note?.headline === SPLIT_RUN_ANALYZING_NOTE.headline}
      organizationId={organizationId}
      factoryId={factoryId}
      factoryKey={factoryKey}
      orderId={orderId}
      orderNumber={orderNumber}
      files={files}
      expandFirstCheck={fixture.footer.kind === "draft"}
      resultFooter={resultFooter}
      analysis={analysis}
      source={fixture.source}
      showContextSidebar={sourceOnly || fixture.footer.kind !== "draft"}
      sourceOnly={sourceOnly}
      canEditDescription={sourceOnly ? edits.canEditDescription : undefined}
      descriptionBusy={sourceOnly ? edits.descriptionBusy : undefined}
      onDescriptionSave={sourceOnly ? edits.saveDescription : undefined}
      sidebarNote={sidebarNote}
      automationRuns={automationRuns}
    />
  );
}

/**
 * Unified task popup content. One timeline tells the task's life: creation
 * with the description, analysis, implement, verify, done. The sticky
 * summary panel carries status, owner, spend, outputs, and the decision
 * actions. A draft with Planning on keeps the refinement view instead.
 */
export function SplitRunPopupTabs(props: SplitRunPopupTabsProps) {
  const { automationChats: chats, organizationId, factoryId, orderId } = props;
  const navigation = <AutomationChatNavigation chats={chats} />;
  if (!chats.selected || !organizationId || !factoryId || !orderId) {
    return <SplitRunPopupContent {...props} header={() => props.header(navigation)} />;
  }
  return (
    <AutomationChatTaskView
      {...props}
      organizationId={organizationId}
      factoryId={factoryId}
      orderId={orderId}
      phase={chats.selected}
      navigation={navigation}
    />
  );
}

function AutomationChatTaskView({
  phase,
  navigation,
  ...props
}: SplitRunPopupTabsProps & {
  organizationId: string;
  factoryId: string;
  orderId: string;
  phase: NonNullable<SplitRunPopupTabsProps["automationChats"]["selected"]>;
  navigation: ReactNode;
}) {
  const order = useWorkOrder(props.organizationId, props.factoryId, props.orderId);
  return (
    <>
      {props.header(navigation)}
      <AutomationRunChat
        key={phase.runId}
        organizationId={props.organizationId}
        factoryId={props.factoryId}
        workOrderId={props.orderId}
        phase={phase}
        canUpdate={props.canUpdate}
        description={props.edits.description}
        files={order.data?.files}
        source={props.fixture.source}
      />
      {props.chatFooter}
    </>
  );
}

function AutomationChatNavigation({ chats }: { chats: SplitRunPopupTabsProps["automationChats"] }) {
  if (!chats.chats.length) return null;
  const views = [
    ...chats.chats.map((phase, index) => ({
      value: phase.runId!,
      label: chats.chats.length === 1 ? "Chat" : `${phase.componentName} · ${chats.chats.length - index}`,
      icon: MessageSquare,
    })),
    { value: "activity", label: "Timeline", icon: ListTree },
  ];
  return (
    <ButtonGroup aria-label="Task view">
      {views.map(({ value, label, icon: Icon }) => (
        <Tooltip key={value}>
          <TooltipTrigger asChild>
            <Button
              type="button"
              variant="outline"
              size="sm"
              aria-label={label}
              aria-pressed={chats.choice === value}
              className="aria-pressed:bg-accent aria-pressed:text-accent-foreground"
              onClick={() => chats.setChoice(value)}
            >
              <Icon className="size-4" aria-hidden />
              {label}
            </Button>
          </TooltipTrigger>
          <TooltipContent side="bottom">{label}</TooltipContent>
        </Tooltip>
      ))}
    </ButtonGroup>
  );
}

function SplitRunPopupContent({
  fixture,
  edits,
  popupData,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  canUpdate,
  manualCustomAutomations,
  backlogCustomRunsStatus,
  footerActions,
  resultFooter,
  sidebarNote,
  analysis,
  sourceOnly = false,
  sessionLookupError,
  panelReview,
  header,
}: SplitRunPopupTabsProps) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const files = liveWorkOrder.isSuccess ? liveWorkOrder.data?.files : undefined;
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const consoleFixture = useMemo(() => {
    const phases = phasesWithRunArtifacts(fixture.phases, artifactIndex);
    return phases === fixture.phases ? fixture : { ...fixture, phases };
  }, [artifactIndex, fixture]);
  const showAutomations = refinePopupShowsAutomations({ footerKind: fixture.footer.kind, sourceOnly });
  const lookupErrorNote = sessionLookupErrorNote(sessionLookupError);
  const historyErrorNote = backlogHistoryErrorNote(backlogCustomRunsStatus);
  if (!showAutomations) {
    return (
      <>
        {header(null)}
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {lookupErrorNote}
          {historyErrorNote}
          <SplitRunPopupOverview
            fixture={fixture}
            edits={edits}
            popupData={popupData}
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
            orderId={orderId}
            orderNumber={orderNumber}
            files={files}
            resultFooter={resultFooter}
            sidebarNote={sidebarNote}
            analysis={analysis}
            sourceOnly={sourceOnly}
            automationRuns={
              consoleFixture.phases.some((phase) => phase.id.startsWith("backlog-custom-")) ? (
                <BacklogCustomRunCards
                  fixture={consoleFixture}
                  organizationId={organizationId}
                  factoryId={factoryId}
                  factoryKey={factoryKey}
                  orderId={orderId}
                  orderNumber={orderNumber}
                  canUpdate={canUpdate}
                  footerActions={footerActions}
                  manualCustomAutomations={manualCustomAutomations}
                />
              ) : undefined
            }
          />
        </div>
      </>
    );
  }

  return (
    <>
      {header(null)}
      <div className="min-h-0 min-w-0 flex-1 overflow-y-auto px-5 py-4">
        {lookupErrorNote}
        {historyErrorNote}
        <AutomationsConsoleVariant
          fixture={consoleFixture}
          organizationId={organizationId}
          factoryId={factoryId}
          orderId={orderId}
          factoryKey={factoryKey}
          orderNumber={orderNumber}
          lineId={lineId}
          taskDescription={edits.description}
          canEditDescription={edits.canEditDescription}
          descriptionBusy={edits.descriptionBusy}
          onDescriptionSave={edits.saveDescription}
          owner={edits.owner}
          assigneeIds={edits.assigneeIds}
          canEditOwner={edits.canEdit}
          ownerBusy={edits.ownerBusy}
          onOwnerSave={edits.saveOwner}
          source={fixture.source}
          files={files}
          pullRequests={popupData.pullRequests}
          artifacts={popupData.artifacts}
          panelReview={panelReview}
          canStopRun={Boolean(canUpdate && organizationId && factoryId && orderId)}
          manualCustomAutomations={manualCustomAutomations}
          actionBusy={footerActions.busy}
          onStopRun={(run) => void footerActions.handleStopAutomation(run)}
          onRerunStep={(phase) =>
            void footerActions.handleStop("rerun-step", {
              kind: "failed",
              lineName: fixture.lineName,
              stepIndex: phase.stepIndex,
            })
          }
        />
      </div>
    </>
  );
}

function sessionLookupErrorNote(error?: string) {
  if (!error) {
    return null;
  }
  return (
    <p className="shrink-0 px-8 pt-4 text-[13px] text-destructive" role="alert">
      {ANALYSIS_PLANNING_COPY.failedLoadRefinement}
    </p>
  );
}

function backlogHistoryErrorNote(status: SplitRunPopupTabsProps["backlogCustomRunsStatus"]) {
  if (!status?.isError) return null;
  return (
    <div role="alert" className="flex shrink-0 items-center justify-between gap-3 px-5 py-3 text-sm text-destructive">
      <span>Could not load Backlog automation runs. Retry to update the run history.</span>
      <Button variant="outline" size="sm" disabled={status.isFetching} onClick={status.retry}>
        {status.isFetching ? "Retrying…" : "Retry"}
      </Button>
    </div>
  );
}
