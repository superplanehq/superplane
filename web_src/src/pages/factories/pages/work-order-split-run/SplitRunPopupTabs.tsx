import { useState, type ReactNode } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
import { ClassicWorkOrderSplitRunOverview } from "./ClassicWorkOrderSplitRunOverview";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { splitRunStatusLabel, type SplitRunFixture } from "./splitRunMocks";
import { hasActivePullRequestActivity, refinePopupShowsAutomations, type SplitRunPopupTab } from "./splitRunPopupModel";
import { displayStatusForLineStatus } from "./splitRunWorkOrderDisplay";
import { useFollowLogScroll } from "./useFollowLogScroll";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import type { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { WorkOrderSplitRunBody } from "./WorkOrderSplitRunBody";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

type SplitRunPopupTabsProps = {
  mode?: "classic" | "analysis";
  fixture: SplitRunFixture;
  edits: ReturnType<typeof useSplitRunWorkOrderEdits>;
  popupData: ReturnType<typeof useSplitRunPopupData>;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  lineId?: string;
  tab: SplitRunPopupTab;
  onTabChange: (tab: SplitRunPopupTab) => void;
  canUpdate: boolean;
  footerActions: SplitRunFooterActions;
  resultFooter?: ReactNode;
  sidebarNote?: ReactNode;
  analysis?: IntentAnalysisChat;
  sessionLookupError?: string;
  header: (views: ReactNode) => ReactNode;
};

export function SplitRunPopupTabs({
  mode = "analysis",
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
  sessionLookupError,
  header,
}: SplitRunPopupTabsProps) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const files = liveWorkOrder.isSuccess ? liveWorkOrder.data?.files : undefined;
  const [streamTick, setStreamTick] = useState("");
  const follow = useFollowLogScroll<HTMLOListElement>(runningSplitRunPhaseId(fixture.phases), streamTick, {
    resumeOnBottom: true,
  });
  const description =
    mode === "classic" ? (
      <ClassicWorkOrderSplitRunOverview
        description={edits.description}
        artifacts={popupData.artifacts}
        artifactsLoading={popupData.artifactsLoading}
        pullRequests={popupData.pullRequests}
        pullRequestsLoading={popupData.pullRequestsLoading}
        pullRequestsError={popupData.pullRequestsError}
        checks={fixture.checks}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        orderId={orderId}
        orderNumber={orderNumber}
        files={files}
        expandFirstCheck={fixture.footer.kind === "draft"}
        canEditDescription={edits.canEditDescription}
        descriptionBusy={edits.descriptionBusy}
        onDescriptionSave={edits.saveDescription}
        source={fixture.source}
        sessionLookupError={sessionLookupError}
        sidebarNote={sidebarNote}
      />
    ) : (
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
        factoryKey={factoryKey}
        orderId={orderId}
        orderNumber={orderNumber}
        files={files}
        expandFirstCheck={fixture.footer.kind === "draft"}
        resultFooter={resultFooter}
        analysis={analysis}
        source={fixture.source}
        showContextSidebar={fixture.footer.kind !== "draft"}
        sidebarNote={sidebarNote}
      />
    );
  const showAutomations = refinePopupShowsAutomations({ mode, footerKind: fixture.footer.kind });
  if (!showAutomations) {
    return (
      <>
        {header(null)}
        <div className="flex min-h-0 min-w-0 flex-1 flex-col">{description}</div>
      </>
    );
  }

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
          lineStatus={fixture.lineStatus}
          hasActivePullRequestActivity={hasActivePullRequestActivity(fixture)}
        />,
      )}
      <TabsContent value="description" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
        {description}
      </TabsContent>
      <TabsContent value="log" className="mt-0 flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden">
        <WorkOrderSplitRunBody
          organizationId={organizationId}
          factoryId={factoryId}
          factoryKey={factoryKey}
          orderId={orderId}
          orderNumber={orderNumber}
          lineId={lineId}
          fixture={fixture}
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
