import { useState } from "react";

import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";

import { ConceptARunOverlay } from "./ConceptARunOverlay";
import { ConceptBRunOverlay } from "./ConceptBRunOverlay";
import { ConceptCRunOverlay } from "./ConceptCRunOverlay";
import { RunOverlayBoardBackdrop } from "./runOverlayShared";
import { IMPLEMENT_RUN_OVERLAY } from "./workOrderRunOverlayMocks";

export type RunOverlayConcept = "a" | "b" | "c";

const CONCEPTS: { id: RunOverlayConcept; label: string; pattern: string }[] = [
  { id: "a", label: "A", pattern: "Pipeline run" },
  { id: "b", label: "B", pattern: "Phase inspector" },
  { id: "c", label: "C", pattern: "Live canvas" },
];

/**
 * Storybook playground: line board under a run overlay, with A / B / C
 * switcher. Production peek is unchanged until a concept is chosen.
 */
export function WorkOrderRunOverlayPlayground({ initialConcept = "a" }: { initialConcept?: RunOverlayConcept }) {
  const [concept, setConcept] = useState<RunOverlayConcept>(initialConcept);
  const caption = CONCEPTS.find((entry) => entry.id === concept)?.pattern ?? "";

  return (
    <div className="relative min-h-svh">
      <RunOverlayBoardBackdrop />
      {concept === "a" ? <ConceptARunOverlay fixture={IMPLEMENT_RUN_OVERLAY} /> : null}
      {concept === "b" ? <ConceptBRunOverlay fixture={IMPLEMENT_RUN_OVERLAY} /> : null}
      {concept === "c" ? <ConceptCRunOverlay fixture={IMPLEMENT_RUN_OVERLAY} /> : null}
      <div className="pointer-events-none absolute inset-x-0 top-0 z-20 flex justify-center p-3">
        <div className="pointer-events-auto flex items-center gap-2 rounded-full border border-border bg-background/95 px-2 py-1.5 shadow-sm">
          {CONCEPTS.map((entry) => (
            <Button
              key={entry.id}
              type="button"
              size="xs"
              variant={concept === entry.id ? "default" : "ghost"}
              className={cn(concept !== entry.id && "text-muted-foreground")}
              onClick={() => setConcept(entry.id)}
              aria-pressed={concept === entry.id}
            >
              {entry.label} · {entry.pattern}
            </Button>
          ))}
        </div>
      </div>
      <p className="sr-only">{caption}</p>
    </div>
  );
}
