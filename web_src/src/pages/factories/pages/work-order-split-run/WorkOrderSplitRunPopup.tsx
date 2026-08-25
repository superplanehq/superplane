import { useWorkOrderArtifacts } from "@/hooks/useFactoryData";
import { useMemo, useState } from "react";

import { FACTORIES_ORGANIZATION_ID } from "../../__fixtures__/factoryPageResponses";

import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";

import { OwnerTimeCostRow, PopupHeader, PopupShell, SectionTitle } from "../work-order-popup-redesign/popupShared";
import { CompactLineCanvas } from "./CompactLineCanvas";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunReview } from "./SplitRunReview";
import { attachArtifactsToStream } from "./attachStreamArtifacts";
import { emptySplitRunCanvas, type SplitRunCanvasKey } from "./splitRunCanvases";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import { autoExpandedPhaseId, splitRunStatusLabel, type SplitRunFixture, type SplitRunPhaseId } from "./splitRunMocks";
import {
  collectSplitRunArtifacts,
  defaultSplitRunPopupTab,
  resolveSplitRunPopupArtifacts,
  SPLIT_RUN_PANE_GRID_CLASSNAME,
  splitRunDescriptionMarkdown,
  splitRunLogTabDotClass,
  splitRunSourceDescription,
} from "./splitRunPopupModel";
import { useSplitRunFooterActions } from "./useSplitRunFooterActions";
import { useSplitRunWorkOrderEdits } from "./useSplitRunWorkOrderEdits";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";
import { WorkOrderStatusDot } from "../../workOrders/WorkOrderStatusDot";
import { WorkOrderSplitRunOverview } from "./WorkOrderSplitRunOverview";

function footerMutationHandlers(
  canUpdate: boolean,
  footerActions: ReturnType<typeof useSplitRunFooterActions>,
  footer: SplitRunFixture["footer"],
) {
  if (!canUpdate) {
    return { onStop: undefined, onReject: undefined };
  }
  return {
    onStop: (choice: Parameters<typeof footerActions.handleStop>[0]) => void footerActions.handleStop(choice, footer),
    onReject: footerActions.handleReject,
  };
}

type WorkOrderSplitRunBodyProps = {
  organizationId?: string;
  factoryId?: string;
  factoryKey?: string;
  orderId?: string;
  orderNumber?: string;
  fixture: SplitRunFixture;
  canvasEditHref?: (key: SplitRunCanvasKey) => string | undefined;
  canvasExpandHref?: (key: SplitRunCanvasKey, phase?: { appId?: string; runId?: string }) => string | undefined;
  onDispatch?: () => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
  canUpdate?: boolean;
};

/** Log and canvas for a split run. The popup wraps this. */
export function WorkOrderSplitRunBody({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  fixture,
  canvasEditHref,
  canvasExpandHref,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
  canUpdate = true,
}: WorkOrderSplitRunBodyProps) {
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture.footer);
  const [phaseId, setPhaseId] = useState<SplitRunPhaseId>(fixture.currentPhaseId);
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
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

  return (
    <div className={SPLIT_RUN_PANE_GRID_CLASSNAME} data-testid="split-run-log-pane">
      <aside className="flex min-h-0 flex-col border-b border-border bg-muted/25 md:border-r md:border-b-0">
        <div className="mb-2 px-3 pt-3">
          <SectionTitle>Log</SectionTitle>
        </div>

        <ol className="min-h-0 flex-1 overflow-y-auto px-3 pb-3">
          {fixture.phases.map((entry) => (
            <li key={entry.id} className="border-b border-border/70 py-1 last:border-b-0">
              <PhaseLogCard
                phase={entry}
                expanded={entry.id === openPhaseId}
                stream={streams.get(entry.id) ?? entry.stream}
                selectedNodeId={nodeId}
                onSelectNode={setNodeId}
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
          onStart={fixture.footer.kind === "draft" ? onDispatch : undefined}
          onStop={mutations.onStop}
          onReject={mutations.onReject}
          startBusy={isDispatching}
          stopBusy={footerActions.busy}
          startDisabled={!canDispatch}
        />
      </aside>

      <section className="flex min-h-0 min-w-0 flex-col" aria-label="Run">
        <CompactLineCanvas
          key={selectedPhase?.id ?? "empty"}
          canvas={visual.canvas}
          selectedId={nodeId}
          onSelect={setNodeId}
          editHref={canvasEditHref?.(visual.canvas.key)}
          expandHref={canvasExpandHref?.(visual.canvas.key, selectedPhase)}
        />
      </section>
    </div>
  );
}

/**
 * Split run popup. Left 60% is a terminal log. Right 40% is the canvas
 * for the selected log step. Storybook and the line board.
 */
export function WorkOrderSplitRunPopup({
  organizationId,
  factoryId,
  factoryKey,
  orderId,
  orderNumber,
  fixture,
  onClose,
  fixed = false,
  canvasEditHref,
  canvasExpandHref,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
  canUpdate = true,
}: WorkOrderSplitRunBodyProps & {
  onClose?: () => void;
  fixed?: boolean;
}) {
  const footerActions = useSplitRunFooterActions(organizationId, factoryId, orderId);
  const mutations = footerMutationHandlers(canUpdate, footerActions, fixture.footer);
  const fixtureArtifacts = collectSplitRunArtifacts(fixture);
  const useLiveArtifacts = Boolean(organizationId && factoryId && orderId);
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
  const ownerOrganizationId = organizationId ?? (edits.canEdit ? FACTORIES_ORGANIZATION_ID : undefined);

  return (
    <PopupShell testId="work-order-split-run" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader
        title={edits.title}
        onClose={onClose}
        canEditTitle={edits.canEdit}
        titleBusy={edits.titleBusy}
        onTitleSave={(next) => void edits.saveTitle(next)}
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
      <Tabs defaultValue={initialTab} className="flex min-h-0 flex-1 flex-col">
        <div className="shrink-0 border-b border-border px-5 py-2">
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
              Log
            </TabsTrigger>
          </TabsList>
        </div>
        <TabsContent value="description" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
          <WorkOrderSplitRunOverview
            description={edits.description}
            artifacts={artifacts}
            checks={fixture.checks}
            artifactsLoading={useLiveArtifacts && liveArtifactsQuery.isLoading}
            organizationId={organizationId}
            factoryKey={factoryKey}
            orderNumber={orderNumber}
            expandFirstCheck={fixture.footer.kind === "draft"}
            canEditDescription={edits.canEditDescription}
            descriptionBusy={edits.descriptionBusy}
            onDescriptionSave={edits.saveDescription}
            source={fixture.source}
            footer={fixture.footer}
            onStart={fixture.footer.kind === "draft" ? onDispatch : undefined}
            onStop={mutations.onStop}
            onReject={mutations.onReject}
            startBusy={isDispatching}
            stopBusy={footerActions.busy}
            startDisabled={!canDispatch}
          />
        </TabsContent>
        <TabsContent value="log" className="mt-0 flex min-h-0 flex-1 flex-col overflow-hidden">
          <WorkOrderSplitRunBody
            organizationId={organizationId}
            factoryId={factoryId}
            factoryKey={factoryKey}
            orderId={orderId}
            orderNumber={orderNumber}
            fixture={fixture}
            canvasEditHref={canvasEditHref}
            canvasExpandHref={canvasExpandHref}
            onDispatch={onDispatch}
            isDispatching={isDispatching}
            canDispatch={canDispatch}
            canUpdate={canUpdate}
          />
        </TabsContent>
      </Tabs>
    </PopupShell>
  );
}
