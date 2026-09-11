import { useCallback, useEffect, useMemo, useRef, useState, type ReactNode } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { useExperimentalFeature } from "@/hooks/useExperimentalFeature";
import { useWorkOrder } from "@/hooks/useFactoryData";
import { FEATURE_FACTORY_DRAFT_START_MODEL } from "@/lib/experimentalFeatures";

import { CopyLinkButton } from "../../CopyLinkButton";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { DraftStartModelSelect } from "./DraftStartModelSelect";
import { DRAFT_START_MODEL_AUTO, draftStartModelPayload, phaseWithRunnerModel } from "./draftStartModel";
import { SplitRunReview } from "./SplitRunReview";
import { attachArtifactsToStream, type StreamArtifactIndex } from "./attachStreamArtifacts";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import { SPLIT_RUN_ANALYZING_NOTE } from "./splitRunFooter";
import {
  autoExpandedPhaseId,
  splitRunStatusLabel,
  type SplitRunFixture,
  type SplitRunPhase,
  type SplitRunPhaseId,
} from "./splitRunMocks";
import {
  defaultSplitRunPopupTab,
  SPLIT_RUN_POPUP_DIALOG_CLASSNAME,
  type SplitRunPopupTab,
  splitRunPhaseRunHref,
} from "./splitRunPopupModel";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions, type SplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { useFollowLogScroll } from "./useFollowLogScroll";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { useCurrentPopupDismiss } from "./useCurrentPopupDismiss";
import { WorkOrderStatusIcon } from "../../workOrders/WorkOrderStatusIcon";
import { displayStatusForLineStatus } from "./splitRunWorkOrderDisplay";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";
import { useAnalysisPlanningSession } from "./useAnalysisPlanningSession";
import type { IntentAnalysisChat } from "./WorkOrderIntentDocument";

/**
 * Absolute work-order permalink, so the popup copies the right link even
 * when it is shown without a route change (e.g. straight from a board card).
 * Falls back to the current address when identifiers are missing.
 */
function popupWorkOrderUrl(organizationId?: string, factoryKey?: string, orderNumber?: string, lineId?: string) {
  if (!organizationId || !factoryKey || !orderNumber) {
    return window.location.href;
  }
  return window.location.origin + workOrderDetailPath(organizationId, factoryKey, orderNumber, lineId);
}

function footerMutationHandlers(
  canUpdate: boolean,
  footerActions: SplitRunFooterActions,
  fixture: SplitRunFixture,
  onDismiss?: () => void,
) {
  if (!canUpdate) {
    return {};
  }
  return {
    onArchive: async () => {
      const archived = await footerActions.handleArchive();
      if (archived) {
        onDismiss?.();
      }
    },
    onReject: () => void footerActions.handleReject(),
    onBackToDraft: () => footerActions.handleBackToDraft(),
    onStop: (choice: Parameters<typeof footerActions.handleStop>[0]) =>
      void footerActions.handleStop(choice, {
        ...fixture.footer,
        lineName: fixture.lineName,
        stepIndex: fixture.currentStepIndex,
      }),
  };
}

type WorkOrderSplitRunBodyProps = {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  lineId?: string;
  fixture: SplitRunFixture;
  canUpdate?: boolean;
  footerActions: SplitRunFooterActions;
};

type SplitRunFollow = ReturnType<typeof useFollowLogScroll<HTMLOListElement>>;

/** Phase log for a work-order popup. The popup wraps this. */
export function WorkOrderSplitRunBody({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  fixture,
  canUpdate = true,
  footerActions,
  follow,
  onStreamTick,
}: WorkOrderSplitRunBodyProps & {
  follow: SplitRunFollow;
  onStreamTick: (tick: string) => void;
}) {
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
  const streamLengthsRef = useRef<Record<string, number>>({});
  const currentPhaseId = fixture.currentPhaseId;
  const expandedPhaseId = autoExpandedPhaseId(fixture);
  useEffect(() => {
    setOpenPhaseId(expandedPhaseId);
  }, [currentPhaseId, expandedPhaseId]);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const demoArtifacts = !organizationId;
  const onStreamLength = useCallback(
    (phaseId: string, length: number) => {
      if (streamLengthsRef.current[phaseId] === length) {
        return;
      }
      streamLengthsRef.current = { ...streamLengthsRef.current, [phaseId]: length };
      onStreamTick(fixture.phases.map((entry) => streamLengthsRef.current[entry.id] ?? 0).join(":"));
    },
    [fixture.phases, onStreamTick],
  );
  const liveOrder = Boolean(organizationId && factoryId && orderId);
  const automationStop = (entry: SplitRunPhase) => {
    const appId = entry.appId;
    const runId = entry.runId;
    if (!canUpdate || !liveOrder || entry.status !== "running" || !appId || !runId) {
      return undefined;
    }
    return () => void footerActions.handleStopAutomation({ appId, runId });
  };
  const automationRerun = (entry: SplitRunPhase) => {
    if (!canUpdate || !liveOrder || entry.status !== "failed" || entry.stepIndex == null) {
      return undefined;
    }
    return () =>
      void footerActions.handleStop("rerun-step", {
        kind: "failed",
        lineName: fixture.lineName,
        stepIndex: entry.stepIndex,
      });
  };

  return (
    <div className="relative flex min-h-0 min-w-0 flex-1 flex-col" data-testid="split-run-log-pane">
      <ol
        ref={follow.scrollRef}
        onScroll={follow.onScroll}
        className="min-h-0 min-w-0 flex-1 list-none space-y-2 overflow-x-hidden overflow-y-auto px-3 pb-3"
        data-testid="split-run-log-scroll"
      >
        {fixture.phases.map((entry) => (
          <li key={entry.id} className="min-w-0 first:mt-3">
            <SplitRunPhaseLogItem
              entry={entry}
              organizationId={organizationId}
              factoryKey={factoryKey}
              orderNumber={orderNumber}
              lineId={lineId}
              expanded={entry.id === openPhaseId}
              selectedNodeId={nodeId}
              onSelectNode={setNodeId}
              demoArtifacts={demoArtifacts}
              artifactIndex={artifactIndex}
              onStop={automationStop(entry)}
              onRerun={automationRerun(entry)}
              actionBusy={footerActions.busy}
              onStreamLength={onStreamLength}
              onToggle={() => {
                setNodeId(null);
                setOpenPhaseId((current) => (current === entry.id ? null : entry.id));
              }}
            />
          </li>
        ))}
      </ol>
      {follow.following ? null : (
        <JumpToLatestPill onJumpToLatest={() => follow.setFollowing(true)} testId="split-run-older" />
      )}
    </div>
  );
}

function SplitRunPhaseLogItem({
  entry,
  organizationId,
  factoryKey,
  orderNumber,
  lineId,
  expanded,
  selectedNodeId,
  onSelectNode,
  demoArtifacts,
  artifactIndex,
  onStop,
  onRerun,
  actionBusy,
  onStreamLength,
  onToggle,
}: {
  entry: SplitRunPhase;
  organizationId?: string;
  factoryKey?: string;
  orderNumber?: string;
  lineId?: string;
  expanded: boolean;
  selectedNodeId?: string | null;
  onSelectNode: (nodeId: string) => void;
  demoArtifacts: boolean;
  artifactIndex: StreamArtifactIndex;
  onStop?: () => void;
  onRerun?: () => void;
  actionBusy: boolean;
  onStreamLength: (phaseId: string, length: number) => void;
  onToggle: () => void;
}) {
  const [usageOpen, setUsageOpen] = useState(false);
  const live = useSplitRunLiveCanvas(organizationId, expanded || usageOpen ? entry : undefined);
  const visual = useMemo(() => resolveSplitRunVisual(entry, live, { demoArtifacts }), [demoArtifacts, entry, live]);
  const stream = useMemo(
    () => attachArtifactsToStream(visual.stream, artifactIndex, entry.runId),
    [artifactIndex, entry.runId, visual.stream],
  );
  useEffect(() => {
    onStreamLength(entry.id, stream?.length ?? 0);
  }, [entry.id, onStreamLength, stream?.length]);

  return (
    <PhaseLogCard
      phase={phaseWithRunnerModel(entry, live.canvas?.nodes)}
      expanded={expanded}
      stream={stream ?? entry.stream}
      streamLoading={live.isLoading}
      selectedNodeId={selectedNodeId}
      onSelectNode={onSelectNode}
      organizationId={organizationId}
      canvasId={entry.appId}
      onStop={onStop}
      onRerun={onRerun}
      runHref={splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase: entry })}
      actionBusy={actionBusy}
      onToggle={onToggle}
      onUsageOpenChange={setUsageOpen}
    />
  );
}

/**
 * Work-order popup from a line-board card. Description and a phase log.
 * The automation canvas lives on the full run page, not here.
 */
export function WorkOrderSplitRunPopup({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  fixture,
  onClose,
  fixed = false,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
  canUpdate = true,
}: Omit<WorkOrderSplitRunBodyProps, "footerActions"> & {
  onClose?: () => void;
  fixed?: boolean;
  onDispatch?: (model?: string) => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
  canUpdate?: boolean;
}) {
  const canPickDraftStartModel = useExperimentalFeature(organizationId).has(FEATURE_FACTORY_DRAFT_START_MODEL);
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const dismissCurrentPopup = useCurrentPopupDismiss(orderId, onClose);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture, dismissCurrentPopup);
  const popupData = useSplitRunPopupData({ organizationId, factoryId, orderId, fixture });
  const edits = useSplitRunWorkOrderEdits({
    organizationId,
    factoryId,
    orderId,
    canUpdate,
    title: fixture.title,
    description: popupData.sourceDescription,
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: fixture.footer.kind,
  });
  const initialTab = defaultSplitRunPopupTab(fixture);
  const [tab, setTab] = useState(initialTab);
  const [fullPage, setFullPage] = useState(false);
  const [draftModel, setDraftModel] = useState(DRAFT_START_MODEL_AUTO);
  const analysis = useAnalysisPlanningSession({
    organizationId,
    factoryId,
    workOrderId: orderId,
    enabled: fixture.footer.kind === "draft" && Boolean(organizationId && factoryId && orderId),
    canUpdate,
  });
  const draftStart = draftStartAction(
    fixture.footer.kind,
    onDispatch
      ? async (model) => {
          await analysis.endSession();
          await onDispatch(model);
        }
      : undefined,
    () => setTab("log"),
    draftModel,
  );
  const backToDraft = returnToBacklogAction(mutations.onBackToDraft, () => setTab("description"));
  const review = (
    <SplitRunReview
      footer={fixture.footer}
      organizationId={organizationId}
      factoryKey={factoryKey}
      orderNumber={orderNumber}
      canAct={canUpdate}
      onStart={draftStart}
      onArchive={mutations.onArchive}
      onReject={mutations.onReject}
      onBackToDraft={backToDraft}
      onStop={mutations.onStop}
      startBusy={isDispatching}
      actionBusy={footerActions.busy}
      startDisabled={!canDispatch}
      modelSelect={
        fixture.footer.kind === "draft" &&
        canPickDraftStartModel &&
        fixture.footer.actions.some((action) => action.kind === "start") ? (
          <DraftStartModelSelect
            organizationId={organizationId}
            factoryId={factoryId}
            lineName={fixture.lineName}
            value={draftModel}
            onChange={setDraftModel}
            disabled={isDispatching}
          />
        ) : undefined
      }
    />
  );

  return (
    <PopupShell
      testId="work-order-split-run"
      fixed={fixed}
      fullPage={fullPage}
      className={fullPage ? undefined : SPLIT_RUN_POPUP_DIALOG_CLASSNAME}
      onDismiss={onClose}
    >
      <PopupHeader
        title={edits.title}
        onClose={onClose}
        canEditTitle={edits.canEdit}
        titleBusy={edits.titleBusy}
        onTitleSave={(next) => void edits.saveTitle(next)}
        expanded={fullPage}
        onToggleExpanded={() => setFullPage((current) => !current)}
        actions={
          <CopyLinkButton
            url={popupWorkOrderUrl(organizationId, factoryKey, orderNumber, lineId)}
            className="flex h-6 w-6 items-center justify-center rounded-full hover:bg-slate-950/5 dark:hover:bg-white/10"
            iconClassName="h-4 w-4"
            testId="popup-work-order-copy-link-button"
          />
        }
      >
        <OwnerTimeCostRow fixture={{ ...fixture, owner: edits.owner }} assigneeIds={edits.assigneeIds} />
      </PopupHeader>
      <SplitRunPopupTabs
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
        onTabChange={setTab}
        canUpdate={canUpdate}
        footerActions={footerActions}
        resultFooter={tab === "description" ? review : undefined}
        analysis={fixture.footer.kind === "draft" ? analysis : undefined}
      />
      {tab !== "description" ? review : null}
    </PopupShell>
  );
}

function draftStartAction(
  kind: SplitRunFixture["footer"]["kind"],
  onDispatch: ((model?: string) => Promise<void>) | undefined,
  openAutomations: () => void,
  selectedModel: string,
) {
  if (kind !== "draft") {
    return undefined;
  }
  return async () => {
    await onDispatch?.(draftStartModelPayload(selectedModel));
    openAutomations();
  };
}

function returnToBacklogAction(
  onBackToDraft: (() => void | Promise<boolean | void>) | undefined,
  openDescription: () => void,
) {
  if (!onBackToDraft) {
    return undefined;
  }
  return async () => {
    const returned = await onBackToDraft();
    if (returned === false) {
      return;
    }
    openDescription();
  };
}

function SplitRunPopupTabs({
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
}: {
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
}) {
  const liveWorkOrder = useWorkOrder(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const [streamTick, setStreamTick] = useState("");
  const follow = useFollowLogScroll<HTMLOListElement>(runningSplitRunPhaseId(fixture.phases), streamTick, {
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
