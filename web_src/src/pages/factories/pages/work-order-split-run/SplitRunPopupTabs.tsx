import { useMemo, type ReactNode } from "react";

import type { FilesFile } from "@/api-client";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useWorkOrder } from "@/hooks/useFactoryData";

import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";
import { phasesWithRunArtifacts } from "./attachStreamArtifacts";
import { AutomationsConsoleVariant } from "./redesign/AutomationsConsoleVariant";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import { splitRunStatusLabel, type SplitRunFixture } from "./splitRunMocks";
import { hasActivePullRequestActivity, refinePopupShowsAutomations, type SplitRunPopupTab } from "./splitRunPopupModel";
import { displayStatusForLineStatus } from "./splitRunWorkOrderDisplay";
import type { SplitRunFooterActions } from "./useSplitRunFooterActions";
import type { useSplitRunPopupData } from "./useSplitRunPopupData";
import type { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
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
  sessionLookupError?: string;
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
  sessionLookupError,
  header,
}: SplitRunPopupTabsProps) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const files = liveWorkOrder.isSuccess ? liveWorkOrder.data?.files : undefined;
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
          {description}
        </div>
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
        {lookupErrorNote}
        {description}
      </TabsContent>
      <TabsContent
        value="log"
        forceMount
        className="mt-0 flex min-h-0 min-w-0 flex-1 flex-col overflow-hidden data-[state=inactive]:hidden"
      >
        <div className="min-h-0 min-w-0 flex-1 overflow-y-auto px-5 py-4">
          <AutomationsConsoleVariant
            fixture={consoleFixture}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            lineId={lineId}
            canStopRun={Boolean(canUpdate && organizationId && factoryId && orderId)}
            actionBusy={footerActions.busy}
            onStopRun={(run) => void footerActions.handleStopAutomation(run)}
          />
        </div>
      </TabsContent>
    </Tabs>
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
