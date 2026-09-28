import { useMemo, useState, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
import { phasesWithRunArtifacts } from "./attachStreamArtifacts";
import { AutomationsConsoleVariant } from "./redesign/AutomationsConsoleVariant";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { splitRunStatusLabel, type SplitRunFixture } from "./splitRunMocks";
import {
  classicSplitRunFixture,
  hasActivePullRequestActivity,
  refinePopupShowsAutomations,
  type SplitRunPopupTab,
} from "./splitRunPopupModel";
import { displayStatusForLineStatus } from "./splitRunWorkOrderDisplay";
import { useFollowLogScroll } from "./useFollowLogScroll";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import type { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { WorkOrderSplitRunBody } from "./WorkOrderSplitRunBody";
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
  /** Show the pre-console Task and Automations tabs instead of the console. */
  classic: boolean;
  tab: SplitRunPopupTab;
  onTabChange: (tab: SplitRunPopupTab) => void;
  canUpdate: boolean;
  footerActions: SplitRunFooterActions;
  resultFooter?: ReactNode;
  sidebarNote?: ReactNode;
  analysis?: IntentAnalysisChat;
  sourceOnly?: boolean;
  sessionLookupError?: string;
  /** Decision note and actions for the console summary panel. */
  panelReview?: ReactNode;
  header: (views: ReactNode) => ReactNode;
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
> & { files?: FilesFile[] }) {
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
    />
  );
}

/**
 * Unified task popup content. One timeline tells the task's life: creation
 * with the description, analysis, implement, verify, done. The sticky
 * summary panel carries status, owner, spend, outputs, and the decision
 * actions. A draft with Planning on keeps the refinement view instead.
 */
export function SplitRunPopupTabs({
  fixture,
  edits,
  popupData,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  classic,
  tab,
  onTabChange,
  canUpdate,
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
  if (!showAutomations) {
    return (
      <>
        {header(null)}
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">
          {lookupErrorNote}
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
          />
        </div>
      </>
    );
  }

  if (classic) {
    return (
      <SplitRunPopupClassicTabs
        fixture={fixture}
        edits={edits}
        popupData={popupData}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        orderId={orderId}
        orderNumber={orderNumber}
        lineId={lineId}
        tab={tab}
        onTabChange={onTabChange}
        canUpdate={canUpdate}
        footerActions={footerActions}
        resultFooter={resultFooter}
        sidebarNote={sidebarNote}
        analysis={analysis}
        sourceOnly={sourceOnly}
        lookupErrorNote={lookupErrorNote}
        files={files}
        header={header}
      />
    );
  }

  return (
    <>
      {header(null)}
      <div className="min-h-0 min-w-0 flex-1 overflow-y-auto px-5 py-4">
        {lookupErrorNote}
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
          source={fixture.source}
          files={files}
          pullRequests={popupData.pullRequests}
          panelReview={panelReview}
          canStopRun={Boolean(canUpdate && organizationId && factoryId && orderId)}
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

/** The pre-console popup: Task and Automations tabs. Kept for orgs without the Task Console feature. */
function SplitRunPopupClassicTabs({
  fixture,
  edits,
  popupData,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  tab,
  onTabChange,
  canUpdate,
  footerActions,
  resultFooter,
  sidebarNote,
  analysis,
  sourceOnly,
  lookupErrorNote,
  files,
  header,
}: Omit<SplitRunPopupTabsProps, "classic" | "panelReview" | "sessionLookupError"> & {
  lookupErrorNote: ReactNode;
  files?: FilesFile[];
}) {
  const classicFixture = useMemo(() => classicSplitRunFixture(fixture), [fixture]);
  const [streamTick, setStreamTick] = useState("");
  const follow = useFollowLogScroll<HTMLOListElement>(runningSplitRunPhaseId(classicFixture.phases), streamTick, {
    resumeOnBottom: true,
  });
  return (
    <Tabs
      value={tab}
      onValueChange={(value) => {
        if (value === "description" || value === "log") {
          onTabChange(value);
        }
      }}
      className="flex min-h-0 min-w-0 flex-1 flex-col"
    >
      {header(
        <SplitRunPopupViewTabs
          lineStatus={classicFixture.lineStatus}
          hasActivePullRequestActivity={hasActivePullRequestActivity(classicFixture)}
        />,
      )}
      <TabsContent value="description" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
        {lookupErrorNote}
        <SplitRunPopupOverview
          fixture={classicFixture}
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
        />
      </TabsContent>
      <TabsContent
        value="log"
        forceMount
        className="mt-0 flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden data-[state=inactive]:hidden"
      >
        <WorkOrderSplitRunBody
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          orderId={orderId}
          orderNumber={orderNumber}
          lineId={lineId}
          fixture={classicFixture}
          canUpdate={canUpdate}
          footerActions={footerActions}
          follow={follow}
          onStreamTick={setStreamTick}
          files={files}
        />
      </TabsContent>
    </Tabs>
  );
}

const VIEW_TAB_CLASSNAME = "sp-popup-view-tab";

function SplitRunPopupViewTabs({
  lineStatus,
  hasActivePullRequestActivity: hasActivePRActivity,
}: {
  lineStatus: SplitRunFixture["lineStatus"];
  hasActivePullRequestActivity: boolean;
}) {
  const automationStatus = hasActivePRActivity ? "running" : displayStatusForLineStatus(lineStatus);
  const automationStatusLabel = hasActivePRActivity ? "Running" : splitRunStatusLabel(lineStatus);
  return (
    <TabsList aria-label="Task views">
      <TabsTrigger value="description" className={VIEW_TAB_CLASSNAME}>
        Task
      </TabsTrigger>
      <TabsTrigger value="log" className={VIEW_TAB_CLASSNAME}>
        <WorkOrderStatusIcon
          status={automationStatus}
          title={automationStatusLabel}
          className="size-3"
          data-testid="split-run-log-tab-dot"
          aria-hidden
        />
        Automations
      </TabsTrigger>
    </TabsList>
  );
}

function sessionLookupErrorNote(error?: string) {
  if (!error) {
    return null;
  }
  return (
    <p className="shrink-0 px-8 pt-4 text-[13px] text-destructive" role="alert">
      The refinement session did not load. Refresh the page to try again.
    </p>
  );
}
