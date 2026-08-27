import { cn } from "@/lib/utils";
import { useState } from "react";

import { PhaseGlyph } from "../linePhaseGlyph";
import { PhaseStepList } from "./CompactPipelineGraph";
import { RunAttachedList, RunCheckGrid, RunOverlayFrame, RunOverlayHeader, SectionLabel } from "./runOverlayShared";
import {
  artifactsForPhase,
  checksForPhase,
  nextPhaseId,
  overlayStatusGlyph,
  overlayStatusLabel,
  phaseById,
  type RunOverlayFixture,
  type RunOverlayPhaseId,
} from "./workOrderRunOverlayMocks";

/**
 * Concept B — phase inspector.
 *
 * Pattern: Vercel deployment detail and Railway deploy inspector. The left
 * rail is the run path. Select a phase to see the steps, checks, and
 * attachments that phase produced. No ticket description and no line picker.
 */
export function ConceptBRunOverlay({ fixture, onClose }: { fixture: RunOverlayFixture; onClose?: () => void }) {
  const [phaseId, setPhaseId] = useState<RunOverlayPhaseId>(fixture.currentPhaseId);
  const [stepId, setStepId] = useState<string | null>(null);
  const phase = phaseById(fixture, phaseId);
  const checks = checksForPhase(fixture, phase);
  const artifacts = artifactsForPhase(fixture, phase);
  const selectedStep =
    phase.steps.find((step) => step.id === stepId) ?? phase.steps.find((step) => step.status === "running") ?? null;

  return (
    <RunOverlayFrame testId="run-overlay-concept-b" wide>
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
      <div className="grid min-h-0 flex-1 grid-cols-1 lg:grid-cols-[16rem_minmax(0,1fr)]">
        <nav
          className="min-h-0 overflow-y-auto border-b border-border p-3 lg:border-r lg:border-b-0"
          aria-label="Run phases"
        >
          <ol className="space-y-1">
            {fixture.phases.map((entry) => {
              const selected = entry.id === phaseId;
              return (
                <li key={entry.id}>
                  <button
                    type="button"
                    onClick={() => {
                      setPhaseId(entry.id);
                      setStepId(null);
                    }}
                    className={cn(
                      "flex w-full items-start gap-2.5 rounded-lg px-3 py-2.5 text-left transition-colors",
                      selected ? "bg-accent/60" : "hover:bg-accent/30",
                    )}
                    aria-current={selected ? "step" : undefined}
                  >
                    <PhaseGlyph kind={overlayStatusGlyph(entry.status)} className="mt-0.5" />
                    <span className="min-w-0">
                      <span className="block text-[13px] font-medium text-foreground">{entry.name}</span>
                      <span className="block text-[12px] text-muted-foreground">
                        {overlayStatusLabel(entry.status)}
                        {entry.duration ? ` · ${entry.duration}` : ""}
                      </span>
                    </span>
                  </button>
                </li>
              );
            })}
          </ol>
        </nav>

        <div className="min-h-0 overflow-y-auto px-5 py-4">
          <p className="text-[13px] leading-relaxed text-muted-foreground">{phase.summary}</p>

          <section className="mt-5">
            <SectionLabel>Steps</SectionLabel>
            <PhaseStepList steps={phase.steps} selectedId={selectedStep?.id ?? null} onSelect={setStepId} />
          </section>

          <section className="mt-5">
            <SectionLabel>Checks from this phase</SectionLabel>
            <RunCheckGrid checks={checks} emptyLabel="This phase has not attached checks yet." />
          </section>

          <section className="mt-5">
            <SectionLabel>Attached in this phase</SectionLabel>
            <RunAttachedList
              artifacts={artifacts}
              emptyLabel="This phase has not attached files or pull requests yet."
            />
          </section>
        </div>
      </div>
    </RunOverlayFrame>
  );
}
