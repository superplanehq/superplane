import { useEffect, useMemo, useState } from "react";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { CopyLinkButton } from "../../CopyLinkButton";
import { workOrderDetailPath } from "../../lib/factoryPagePaths";
import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { JumpToLatestPill } from "./JumpToLatestPill";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunReview } from "./SplitRunReview";
import { attachArtifactsToStream } from "./attachStreamArtifacts";
import { emptySplitRunCanvas } from "./splitRunCanvases";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import {
  autoExpandedPhaseId,
  splitRunStatusLabel,
  type SplitRunFixture,
  type SplitRunPhase,
  type SplitRunPhaseId,
} from "./splitRunMocks";
import {
  defaultSplitRunPopupTab,
  type SplitRunPopupTab,
  splitRunLogTabDotClass,
  splitRunPhaseAutomationHref,
  splitRunPhaseRunHref,
} from "./splitRunPopupModel";
import { useSplitRunPopupData } from "./useSplitRunPopupData";
import { useSplitRunFooterActions, type SplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { useFollowLogScroll } from "./useFollowLogScroll";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { WorkOrderStatusDot } from "../../workOrders/WorkOrderStatusDot";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

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

function footerMutationHandlers(canUpdate: boolean, footerActions: SplitRunFooterActions, fixture: SplitRunFixture) {
  if (!canUpdate) {
    return {};
  }
  return {
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
  const [phaseId, setPhaseId] = useState<SplitRunPhaseId>(fixture.currentPhaseId);
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
  const currentPhaseId = fixture.currentPhaseId;
  const expandedPhaseId = autoExpandedPhaseId(fixture);
  useEffect(() => {
    setPhaseId(currentPhaseId);
    setOpenPhaseId(expandedPhaseId);
  }, [currentPhaseId, expandedPhaseId]);
  const selectedPhase = fixture.phases.find((entry) => entry.id === phaseId) ?? fixture.phases[0];
  const live = useSplitRunLiveCanvas(organizationId, selectedPhase);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const demoArtifacts = !organizationId;
  const visual = useMemo(
    () =>
      selectedPhase
        ? resolveSplitRunVisual(selectedPhase, live, { demoArtifacts })
        : { canvas: emptySplitRunCanvas(), stream: undefined },
    [demoArtifacts, live, selectedPhase],
  );
  const streams = useMemo(() => {
    const yamlOnly = { enabled: false, stream: [] };
    return new Map(
      fixture.phases.map((entry) => [
        entry.id,
        attachArtifactsToStream(
          entry.id === selectedPhase?.id
            ? visual.stream
            : resolveSplitRunVisual(entry, yamlOnly, { demoArtifacts }).stream,
          artifactIndex,
          entry.runId,
        ),
      ]),
    );
  }, [artifactIndex, demoArtifacts, fixture.phases, selectedPhase?.id, visual.stream]);
  const streamTick = useMemo(() => [...streams.values()].map((stream) => stream?.length ?? 0).join(":"), [streams]);
  useEffect(() => {
    onStreamTick(streamTick);
  }, [onStreamTick, streamTick]);
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
            <PhaseLogCard
              phase={entry}
              expanded={entry.id === openPhaseId}
              stream={streams.get(entry.id) ?? entry.stream}
              selectedNodeId={nodeId}
              onSelectNode={setNodeId}
              organizationId={organizationId}
              canvasId={entry.appId}
              onStop={automationStop(entry)}
              onRerun={automationRerun(entry)}
              runHref={splitRunPhaseRunHref({ organizationId, factoryKey, orderNumber, lineId, phase: entry })}
              editHref={splitRunPhaseAutomationHref({ organizationId, factoryKey, orderNumber, phase: entry })}
              actionBusy={footerActions.busy}
              onToggle={() => {
                setPhaseId(entry.id);
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
  onDispatch?: () => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
  canUpdate?: boolean;
}) {
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture);
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
  const draftStart = draftStartAction(fixture.footer.kind, onDispatch, () => setTab("log"));
  const backToDraft = returnToBacklogAction(mutations.onBackToDraft, () => setTab("description"));

  return (
    <PopupShell testId="work-order-split-run" fixed={fixed} fullPage={fullPage} onDismiss={onClose}>
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
        artifacts={popupData.artifacts}
        artifactsLoading={popupData.artifactsLoading}
        pullRequests={popupData.pullRequests}
        pullRequestsLoading={popupData.pullRequestsLoading}
        pullRequestsError={popupData.pullRequestsError}
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
      />
      <SplitRunReview
        footer={fixture.footer}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
        canAct={canUpdate}
        onStart={draftStart}
        onReject={mutations.onReject}
        onBackToDraft={backToDraft}
        onStop={mutations.onStop}
        startBusy={isDispatching}
        actionBusy={footerActions.busy}
        startDisabled={!canDispatch}
      />
    </PopupShell>
  );
}

function draftStartAction(
  kind: SplitRunFixture["footer"]["kind"],
  onDispatch: (() => Promise<void>) | undefined,
  openAutomations: () => void,
) {
  if (kind !== "draft") {
    return undefined;
  }
  return async () => {
    await onDispatch?.();
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
  artifacts,
  artifactsLoading,
  pullRequests,
  pullRequestsLoading,
  pullRequestsError,
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
}: {
  fixture: SplitRunFixture;
  edits: ReturnType<typeof useSplitRunWorkOrderEdits>;
  artifacts: ReturnType<typeof useSplitRunPopupData>["artifacts"];
  artifactsLoading: boolean;
  pullRequests: ReturnType<typeof useSplitRunPopupData>["pullRequests"];
  pullRequestsLoading: boolean;
  pullRequestsError: Error | null;
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
}) {
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
            <WorkOrderStatusDot
              colorClassName={splitRunLogTabDotClass(fixture.lineStatus)}
              pulsing={fixture.lineStatus === "running"}
              title={splitRunStatusLabel(fixture.lineStatus)}
              className="size-1.5"
              data-testid="split-run-log-tab-dot"
              aria-hidden
            />
            Automations
          </TabsTrigger>
        </TabsList>
      </div>
      <TabsContent value="description" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
        <WorkOrderSplitRunOverview
          description={edits.description}
          artifacts={artifacts}
          checks={fixture.checks}
          artifactsLoading={artifactsLoading}
          pullRequests={pullRequests}
          pullRequestsLoading={pullRequestsLoading}
          pullRequestsError={pullRequestsError}
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={orderNumber}
          expandFirstCheck={fixture.footer.kind === "draft"}
          canEditDescription={edits.canEditDescription}
          descriptionBusy={edits.descriptionBusy}
          onDescriptionSave={edits.saveDescription}
          source={fixture.source}
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
