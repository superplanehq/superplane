import { useState } from "react";

import {
  HorizontalPhaseStepper,
  RunAttachedList,
  RunCheckGrid,
  RunOverlayFrame,
  RunOverlayHeader,
  SectionLabel,
} from "./runOverlayShared";
import { CompactPipelineGraph } from "./CompactPipelineGraph";
import {
  artifactsForPhase,
  checksForPhase,
  nextPhaseId,
  phaseById,
  type RunOverlayFixture,
  type RunOverlayPhaseId,
} from "./workOrderRunOverlayMocks";

/**
 * Concept A — pipeline run.
 *
 * Pattern: GitHub Actions run summary and CircleCI pipeline. Phases are a
 * connected stepper. The current phase shows a compact job graph. Check
 * cards sit as the score strip. Factory Lines and ticket fields are absent.
 */
export function ConceptARunOverlay({ fixture, onClose }: { fixture: RunOverlayFixture; onClose?: () => void }) {
  const [phaseId, setPhaseId] = useState<RunOverlayPhaseId>(fixture.currentPhaseId);
  const [stepId, setStepId] = useState<string | null>(null);
  const phase = phaseById(fixture, phaseId);
  const checks = checksForPhase(fixture, phase);
  const artifacts = artifactsForPhase(fixture, phase);

  return (
    <RunOverlayFrame testId="run-overlay-concept-a">
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
      />
      <div className="min-h-0 flex-1 overflow-y-auto px-5 py-4">
        <HorizontalPhaseStepper
          fixture={fixture}
          selectedId={phaseId}
          onSelect={(id) => {
            setPhaseId(id);
            setStepId(null);
          }}
        />

        <p className="mt-4 text-[13px] leading-relaxed text-muted-foreground">{phase.summary}</p>

        <section className="mt-5">
          <SectionLabel>Checks</SectionLabel>
          <RunCheckGrid checks={checks} emptyLabel="This phase has not attached checks yet." />
        </section>

        <section className="mt-5">
          <SectionLabel>This phase</SectionLabel>
          <CompactPipelineGraph steps={phase.steps} selectedId={stepId} onSelect={setStepId} />
        </section>

        <section className="mt-5">
          <SectionLabel>Attached</SectionLabel>
          <RunAttachedList artifacts={artifacts} emptyLabel="This phase has not attached files or pull requests yet." />
        </section>
      </div>
    </RunOverlayFrame>
  );
}
