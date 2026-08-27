import { useWorkOrderArtifacts } from "@/hooks/useFactoryData";
import { useEffect, useMemo, useState } from "react";

import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageResponses";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { OwnerTimeCostRow, PopupHeader, PopupShell } from "../work-order-popup-redesign/popupShared";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunHeaderActions } from "./SplitRunHeaderActions";
import { SplitRunFollowSwitch } from "./SplitRunLogHeader";
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
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  resolveSplitRunPopupArtifacts,
  splitRunDescriptionMarkdown,
  splitRunLogTabDotClass,
  splitRunPhaseAutomationHref,
  splitRunPhaseRunHref,
  splitRunSourceDescription,
} from "./splitRunPopupModel";
import { useSplitRunFooterActions, type SplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { runningSplitRunPhaseId } from "./followLogScroll";
import { useFollowLogScroll } from "./useFollowLogScroll";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { WorkOrderStatusDot } from "../../workOrders/WorkOrderStatusDot";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

function footerMutationHandlers(canUpdate: boolean, footerActions: SplitRunFooterActions, fixture: SplitRunFixture) {
  if (!canUpdate) {
    return { onStop: undefined };
  }
  return {
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

type SplitRunFollow = ReturnType<typeof useFollowLogScroll>;

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
    <div className="flex min-h-0 min-w-0 flex-1 flex-col" data-testid="split-run-log-pane">
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
      <SplitRunReview
        footer={fixture.footer}
        organizationId={organizationId}
        factoryKey={factoryKey}
        orderNumber={orderNumber}
      />
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
  const draftStart = draftStartAction(fixture.footer.kind, onDispatch);
  const fixtureArtifacts = collectSplitRunArtifacts(fixture);
  const useLiveArtifacts = hasLiveWorkOrder(organizationId, factoryId, orderId);
  const liveArtifactsQuery = useWorkOrderArtifacts(organizationId ?? "", factoryId ?? "", orderId ?? "");
  const artifacts = resolveSplitRunPopupArtifacts({
    fixtureArtifacts,
    liveArtifacts: liveArtifactsQuery.data,
    useLive: useLiveArtifacts,
  });
  const artifactDescription = splitRunDescriptionMarkdown(artifacts) || splitRunDescriptionMarkdown(fixtureArtifacts);
  const sourceDescription = splitRunSourceDescription({
    workOrderDescription: fixture.descriptionText,
    artifactDescription,
    preferWorkOrder: useLiveArtifacts,
  });
  const edits = useSplitRunWorkOrderEdits({
    organizationId,
    factoryId,
    orderId,
    canUpdate,
    title: fixture.title,
    description: sourceDescription,
    owner: fixture.owner,
    assigneeIds: fixture.assigneeIds ?? [],
    footerKind: fixture.footer.kind,
  });
  const initialTab = defaultSplitRunPopupTab(fixture);
  const ownerOrganizationId = popupOwnerOrganizationId(organizationId, edits.canEdit);
  const [fullPage, setFullPage] = useState(false);

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
          <SplitRunHeaderActions
            footer={fixture.footer}
            onStart={draftStart}
            onStop={mutations.onStop}
            startBusy={isDispatching}
            stopBusy={footerActions.busy}
            startDisabled={!canDispatch}
          />
        }
      >
        <OwnerTimeCostRow
          fixture={{ ...fixture, owner: edits.owner }}
          organizationId={ownerOrganizationId}
          canEditOwner={edits.canEdit}
          assigneeIds={edits.assigneeIds}
          ownerBusy={edits.ownerBusy}
          onOwnerSave={edits.saveOwner}
        />
      </PopupHeader>
      <SplitRunPopupTabs
        fixture={fixture}
        edits={edits}
        artifacts={artifacts}
        artifactsLoading={useLiveArtifacts && liveArtifactsQuery.isLoading}
        organizationId={organizationId}
        factoryId={factoryId}
        factoryKey={factoryKey}
        orderId={orderId}
        orderNumber={orderNumber}
        lineId={lineId}
        initialTab={initialTab}
        canUpdate={canUpdate}
        footerActions={footerActions}
      />
    </PopupShell>
  );
}

function draftStartAction(kind: SplitRunFixture["footer"]["kind"], onDispatch?: () => Promise<void>) {
  if (kind !== "draft") {
    return undefined;
  }
  return onDispatch;
}

function hasLiveWorkOrder(organizationId?: string, factoryId?: string, orderId?: string) {
  return Boolean(organizationId && factoryId && orderId);
}

function popupOwnerOrganizationId(organizationId: string | undefined, canEdit: boolean) {
  if (organizationId) {
    return organizationId;
  }
  return canEdit ? FACTORIES_ORGANIZATION_ID : undefined;
}

function SplitRunPopupTabs({
  fixture,
  edits,
  artifacts,
  artifactsLoading,
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  lineId,
  initialTab,
  canUpdate,
  footerActions,
}: {
  fixture: SplitRunFixture;
  edits: ReturnType<typeof useSplitRunWorkOrderEdits>;
  artifacts: ReturnType<typeof collectSplitRunArtifacts>;
  artifactsLoading: boolean;
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  lineId?: string;
  initialTab: string;
  canUpdate: boolean;
  footerActions: SplitRunFooterActions;
}) {
  const [tab, setTab] = useState(initialTab);
  const [streamTick, setStreamTick] = useState("");
  const follow = useFollowLogScroll(runningSplitRunPhaseId(fixture.phases), streamTick);

  return (
    <Tabs value={tab} onValueChange={setTab} className="flex min-h-0 min-w-0 flex-1 flex-col">
      <div className="flex shrink-0 items-center gap-2 border-b border-border px-5 py-2">
        <TabsList aria-label="Work order views">
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
        {tab === "log" ? (
          <SplitRunFollowSwitch following={follow.following} onFollowingChange={follow.setFollowing} />
        ) : null}
      </div>
      <TabsContent value="description" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
        <WorkOrderSplitRunOverview
          description={edits.description}
          artifacts={artifacts}
          checks={fixture.checks}
          artifactsLoading={artifactsLoading}
          organizationId={organizationId}
          factoryKey={factoryKey}
          orderNumber={orderNumber}
          expandFirstCheck={fixture.footer.kind === "draft"}
          canEditDescription={edits.canEditDescription}
          descriptionBusy={edits.descriptionBusy}
          onDescriptionSave={edits.saveDescription}
          source={fixture.source}
          footer={fixture.footer}
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
