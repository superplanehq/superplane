import { useState, type ReactNode } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
import { ClassicWorkOrderSplitRunOverview } from "./ClassicWorkOrderSplitRunOverview";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { splitRunStatusLabel, type SplitRunFixture } from "./splitRunMocks";
import type { SplitRunPopupTab } from "./splitRunPopupModel";
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
  analysis?: IntentAnalysisChat;
  sessionLookupError?: string;
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
  analysis,
  sessionLookupError,
}: SplitRunPopupTabsProps) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
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
        files={liveWorkOrder.data?.files}
        expandFirstCheck={fixture.footer.kind === "draft"}
        canEditDescription={edits.canEditDescription}
        descriptionBusy={edits.descriptionBusy}
        onDescriptionSave={edits.saveDescription}
        source={fixture.source}
        sessionLookupError={sessionLookupError}
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
        orderNumber={orderNumber}
        files={liveWorkOrder.data?.files}
        expandFirstCheck={fixture.footer.kind === "draft"}
        resultFooter={resultFooter}
        analysis={analysis}
        source={fixture.source}
        showContextSidebar={fixture.footer.kind !== "draft"}
      />
    );

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
      <div className="flex shrink-0 items-center gap-2 border-b border-border px-5 py-2">
        <TabsList aria-label="Task views">
          <TabsTrigger value="description">Description</TabsTrigger>
          <TabsTrigger value="log">
            <WorkOrderStatusIcon
              status={displayStatusForLineStatus(fixture.lineStatus)}
              title={splitRunStatusLabel(fixture.lineStatus)}
              className="size-3"
              data-testid="split-run-log-tab-dot"
              aria-hidden
            />
            Automations
          </TabsTrigger>
        </TabsList>
      </div>
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
        />
      </TabsContent>
    </Tabs>
  );
}
