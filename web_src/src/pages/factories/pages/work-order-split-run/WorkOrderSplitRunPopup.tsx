import { useState } from "react";

import { OwnerTimeCostRow, PopupHeader, PopupShell, SectionTitle } from "../work-order-popup-redesign/popupShared";
import { CompactLineCanvas } from "./CompactLineCanvas";
import { PhaseLogCard } from "./PhaseLogCard";
import { SplitRunCheckPills, SplitRunReview } from "./SplitRunReview";
import { richStreamForCanvas, splitRunCanvasForPhase, type SplitRunCanvasKey } from "./splitRunCanvases";
import { phaseById, type SplitRunFixture, type SplitRunPhaseId } from "./splitRunMocks";

/**
 * Split run popup. Left 60% is a terminal log. Right 40% is the canvas
 * for the selected log step. Storybook and the line board.
 */
export function WorkOrderSplitRunPopup({
  fixture,
  onClose,
  fixed = false,
  canvasEditHref,
  canvasExpandHref,
}: {
  fixture: SplitRunFixture;
  onClose?: () => void;
  fixed?: boolean;
  canvasEditHref?: (key: SplitRunCanvasKey) => string | undefined;
  canvasExpandHref?: (key: SplitRunCanvasKey) => string | undefined;
}) {
  const [phaseId, setPhaseId] = useState<SplitRunPhaseId>(fixture.currentPhaseId);
  const [openPhaseId, setOpenPhaseId] = useState<SplitRunPhaseId | null>(fixture.currentPhaseId);
  const [nodeId, setNodeId] = useState<string | null>(null);
  const selectedPhase = phaseById(fixture, phaseId);
  const selectedCanvas = splitRunCanvasForPhase(selectedPhase);

  return (
    <PopupShell testId="work-order-split-run" canvas fixed={fixed} onDismiss={onClose}>
      <PopupHeader title={fixture.title} onClose={onClose}>
        <OwnerTimeCostRow fixture={fixture}>
          <SplitRunCheckPills checks={fixture.checks} />
        </OwnerTimeCostRow>
      </PopupHeader>

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
                  stream={entry.id === openPhaseId ? richStreamForCanvas(splitRunCanvasForPhase(entry)) : undefined}
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
          <SplitRunReview notes={fixture.waitingNotes} tone={fixture.footerTone} />
        </aside>

        <section className="flex min-h-0 min-w-0 flex-col" aria-label="Run">
          <CompactLineCanvas
            key={selectedPhase.id}
            canvas={selectedCanvas}
            selectedId={nodeId}
            onSelect={setNodeId}
            editHref={canvasEditHref?.(selectedCanvas.key)}
            expandHref={canvasExpandHref?.(selectedCanvas.key)}
          />
        </section>
      </div>
    </PopupShell>
  );
}
