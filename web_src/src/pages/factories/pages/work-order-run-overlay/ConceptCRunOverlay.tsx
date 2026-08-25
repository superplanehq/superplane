import { Tabs, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useState } from "react";

import { CompactPhaseCanvas } from "./CompactPhaseCanvas";
import { RunAttachedList, RunCheckGrid, RunOverlayFrame, RunOverlayHeader, SectionLabel } from "./runOverlayShared";
import {
  artifactsForPhase,
  checksForPhase,
  nextPhaseId,
  phaseById,
  type RunOverlayFixture,
  type RunOverlayPhaseId,
} from "./workOrderRunOverlayMocks";

/**
 * Concept C — canvas-first run.
 *
 * Pattern: n8n execution view and the SuperPlane factory canvas. The modal
 * body is a compact React Flow graph of the selected phase. Check cards sit
 * as a score strip under the graph. Selecting a node shows what that step
 * attached.
 */
export function ConceptCRunOverlay({ fixture, onClose }: { fixture: RunOverlayFixture; onClose?: () => void }) {
  const [phaseId, setPhaseId] = useState<RunOverlayPhaseId>(fixture.currentPhaseId);
  const [stepId, setStepId] = useState<string | null>(null);
  const phase = phaseById(fixture, phaseId);
  const checks = checksForPhase(fixture, phase);
  const artifacts = artifactsForPhase(fixture, phase);
  const selectedStep = phase.steps.find((step) => step.id === stepId) ?? null;

  return (
    <RunOverlayFrame testId="run-overlay-concept-c" canvas>
      <RunOverlayHeader
        fixture={fixture}
        phase={phase}
        onClose={onClose}
        onContinue={() => {
          const next = nextPhaseId(phaseId);
          if (next) {
            setPhaseId(next);
            setStepId(null);
          }
        }}
        trailing={
          <Tabs
            value={phaseId}
            onValueChange={(value) => {
              setPhaseId(value as RunOverlayPhaseId);
              setStepId(null);
            }}
          >
            <TabsList aria-label="Run phases">
              {fixture.phases.map((entry) => (
                <TabsTrigger key={entry.id} value={entry.id}>
                  {entry.name}
                </TabsTrigger>
              ))}
            </TabsList>
          </Tabs>
        }
      />
      <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[minmax(0,1fr)_18rem]">
        <div className="relative min-h-0 min-w-0">
          <CompactPhaseCanvas steps={phase.steps} selectedId={stepId} onSelect={setStepId} />
        </div>
        <aside className="flex min-h-0 flex-col overflow-y-auto border-t border-border px-4 py-4 lg:border-t-0 lg:border-l">
          <p className="text-[13px] leading-relaxed text-muted-foreground">{phase.summary}</p>
          {selectedStep ? (
            <p className="mt-3 text-[13px] text-foreground">
              Selected: {selectedStep.title}
              {selectedStep.detail ? ` · ${selectedStep.detail}` : ""}
            </p>
          ) : (
            <p className="mt-3 text-[13px] text-muted-foreground">Select a node to inspect that step.</p>
          )}
          <section className="mt-4">
            <SectionLabel>Attached</SectionLabel>
            <RunAttachedList
              artifacts={artifacts}
              emptyLabel="This phase has not attached files or pull requests yet."
            />
          </section>
        </aside>
      </div>
      <div className="shrink-0 border-t border-border px-4 py-3">
        <SectionLabel>Checks</SectionLabel>
        <RunCheckGrid checks={checks} emptyLabel="This phase has not attached checks yet." />
      </div>
    </RunOverlayFrame>
  );
}
