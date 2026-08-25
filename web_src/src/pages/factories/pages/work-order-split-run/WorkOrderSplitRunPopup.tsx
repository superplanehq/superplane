import { useMemo, useState } from "react";

import { OwnerTimeCostRow, PopupHeader, PopupShell, SectionTitle } from "../work-order-popup-redesign/popupShared";
import { CompactLineCanvas } from "./CompactLineCanvas";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunCheckPills, SplitRunReview } from "./SplitRunReview";
import { attachArtifactsToStream } from "./attachStreamArtifacts";
import { emptySplitRunCanvas, type SplitRunCanvasKey } from "./splitRunCanvases";
import { resolveSplitRunVisual } from "./splitRunLiveCanvas";
import { autoExpandedPhaseId, type SplitRunFixture, type SplitRunPhaseId } from "./splitRunMocks";
import { useSplitRunLiveCanvas } from "./useSplitRunLiveCanvas";
import { useSplitRunStreamArtifacts } from "./useSplitRunStreamArtifacts";

type WorkOrderSplitRunBodyProps = {
  organizationId?: string;
  factoryId?: string;
  orderId?: string;
  fixture: SplitRunFixture;
  canvasEditHref?: (key: SplitRunCanvasKey) => string | undefined;
  canvasExpandHref?: (key: SplitRunCanvasKey) => string | undefined;
  onDispatch?: () => Promise<void>;
  isDispatching?: boolean;
  canDispatch?: boolean;
};

/** Log and canvas for a split run. The popup wraps this. */
export function WorkOrderSplitRunBody({
  organizationId,
  factoryId,
  orderId,
  fixture,
  canvasEditHref,
  canvasExpandHref,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
}: WorkOrderSplitRunBodyProps) {
  const [phaseId, setPhaseId] = useState<SplitRunPhaseId>(fixture.currentPhaseId);
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(() => autoExpandedPhaseId(fixture));
  const [nodeId, setNodeId] = useState<string | null>(null);
  const selectedPhase = fixture.phases.find((entry) => entry.id === phaseId) ?? fixture.phases[0];
  const live = useSplitRunLiveCanvas(organizationId, selectedPhase);
  const artifactIndex = useSplitRunStreamArtifacts(organizationId, factoryId, orderId);
  const visual = useMemo(
    () =>
      selectedPhase ? resolveSplitRunVisual(selectedPhase, live) : { canvas: emptySplitRunCanvas(), stream: undefined },
    [live, selectedPhase],
  );
  const streams = useMemo(() => {
    const yamlOnly = { enabled: false, stream: [] };
    return new Map(
      fixture.phases.map((entry) => [
        entry.id,
        attachArtifactsToStream(
          entry.id === selectedPhase?.id ? visual.stream : resolveSplitRunVisual(entry, yamlOnly).stream,
          artifactIndex,
        ),
      ]),
    );
  }, [artifactIndex, fixture.phases, selectedPhase?.id, visual.stream]);

  return (
    <div className="grid min-h-0 flex-1 grid-cols-1 md:grid-cols-[minmax(0,3fr)_minmax(0,2fr)]">
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
          onStart={fixture.footer.kind === "draft" ? onDispatch : undefined}
          startBusy={isDispatching}
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
          expandHref={canvasExpandHref?.(visual.canvas.key)}
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
  orderId,
  fixture,
  onClose,
  fixed = false,
  canvasEditHref,
  canvasExpandHref,
  onDispatch,
  isDispatching = false,
  canDispatch = false,
}: WorkOrderSplitRunBodyProps & {
  onClose?: () => void;
  fixed?: boolean;
}) {
  return (
    <PopupShell testId="work-order-split-run" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={fixture.title} onClose={onClose}>
        <OwnerTimeCostRow fixture={fixture}>
          <SplitRunCheckPills checks={fixture.checks} />
        </OwnerTimeCostRow>
      </PopupHeader>
      <WorkOrderSplitRunBody
        organizationId={organizationId}
        factoryId={factoryId}
        orderId={orderId}
        fixture={fixture}
        canvasEditHref={canvasEditHref}
        canvasExpandHref={canvasExpandHref}
        onDispatch={onDispatch}
        isDispatching={isDispatching}
        canDispatch={canDispatch}
      />
    </PopupShell>
  );
}
