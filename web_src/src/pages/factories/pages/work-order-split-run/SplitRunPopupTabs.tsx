import { useState, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
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
  sourceOnly?: boolean;
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
  tab,
  onTabChange,
  canUpdate,
  footerActions,
  resultFooter,
  sidebarNote,
  analysis,
  sourceOnly = false,
  header,
}: SplitRunPopupTabsProps) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const files = liveWorkOrder.isSuccess ? liveWorkOrder.data?.files : undefined;
  const [streamTick, setStreamTick] = useState("");
  const follow = useFollowLogScroll<HTMLOListElement>(runningSplitRunPhaseId(fixture.phases), streamTick, {
    resumeOnBottom: true,
  });
  const description = (
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
  );
  const showAutomations = refinePopupShowsAutomations({ footerKind: fixture.footer.kind, sourceOnly });
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
